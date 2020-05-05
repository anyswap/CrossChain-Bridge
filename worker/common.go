package worker

import (
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/mongodb"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
)

type MatchTx struct {
	SwapTx        string
	SwapHeight    uint64
	SwapTime      uint64
	SetRecallMemo bool
}

const RecallTxMemo = "IsRecalled"

func addInitialSwapinResult(tx *tokens.TxSwapInfo) error {
	return addInitialSwapResult(tx, true)
}

func addInitialSwapoutResult(tx *tokens.TxSwapInfo) error {
	return addInitialSwapResult(tx, false)
}

func addInitialSwapResult(tx *tokens.TxSwapInfo, isSwapin bool) (err error) {
	txid := tx.Hash
	swapResult := &mongodb.MgoSwapResult{
		Key:        txid,
		TxId:       txid,
		TxHeight:   tx.Height,
		TxTime:     tx.Timestamp,
		From:       tx.From,
		To:         tx.To,
		Bind:       tx.Bind,
		Value:      tx.Value,
		SwapTx:     "",
		SwapHeight: 0,
		SwapTime:   0,
		Status:     mongodb.MatchTxEmpty,
		Timestamp:  now(),
		Memo:       "",
	}
	if isSwapin {
		err = mongodb.AddSwapinResult(swapResult)
	} else {
		err = mongodb.AddSwapoutResult(swapResult)
	}
	if err != nil {
		log.Debug("addInitialSwapResult", "txid", txid, "err", err)
	} else {
		log.Debug("addInitialSwapResult", "txid", txid)
	}
	return err
}

func updateSwapinResult(key string, mtx *MatchTx) error {
	return updateSwapResult(key, mtx, true)
}

func updateSwapoutResult(key string, mtx *MatchTx) error {
	return updateSwapResult(key, mtx, false)
}

func updateSwapResult(key string, mtx *MatchTx, isSwapin bool) (err error) {
	memo := ""
	if mtx.SetRecallMemo {
		memo = RecallTxMemo
	}
	updates := &mongodb.SwapResultUpdateItems{
		SwapTx:     mtx.SwapTx,
		SwapHeight: mtx.SwapHeight,
		SwapTime:   mtx.SwapTime,
		Status:     mongodb.MatchTxNotStable,
		Timestamp:  now(),
		Memo:       memo,
	}
	if isSwapin {
		err = mongodb.UpdateSwapinResult(key, updates)
	} else {
		err = mongodb.UpdateSwapoutResult(key, updates)
	}
	if err != nil {
		log.Debug("updateSwapResult", "txid", key, "swaptx", mtx.SwapTx, "swapheight", mtx.SwapHeight, "swaptime", mtx.SwapTime, "memo", memo, "err", err)
	} else {
		log.Debug("updateSwapResult", "txid", key, "swaptx", mtx.SwapTx, "swapheight", mtx.SwapHeight, "swaptime", mtx.SwapTime, "memo", memo)
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
		log.Debug("markSwapResultStable", "txid", key, "isSwapin", isSwapin, "err", err)
	} else {
		log.Debug("markSwapResultStable", "txid", key, "isSwapin", isSwapin)
	}
	return err
}
