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
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	"github.com/fsn-dev/crossChain-Bridge/types"
)

var (
	acceptSignStarter sync.Once

	acceptRing        *ring.Ring
	acceptRingLock    sync.RWMutex
	acceptRingMaxSize = 500

	retryInterval = 3 * time.Second
	waitInterval  = 20 * time.Second
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
				continue
			}
			agreeResult := "AGREE"
			if err := verifySignInfo(info); err != nil {
				logWorkerError("accept", "disagree sign", err, "keyID", keyID)
				agreeResult = "DISAGREE"
			}
			res, err := dcrm.DoAcceptSign(keyID, agreeResult)
			if err != nil {
				logWorkerError("accept", "accept sign job failed", err, "keyID", keyID)
			} else {
				logWorker("accept", "accept sign job success", "keyID", keyID, "result", res)
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
	var info tokens.SwapInfo
	err := json.Unmarshal([]byte(msgContext), &info)
	if err != nil {
		return err
	}
	var (
		srcBridge, dstBridge tokens.CrossChainBridge
		isSwapIn             bool
	)
	switch info.SwapType {
	case tokens.Swap_Swapin:
		srcBridge = tokens.SrcBridge
		dstBridge = tokens.DstBridge
		isSwapIn = true
	case tokens.Swap_Swapout:
		srcBridge = tokens.DstBridge
		dstBridge = tokens.SrcBridge
	case tokens.Swap_Recall:
		srcBridge = tokens.SrcBridge
		dstBridge = tokens.SrcBridge
	}
	swapInfo, err := srcBridge.VerifyTransaction(info.TxHash)
	value, err := common.GetBigIntFromStr(swapInfo.Value)
	if err != nil {
		return fmt.Errorf("wrong value %v", swapInfo.Value)
	}
	gasPrice, err := common.GetBigIntFromStr(info.Extra)
	if err != nil {
		return fmt.Errorf("wrong gas price %v", info.Extra)
	}
	args := &tokens.BuildTxArgs{
		IsSwapin: isSwapIn,
		To:       swapInfo.Bind,
		Value:    value,
		Memo:     swapInfo.Hash,
		GasPrice: gasPrice,
	}
	rawTx, err := dstBridge.BuildRawTransaction(args)
	if err != nil {
		return err
	}
	tx, ok := rawTx.(*types.Transaction)
	if !ok {
		return errors.New("wrong raw tx")
	}
	sigHash := dcrm.Signer.Hash(tx)
	if sigHash.String() != msgHash {
		return errors.New("msg hash mismatch")
	}
	return nil
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
