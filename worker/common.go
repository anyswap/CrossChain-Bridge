package worker

import (
	"github.com/fsn-dev/crossChain-Bridge/mongodb"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
)

type MatchTx struct {
	SwapTx     string
	SwapHeight uint64
	SwapTime   uint64
	SwapValue  string
	SwapType   tokens.SwapType
}

func addInitialSwapinResult(tx *tokens.TxSwapInfo, status mongodb.SwapStatus) error {
	return addInitialSwapResult(tx, status, true)
}

func addInitialSwapoutResult(tx *tokens.TxSwapInfo, status mongodb.SwapStatus) error {
	return addInitialSwapResult(tx, status, false)
}

func addInitialSwapResult(tx *tokens.TxSwapInfo, status mongodb.SwapStatus, isSwapin bool) (err error) {
	txid := tx.Hash
	var swapType tokens.SwapType
	if isSwapin {
		swapType = tokens.Swap_Swapin
	} else {
		swapType = tokens.Swap_Swapout
	}
	swapResult := &mongodb.MgoSwapResult{
		Key:        txid,
		TxId:       txid,
		TxHeight:   tx.Height,
		TxTime:     tx.Timestamp,
		From:       tx.From,
		To:         tx.To,
		Bind:       tx.Bind,
		Value:      tx.Value.String(),
		SwapTx:     "",
		SwapHeight: 0,
		SwapTime:   0,
		SwapValue:  "0",
		SwapType:   uint32(swapType),
		Status:     status,
		Timestamp:  now(),
		Memo:       "",
	}
	if isSwapin {
		err = mongodb.AddSwapinResult(swapResult)
	} else {
		err = mongodb.AddSwapoutResult(swapResult)
	}
	if err != nil {
		logWorkerError("add", "addInitialSwapResult", err, "txid", txid)
	} else {
		logWorker("add", "addInitialSwapResult", "txid", txid)
	}
	return err
}

func updateSwapinResult(key string, mtx *MatchTx) error {
	return updateSwapResult(key, mtx)
}

func updateSwapoutResult(key string, mtx *MatchTx) error {
	return updateSwapResult(key, mtx)
}

func updateSwapResult(key string, mtx *MatchTx) (err error) {
	updates := &mongodb.SwapResultUpdateItems{
		Status:    mongodb.MatchTxNotStable,
		Timestamp: now(),
	}
	if mtx.SwapTx != "" {
		updates.SwapTx = mtx.SwapTx
		updates.SwapValue = mtx.SwapValue
		updates.SwapHeight = 0
		updates.SwapTime = 0
	} else {
		updates.SwapHeight = mtx.SwapHeight
		updates.SwapTime = mtx.SwapTime
	}
	switch mtx.SwapType {
	case tokens.Swap_Recall:
		updates.SwapType = uint32(mtx.SwapType)
		fallthrough
	case tokens.Swap_Swapin:
		err = mongodb.UpdateSwapinResult(key, updates)
	case tokens.Swap_Swapout:
		err = mongodb.UpdateSwapoutResult(key, updates)
	default:
		err = tokens.ErrUnknownSwapType
	}
	if err != nil {
		logWorkerError("update", "updateSwapResult", err, "txid", key, "swaptx", mtx.SwapTx, "swapheight", mtx.SwapHeight, "swaptime", mtx.SwapTime, "swapvalue", mtx.SwapValue, "swaptype", mtx.SwapType)
	} else {
		logWorker("update", "updateSwapResult", "txid", key, "swaptx", mtx.SwapTx, "swapheight", mtx.SwapHeight, "swaptime", mtx.SwapTime, "swapvalue", mtx.SwapValue, "swaptype", mtx.SwapType)
	}
	return err
}

func markSwapinResultStable(key string) error {
	return markSwapResultStable(key, true)
}

func markSwapoutResultStable(key string) error {
	return markSwapResultStable(key, false)
}

func markSwapResultStable(key string, isSwapin bool) (err error) {
	status := mongodb.MatchTxStable
	timestamp := now()
	memo := "" // unchange
	if isSwapin {
		err = mongodb.UpdateSwapinResultStatus(key, status, timestamp, memo)
	} else {
		err = mongodb.UpdateSwapoutResultStatus(key, status, timestamp, memo)
	}
	if err != nil {
		logWorkerError("stable", "markSwapResultStable", err, "txid", key, "isSwapin", isSwapin)
	} else {
		logWorker("stable", "markSwapResultStable", "txid", key, "isSwapin", isSwapin)
	}
	return err
}
