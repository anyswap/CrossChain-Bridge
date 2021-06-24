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

	retryInterval = 1 * time.Second
	waitInterval  = 3 * time.Second

	acceptInfoCh      = make(chan *dcrm.SignInfoData, 10)
	maxAcceptRoutines = int64(10)
	curAcceptRoutines = int64(0)

	// those errors will be ignored in accepting
	errIdentifierMismatch = errors.New("cross chain bridge identifier mismatch")
	errInitiatorMismatch  = errors.New("initiator mismatch")
	errWrongMsgContext    = errors.New("wrong msg context")
	errNonceMismatch      = errors.New("nonce mismatch")
)

// StartAcceptSignJob accept job
func StartAcceptSignJob() {
	if !params.IsDcrmEnabled() {
		logWorker("accept", "no need to start accept sign job as dcrm is disabled")
		return
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
		signInfo, err := dcrm.GetCurNodeSignInfo()
		if err != nil {
			logWorkerError("accept", "getCurNodeSignInfo failed", err)
			time.Sleep(retryInterval)
			continue
		}
		i++
		if i%20 == 0 {
			logWorker("accept", "getCurNodeSignInfo", "count", len(signInfo))
		}
		for _, info := range signInfo {
			if utils.IsCleanuping() {
				return
			}
			keyID := info.Key
			if keyID == "" || info.Account == "" || info.GroupID == "" {
				logWorkerWarn("accept", "invalid accept sign info", "signInfo", info)
				continue
			}
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

	agreeResult := acceptAgree
	args, err := getBuildTxArgsFromMsgContext(info)
	if err == nil {
		err = verifySignInfo(info, args)
	}
	switch {
	case errors.Is(err, tokens.ErrTxNotStable),
		errors.Is(err, tokens.ErrTxNotFound),
		errors.Is(err, tokens.ErrRPCQueryError):
		logWorkerTrace("accept", "ignore sign", "keyID", keyID, "err", err)
		return
	case errors.Is(err, errIdentifierMismatch),
		errors.Is(err, errInitiatorMismatch),
		errors.Is(err, errWrongMsgContext),
		errors.Is(err, tokens.ErrUnknownPairID),
		errors.Is(err, tokens.ErrNoBtcBridge):
		logWorker("accept", "ignore sign", "keyID", keyID, "err", err)
		isProcessed = true
		return
	}
	if err != nil {
		logWorkerError("accept", "DISAGREE sign", err, "keyID", keyID, "pairID", args.PairID, "txid", args.SwapID, "bind", args.Bind, "swaptype", args.SwapType)
		agreeResult = acceptDisagree
	}
	res, err := dcrm.DoAcceptSign(keyID, agreeResult, info.MsgHash, info.MsgContext)
	if err != nil {
		logWorkerError("accept", "accept sign job failed", err, "keyID", keyID, "result", res, "pairID", args.PairID, "txid", args.SwapID, "bind", args.Bind, "swaptype", args.SwapType)
	} else {
		logWorker("accept", "accept sign job finish", "keyID", keyID, "result", agreeResult, "pairID", args.PairID, "txid", args.SwapID, "bind", args.Bind, "swaptype", args.SwapType)
		if agreeResult == acceptAgree { // only record agree result
			saveAcceptRecord(keyID, args)
		}
		isProcessed = true
	}
}

func saveAcceptRecord(keyID string, args *tokens.BuildTxArgs) {
	var err error
	for i := 0; i < 3; i++ {
		err = AddAcceptRecord(args)
		if err == nil {
			return
		}
	}
	logWorkerWarn("accept", "save accept record to db failed", err,
		"keyID", keyID, "pairID", args.PairID, "txid", args.SwapID,
		"bind", args.Bind, "swaptype", args.SwapType.String())
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

func verifySignInfo(signInfo *dcrm.SignInfoData, args *tokens.BuildTxArgs) error {
	if !params.IsDcrmInitiator(signInfo.Account) {
		return errInitiatorMismatch
	}
	msgHash := signInfo.MsgHash
	msgContext := signInfo.MsgContext
	switch args.Identifier {
	case params.GetIdentifier():
	case params.GetReplaceIdentifier():
	case tokens.AggregateIdentifier:
		if btc.BridgeInstance == nil {
			return tokens.ErrNoBtcBridge
		}
		logWorker("accept", "verifySignInfo", "msgHash", msgHash, "msgContext", msgContext)
		return btc.BridgeInstance.VerifyAggregateMsgHash(msgHash, args)
	default:
		return errIdentifierMismatch
	}
	logWorker("accept", "verifySignInfo", "keyID", signInfo.Key, "msgHash", msgHash, "msgContext", msgContext)
	err := CheckAcceptRecord(args)
	if err != nil {
		return err
	}
	return rebuildAndVerifyMsgHash(msgHash, args)
}

func rebuildAndVerifyMsgHash(msgHash []string, args *tokens.BuildTxArgs) error {
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

	swapInfo, err := verifySwapTransaction(srcBridge, args.PairID, args.SwapID, args.Bind, args.TxType)
	if err != nil {
		logWorkerError("accept", "verifySignInfo failed", err, "pairID", args.PairID, "txid", args.SwapID, "bind", args.Bind, "swaptype", args.SwapType)
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
		return err
	}
	return dstBridge.VerifyMsgHash(rawTx, msgHash)
}
