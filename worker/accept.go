package worker

import (
	"container/ring"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/dcrm"
	"github.com/fsn-dev/crossChain-Bridge/params"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	"github.com/fsn-dev/crossChain-Bridge/tokens/btc"
)

var (
	acceptSignStarter sync.Once

	acceptRing        *ring.Ring
	acceptRingLock    sync.RWMutex
	acceptRingMaxSize = 500

	retryInterval = 3 * time.Second
	waitInterval  = 20 * time.Second

	// those errors will be ignored in accepting
	errIdentifierMismatch = errors.New("cross chain bridge identifier mismatch")
	errInitiatorMismatch  = errors.New("initiator mismatch")
)

// StartAcceptSignJob accept job
func StartAcceptSignJob() {
	acceptSignStarter.Do(func() {
		logWorker("accept", "start accept sign job")
		acceptSign()
	})
}

func acceptSign() {
	for {
		signInfo, err := dcrm.GetCurNodeSignInfo()
		if err != nil {
			time.Sleep(retryInterval)
			continue
		}
		logWorker("accept", "acceptSign", "count", len(signInfo))
		for _, info := range signInfo {
			keyID := info.Key
			history := getAcceptSignHistory(keyID)
			if history != nil {
				logWorker("accept", "ignore accepted sign", "keyID", keyID, "result", history.result)
				_, _ = dcrm.DoAcceptSign(keyID, history.result)
				continue
			}
			agreeResult := "AGREE"
			err := verifySignInfo(info)
			switch err {
			case errIdentifierMismatch,
				errInitiatorMismatch,
				tokens.ErrTxNotStable,
				tokens.ErrTxNotFound:
				logWorkerTrace("accept", "ignore sign info", "keyID", keyID, "err", err)
				continue
			}
			if err != nil {
				logWorkerError("accept", "disagree sign", err, "keyID", keyID)
				agreeResult = "DISAGREE"
			}
			logWorker("accept", "dcrm DoAcceptSign", "keyID", keyID, "result", agreeResult)
			res, err := dcrm.DoAcceptSign(keyID, agreeResult)
			if err != nil {
				logWorkerError("accept", "accept sign job failed", err, "keyID", keyID, "result", res)
			} else {
				logWorker("accept", "accept sign job finish", "keyID", keyID, "result", agreeResult)
				addAcceptSignHistory(keyID, agreeResult)
			}
		}
		time.Sleep(waitInterval)
	}
}

func verifySignInfo(signInfo *dcrm.SignInfoData) error {
	if common.HexToAddress(signInfo.Account) != common.HexToAddress(params.GetServerDcrmUser()) {
		return errInitiatorMismatch
	}
	msgHash := signInfo.MsgHash
	msgContext := signInfo.MsgContext
	logWorker("accept", "verifySignInfo", "msgHash", msgHash, "msgContext", msgContext)
	var args tokens.BuildTxArgs
	err := json.Unmarshal([]byte(msgContext), &args)
	if err != nil {
		return errIdentifierMismatch
	}
	switch args.Identifier {
	case params.GetIdentifier():
	case btc.AggregateIdentifier:
		return btc.BridgeInstance.VerifyAggregateMsgHash(msgHash, &args)
	default:
		return errIdentifierMismatch
	}

	var (
		srcBridge, dstBridge tokens.CrossChainBridge
		memo                 string
	)
	switch args.SwapType {
	case tokens.SwapinType:
		srcBridge = tokens.SrcBridge
		dstBridge = tokens.DstBridge
	case tokens.SwapoutType:
		srcBridge = tokens.DstBridge
		dstBridge = tokens.SrcBridge
		memo = fmt.Sprintf("%s%s", tokens.UnlockMemoPrefix, args.SwapID)
	case tokens.SwapRecallType:
		srcBridge = tokens.SrcBridge
		dstBridge = tokens.SrcBridge
		memo = fmt.Sprintf("%s%s", tokens.RecallMemoPrefix, args.SwapID)
	default:
		return fmt.Errorf("unknown swap type %v", args.SwapType)
	}
	var swap *tokens.TxSwapInfo
	switch args.TxType {
	case tokens.P2shSwapinTx:
		if btc.BridgeInstance == nil {
			return tokens.ErrWrongP2shSwapin
		}
		swap, err = btc.BridgeInstance.VerifyP2shTransaction(args.SwapID, args.Bind, false)
	default:
		swap, err = srcBridge.VerifyTransaction(args.SwapID, false)
	}
	if err != nil {
		logWorkerError("accept", "verifySignInfo failed", err, "txid", args.SwapID, "swaptype", args.SwapType)
		return err
	}

	buildTxArgs := &tokens.BuildTxArgs{
		SwapInfo: args.SwapInfo,
		To:       swap.Bind,
		Value:    swap.Value,
		Memo:     memo,
		Extra:    args.Extra,
	}
	rawTx, err := dstBridge.BuildRawTransaction(buildTxArgs)
	if err != nil {
		return err
	}
	return dstBridge.VerifyMsgHash(rawTx, msgHash, args.Extra)
}

type acceptSignInfo struct {
	keyID  string
	result string
}

func addAcceptSignHistory(keyID, result string) {
	// Create the new item as its own ring
	item := ring.New(1)
	item.Value = &acceptSignInfo{
		keyID:  keyID,
		result: result,
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
