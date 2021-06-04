package worker

import (
	"container/ring"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/dcrm"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc"
)

var (
	acceptSignStarter sync.Once

	acceptRing        *ring.Ring
	acceptRingLock    sync.RWMutex
	acceptRingMaxSize = 500

	retryInterval = 1 * time.Second
	waitInterval  = 1 * time.Second

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
		utils.TopWaitGroup.Add(1)
		go acceptSign()
	})
}

func acceptSign() {
	logWorker("accept", "start accept sign job")
	openLeveldb()
	defer func() {
		logWorker("accept", "stop accept sign job")
		closeLeveldb()
		utils.TopWaitGroup.Done()
	}()
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
			history := getAcceptSignHistory(keyID)
			if history != nil {
				if history.result != "IGNORE" {
					logWorkerTrace("accept", "quick process history accept", "keyID", keyID, "result", history.result)
					_, _ = dcrm.DoAcceptSign(keyID, history.result, history.msgHash, history.msgContext)
				}
				continue
			}
			agreeResult := "AGREE"
			args, err := getBuildTxArgsFromMsgContext(info)
			if err == nil {
				err = verifySignInfo(info, args)
			}
			switch {
			case errors.Is(err, tokens.ErrTxNotStable),
				errors.Is(err, tokens.ErrTxNotFound),
				errors.Is(err, tokens.ErrRPCQueryError):
				logWorkerTrace("accept", "ignore sign", "keyID", keyID, "err", err)
				continue
			case errors.Is(err, errIdentifierMismatch),
				errors.Is(err, errInitiatorMismatch),
				errors.Is(err, errWrongMsgContext),
				errors.Is(err, tokens.ErrUnknownPairID),
				errors.Is(err, tokens.ErrNoBtcBridge):
				addAcceptSignHistory(keyID, "IGNORE", info.MsgHash, info.MsgContext)
				logWorkerTrace("accept", "ignore sign", "keyID", keyID, "err", err)
				continue
			}
			if err != nil {
				logWorkerError("accept", "DISAGREE sign", err, "keyID", keyID, "pairID", args.PairID, "txid", args.SwapID, "bind", args.Bind, "swaptype", args.SwapType)
				agreeResult = "DISAGREE"
			}
			res, err := dcrm.DoAcceptSign(keyID, agreeResult, info.MsgHash, info.MsgContext)
			if err != nil {
				logWorkerError("accept", "accept sign job failed", err, "keyID", keyID, "result", res, "pairID", args.PairID, "txid", args.SwapID, "bind", args.Bind, "swaptype", args.SwapType)
			} else {
				logWorker("accept", "accept sign job finish", "keyID", keyID, "result", agreeResult, "pairID", args.PairID, "txid", args.SwapID, "bind", args.Bind, "swaptype", args.SwapType)
				addAcceptSignHistory(keyID, agreeResult, info.MsgHash, info.MsgContext)
				_ = AddAcceptRecord(args)
			}
		}
		time.Sleep(waitInterval)
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

type acceptSignInfo struct {
	keyID      string
	result     string
	msgHash    []string
	msgContext []string
}

func addAcceptSignHistory(keyID, result string, msgHash, msgContext []string) {
	// Create the new item as its own ring
	item := ring.New(1)
	item.Value = &acceptSignInfo{
		keyID:      keyID,
		result:     result,
		msgHash:    msgHash,
		msgContext: msgContext,
	}

	acceptRingLock.Lock()
	defer acceptRingLock.Unlock()

	if acceptRing == nil {
		acceptRing = item
	} else {
		if acceptRing.Len() == acceptRingMaxSize {
			// Drop the block out of the ring
			acceptRing = acceptRing.Move(-1)
			acceptRing.Unlink(1)
			acceptRing = acceptRing.Move(1)
		}
		acceptRing.Move(-1).Link(item)
	}
}

func getAcceptSignHistory(keyID string) *acceptSignInfo {
	acceptRingLock.RLock()
	defer acceptRingLock.RUnlock()

	if acceptRing == nil {
		return nil
	}

	r := acceptRing
	for i := 0; i < r.Len(); i++ {
		item := r.Value.(*acceptSignInfo)
		if item.keyID == keyID {
			return item
		}
		r = r.Prev()
	}

	return nil
}
