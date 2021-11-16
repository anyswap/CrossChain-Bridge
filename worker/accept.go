package worker

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
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
)

// StartAcceptSignJob accept job
func StartAcceptSignJob() {
	if !params.IsDcrmEnabled() {
		logWorker("accept", "no need to start accept sign job as dcrm is disabled")
		return
	}
	getAcceptListInterval := params.GetOracleConfig().GetAcceptListInterval
	if getAcceptListInterval > 0 {
		waitInterval = time.Duration(getAcceptListInterval) * time.Second
	}
	acceptSignStarter.Do(func() {
		logWorker("accept", "start accept sign job")
		openLeveldb()
		go startAcceptProducer()

		utils.TopWaitGroup.Add(1)
		go startAcceptConsumer()
	})
}

func startAcceptProducer() {
	i := 0
	for {
		signInfo, err := dcrm.GetCurNodeSignInfo(maxAcceptSignTimeInterval)
		if err != nil {
			logWorkerError("accept", "getCurNodeSignInfo failed", err)
			time.Sleep(retryInterval)
			continue
		}
		i++
		if i%7 == 0 {
			logWorker("accept", "getCurNodeSignInfo", "count", len(signInfo))
		}
		for _, info := range signInfo {
			if utils.IsCleanuping() {
				return
			}
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
		if utils.IsCleanuping() {
			return
		}
		time.Sleep(waitInterval)
	}
}

func startAcceptConsumer() {
	defer func() {
		closeLeveldb()
		utils.TopWaitGroup.Done()
	}()
	for {
		select {
		case <-utils.CleanupChan:
			logWorker("accept", "stop accept sign job")
			return
		case info := <-acceptInfoCh: // consume
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
	case params.GetReplaceIdentifier():
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
	if lvldbHandle != nil && args.GetTxNonce() > 0 { // only for eth like chain
		err = CheckAcceptRecord(args)
		if err != nil {
			return args, err
		}
	}
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
	if lvldbHandle != nil && args.GetTxNonce() > 0 { // only for eth like chain
		go saveAcceptRecord(dstBridge, keyID, buildTxArgs, rawTx)
	}
	logWorker("accept", "verify message hash success", ctx...)
	return nil
}

func saveAcceptRecord(bridge tokens.CrossChainBridge, keyID string, args *tokens.BuildTxArgs, rawTx interface{}) {
	impl, ok := bridge.(interface {
		GetSignedTxHashOfKeyID(keyID, pairID string, rawTx interface{}) (txHash string, err error)
	})
	if !ok {
		return
	}

	ctx := []interface{}{
		"keyID", keyID,
		"identifier", args.Identifier,
		"swaptype", args.SwapType.String(),
		"pairID", args.PairID,
		"swapID", args.SwapID,
		"bind", args.Bind,
	}

	swapTx, err := impl.GetSignedTxHashOfKeyID(keyID, args.PairID, rawTx)
	if err != nil {
		logWorkerError("accept", "get signed tx hash failed", err, ctx...)
		return
	}
	ctx = append(ctx, "swaptx", swapTx)

	err = AddAcceptRecord(args, swapTx)
	if err != nil {
		logWorkerError("accept", "save accept record to db failed", err, ctx...)
		return
	}
	logWorker("accept", "save accept record to db sucess", ctx...)
}
