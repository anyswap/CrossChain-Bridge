package worker

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/dcrm"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc"
	mapset "github.com/deckarep/golang-set"
)

const (
	acceptAgree    = "AGREE"
	acceptDisagree = "DISAGREE"
)

var (
	acceptSignStarter sync.Once

	cachedAcceptInfos    = mapset.NewSet()
	maxCachedAcceptInfos = 500

	maxAcceptSignTimeInterval = int64(600) // seconds

	retryInterval = 3 * time.Second
	waitInterval  = 20 * time.Second

	acceptInfoCh      = make(chan *dcrm.SignInfoData, 10)
	maxAcceptRoutines = int64(10)
	curAcceptRoutines = int64(0)

	// those errors will be ignored in accepting
	errIdentifierMismatch = errors.New("cross chain bridge identifier mismatch")
	errInitiatorMismatch  = errors.New("initiator mismatch")
	errWrongMsgContext    = errors.New("wrong msg context")
	errInvalidSignInfo    = errors.New("invalid sign info")
	errExpiredSignInfo    = errors.New("expired sign info")
)

// StartAcceptSignJob accept job
func StartAcceptSignJob() {
	if !params.IsDcrmEnabled() {
		logWorker("accept", "no need to start accept sign job as dcrm is disabled")
		return
	}
	acceptSignStarter.Do(func() {
		logWorker("accept", "start accept sign job")
		go startAcceptProducer()
		go startAcceptConsumer()
	})
}

func startAcceptProducer() {
	for {
		signInfo, err := dcrm.GetCurNodeSignInfo()
		if err != nil {
			logWorkerError("accept", "getCurNodeSignInfo failed", err)
			time.Sleep(retryInterval)
			continue
		}
		logWorker("accept", "getCurNodeSignInfo", "count", len(signInfo))
		for _, info := range signInfo {
			if info == nil { // maybe a dcrm RPC problem
				continue
			}
			keyID := info.Key
			if cachedAcceptInfos.Contains(keyID) {
				logWorkerTrace("accept", "ignore cached accept sign info before dispatch", "keyID", keyID)
				continue
			}
			logWorker("accept", "dispatch accept sign info", "keyID", keyID)
			acceptInfoCh <- info // produce
		}
		time.Sleep(waitInterval)
	}
}

func startAcceptConsumer() {
	for {
		info := <-acceptInfoCh // consume
		// loop and check, break if free worker exist
		for {
			if atomic.LoadInt64(&curAcceptRoutines) < maxAcceptRoutines {
				break
			}
			time.Sleep(1 * time.Second)
		}

		atomic.AddInt64(&curAcceptRoutines, 1)
		go processAcceptInfo(info)
	}
}

func checkAndUpdateCachedAcceptInfoMap(keyID string) (ok bool) {
	if cachedAcceptInfos.Contains(keyID) {
		logWorkerTrace("accept", "ignore cached accept sign info in process", "keyID", keyID)
		return false
	}
	if cachedAcceptInfos.Cardinality() >= maxCachedAcceptInfos {
		cachedAcceptInfos.Pop()
	}
	cachedAcceptInfos.Add(keyID)
	return true
}

func processAcceptInfo(info *dcrm.SignInfoData) {
	defer atomic.AddInt64(&curAcceptRoutines, -1)

	keyID := info.Key
	if !checkAndUpdateCachedAcceptInfoMap(keyID) {
		return
	}
	isProcessed := false
	defer func() {
		if !isProcessed {
			cachedAcceptInfos.Remove(keyID)
		}
	}()

	args, err := verifySignInfo(info)

	ctx := []interface{}{
		"keyID", keyID,
	}
	if args != nil {
		ctx = append(ctx,
			"identifier", args.Identifier,
			"swaptype", args.SwapType.String(),
			"pairID", args.PairID,
			"swapID", args.SwapID,
			"bind", args.Bind,
		)
	}

	switch {
	case errors.Is(err, tokens.ErrTxNotStable),
		errors.Is(err, tokens.ErrTxNotFound),
		errors.Is(err, tokens.ErrRPCQueryError):
		ctx = append(ctx, "err", err)
		logWorkerTrace("accept", "ignore sign", ctx...)
		return
	case errors.Is(err, errIdentifierMismatch):
		ctx = append(ctx, "err", err)
		logWorkerTrace("accept", "discard sign", ctx...)
		isProcessed = true
		return
	case errors.Is(err, errInitiatorMismatch),
		errors.Is(err, errWrongMsgContext),
		errors.Is(err, errExpiredSignInfo),
		errors.Is(err, errInvalidSignInfo),
		errors.Is(err, tokens.ErrUnknownPairID),
		errors.Is(err, tokens.ErrNoBtcBridge):
		ctx = append(ctx, "err", err)
		logWorker("accept", "discard sign", ctx...)
		isProcessed = true
		return
	}

	agreeResult := acceptAgree
	if err != nil {
		logWorkerError("accept", "DISAGREE sign", err, ctx...)
		agreeResult = acceptDisagree
	}
	ctx = append(ctx, "result", agreeResult)

	res, err := dcrm.DoAcceptSign(keyID, agreeResult, info.MsgHash, info.MsgContext)
	if err != nil {
		ctx = append(ctx, "rpcResult", res)
		logWorkerError("accept", "accept sign job failed", err, ctx...)
	} else {
		logWorker("accept", "accept sign job finish", ctx...)
		isProcessed = true
	}
}

func getBuildTxArgsFromMsgContext(signInfo *dcrm.SignInfoData) (*tokens.BuildTxArgs, error) {
	msgContext := signInfo.MsgContext
	if len(msgContext) != 1 {
		return nil, errWrongMsgContext
	}
	var args tokens.BuildTxArgs
	err := json.Unmarshal([]byte(msgContext[0]), &args)
	if err != nil {
		return nil, errWrongMsgContext
	}
	return &args, nil
}

func verifySignInfo(signInfo *dcrm.SignInfoData) (args *tokens.BuildTxArgs, err error) {
	timestamp, err := common.GetUint64FromStr(signInfo.TimeStamp)
	if err != nil || int64(timestamp/1000)+maxAcceptSignTimeInterval < time.Now().Unix() {
		logWorkerTrace("accept", "expired accept sign info", "signInfo", signInfo)
		return nil, errExpiredSignInfo
	}
	if signInfo.Key == "" || signInfo.Account == "" || signInfo.GroupID == "" {
		logWorkerWarn("accept", "invalid accept sign info", "signInfo", signInfo)
		return nil, errInvalidSignInfo
	}
	if !params.IsDcrmInitiator(signInfo.Account) {
		return nil, errInitiatorMismatch
	}
	args, err = getBuildTxArgsFromMsgContext(signInfo)
	if err != nil {
		return args, err
	}
	msgHash := signInfo.MsgHash
	msgContext := signInfo.MsgContext
	switch args.Identifier {
	case params.GetIdentifier():
	case tokens.AggregateIdentifier:
		if btc.BridgeInstance == nil {
			return args, tokens.ErrNoBtcBridge
		}
		logWorker("accept", "verifySignInfo", "msgHash", msgHash, "msgContext", msgContext)
		err = btc.BridgeInstance.VerifyAggregateMsgHash(msgHash, args)
		if err != nil {
			return args, err
		}
		return args, nil
	default:
		return args, errIdentifierMismatch
	}
	logWorker("accept", "verifySignInfo", "keyID", signInfo.Key, "msgHash", msgHash, "msgContext", msgContext)
	err = rebuildAndVerifyMsgHash(signInfo.Key, msgHash, args)
	if err != nil {
		return args, err
	}
	return args, nil
}

func rebuildAndVerifyMsgHash(keyID string, msgHash []string, args *tokens.BuildTxArgs) error {
	var srcBridge, dstBridge tokens.CrossChainBridge
	switch args.SwapType {
	case tokens.SwapinType:
		srcBridge = tokens.SrcBridge
		dstBridge = tokens.DstBridge
	case tokens.SwapoutType:
		srcBridge = tokens.DstBridge
		dstBridge = tokens.SrcBridge
	default:
		return fmt.Errorf("unknown swap type %v", args.SwapType)
	}

	tokenCfg := dstBridge.GetTokenConfig(args.PairID)
	if tokenCfg == nil {
		return tokens.ErrUnknownPairID
	}

	ctx := []interface{}{
		"keyID", keyID,
		"identifier", args.Identifier,
		"swaptype", args.SwapType.String(),
		"pairID", args.PairID,
		"swapID", args.SwapID,
		"bind", args.Bind,
	}

	swapInfo, err := verifySwapTransaction(srcBridge, args.PairID, args.SwapID, args.Bind, args.TxType)
	if err != nil {
		logWorkerError("accept", "verifySignInfo failed", err, ctx...)
		return err
	}

	buildTxArgs := &tokens.BuildTxArgs{
		SwapInfo:    args.SwapInfo,
		From:        tokenCfg.DcrmAddress,
		OriginValue: swapInfo.Value,
		Extra:       args.Extra,
	}
	rawTx, err := dstBridge.BuildRawTransaction(buildTxArgs)
	if err != nil {
		logWorkerError("accept", "build raw tx failed", err, ctx...)
		return err
	}
	err = dstBridge.VerifyMsgHash(rawTx, msgHash)
	if err != nil {
		logWorkerError("accept", "verify message hash failed", err, ctx...)
		return err
	}
	logWorker("accept", "verify message hash success", ctx...)
	return nil
}
