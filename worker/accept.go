package worker

import (
	"container/ring"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/dcrm"
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/params"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
)

var (
	acceptSignStarter sync.Once

	acceptRing        *ring.Ring
	acceptRingLock    sync.RWMutex
	acceptRingMaxSize = 500

	retryInterval = 3 * time.Second
	waitInterval  = 20 * time.Second

	ErrIdentifierMismatch = errors.New("cross chain bridge identifier mismatch")
)

func StartAcceptSignJob() error {
	acceptSignStarter.Do(func() {
		logWorker("accept", "start accept sign job")
		acceptSign()
	})
	return nil
}

func acceptSign() error {
	for {
		signInfo, err := dcrm.GetCurNodeSignInfo()
		if err != nil {
			time.Sleep(retryInterval)
			continue
		}
		log.Debug("acceptSign", "count", len(signInfo))
		for _, info := range signInfo {
			keyID := info.Key
			history := getAcceptSignHistory(keyID)
			if history != nil {
				logWorker("accept", "ignore accepted sign", "keyID", keyID, "result", history.result)
				dcrm.DoAcceptSign(keyID, history.result)
				continue
			}
			agreeResult := "AGREE"
			err := verifySignInfo(info)
			switch err {
			case ErrIdentifierMismatch,
				tokens.ErrTxNotStable,
				tokens.ErrTxNotFound:
				continue
			}
			if err != nil {
				logWorkerError("accept", "disagree sign", err, "keyID", keyID)
				agreeResult = "DISAGREE"
			}
			log.Debug("dcrm DoAcceptSign", "keyID", keyID, "result", agreeResult)
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
	return nil
}

func verifySignInfo(signInfo *dcrm.SignInfoData) error {
	msgHash := signInfo.MsgHash
	msgContext := signInfo.MsgContext
	log.Debug("verifySignInfo", "msgContext", msgContext)
	var args tokens.BuildTxArgs
	err := json.Unmarshal([]byte(msgContext), &args)
	if err != nil {
		return err
	}
	if args.Identifier != params.GetIdentifier() {
		return ErrIdentifierMismatch
	}
	var (
		srcBridge, dstBridge tokens.CrossChainBridge
		memo                 string
	)
	switch args.SwapType {
	case tokens.Swap_Swapin:
		srcBridge = tokens.SrcBridge
		dstBridge = tokens.DstBridge
	case tokens.Swap_Swapout:
		srcBridge = tokens.DstBridge
		dstBridge = tokens.SrcBridge
		memo = fmt.Sprintf("%s%s", tokens.UnlockMemoPrefix, args.SwapID)
	case tokens.Swap_Recall:
		srcBridge = tokens.SrcBridge
		dstBridge = tokens.SrcBridge
		memo = fmt.Sprintf("%s%s", tokens.RecallMemoPrefix, args.SwapID)
	default:
		return fmt.Errorf("unknown swap type %v", args.SwapType)
	}
	swap, err := srcBridge.VerifyTransaction(args.SwapID, false)
	if err != nil {
		log.Info("verifySignInfo failed", "txid", args.SwapID, "swaptype", args.SwapType, "err", err)
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
