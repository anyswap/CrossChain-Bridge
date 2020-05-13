package worker

import (
	"github.com/fsn-dev/crossChain-Bridge/log"
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

func addInitialSwapinResult(tx *tokens.TxSwapInfo) error {
	return addInitialSwapResult(tx, true)
}

func addInitialSwapoutResult(tx *tokens.TxSwapInfo) error {
	return addInitialSwapResult(tx, false)
}

func addInitialSwapResult(tx *tokens.TxSwapInfo, isSwapin bool) (err error) {
	if tx == nil {
		log.Warn("addInitialSwapoutResult add empty swap", "isSwapin", isSwapin)
		return nil
	}
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
		SwapValue:  "0",
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
		updates.SwapType = uint32(mtx.SwapType)
	} else {
		updates.SwapHeight = mtx.SwapHeight
		updates.SwapTime = mtx.SwapTime
	}
	switch mtx.SwapType {
	case tokens.Swap_Swapin, tokens.Swap_Recall:
		err = mongodb.UpdateSwapinResult(key, updates)
	case tokens.Swap_Swapout:
		err = mongodb.UpdateSwapoutResult(key, updates)
	default:
		err = tokens.ErrUnknownSwapType
	}
	if err != nil {
		log.Debug("updateSwapResult", "txid", key, "swaptx", mtx.SwapTx, "swapheight", mtx.SwapHeight, "swaptime", mtx.SwapTime, "swapvalue", mtx.SwapValue, "swaptype", mtx.SwapType, "err", err)
	} else {
		log.Debug("updateSwapResult", "txid", key, "swaptx", mtx.SwapTx, "swapheight", mtx.SwapHeight, "swaptime", mtx.SwapTime, "swapvalue", mtx.SwapValue, "swaptype", mtx.SwapType)
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
