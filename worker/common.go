package worker

import (
	"time"

	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// MatchTx struct
type MatchTx struct {
	SwapTx     string
	SwapHeight uint64
	SwapTime   uint64
	SwapValue  string
	SwapType   tokens.SwapType
	SwapNonce  uint64
}

func addInitialSwapResult(tx *tokens.TxSwapInfo, status mongodb.SwapStatus, isSwapin bool) (err error) {
	txid := tx.Hash
	var swapType tokens.SwapType
	if isSwapin {
		swapType = tokens.SwapinType
	} else {
		swapType = tokens.SwapoutType
	}
	swapResult := &mongodb.MgoSwapResult{
		PairID:     tx.PairID,
		TxID:       txid,
		TxTo:       tx.TxTo,
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
		SwapNonce:  0,
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

func updateSwapResult(txid, pairID string, mtx *MatchTx) (err error) {
	updates := &mongodb.SwapResultUpdateItems{
		Status:    mongodb.MatchTxNotStable,
		Timestamp: now(),
	}
	if mtx.SwapTx != "" {
		updates.SwapTx = mtx.SwapTx
		updates.SwapValue = mtx.SwapValue
		updates.SwapNonce = mtx.SwapNonce
		updates.SwapHeight = 0
		updates.SwapTime = 0
	} else {
		updates.SwapHeight = mtx.SwapHeight
		updates.SwapTime = mtx.SwapTime
	}
	switch mtx.SwapType {
	case tokens.SwapinType:
		err = mongodb.UpdateSwapinResult(txid, pairID, updates)
	case tokens.SwapoutType:
		err = mongodb.UpdateSwapoutResult(txid, pairID, updates)
	default:
		err = tokens.ErrUnknownSwapType
	}
	if err != nil {
		logWorkerError("update", "updateSwapResult", err,
			"txid", txid, "pairID", pairID,
			"swaptx", mtx.SwapTx, "swapheight", mtx.SwapHeight,
			"swaptime", mtx.SwapTime, "swapvalue", mtx.SwapValue,
			"swaptype", mtx.SwapType, "swapnonce", mtx.SwapNonce)
	} else {
		logWorker("update", "updateSwapResult",
			"txid", txid, "pairID", pairID,
			"swaptx", mtx.SwapTx, "swapheight", mtx.SwapHeight,
			"swaptime", mtx.SwapTime, "swapvalue", mtx.SwapValue,
			"swaptype", mtx.SwapType, "swapnonce", mtx.SwapNonce)
	}
	return err
}

func markSwapResultStable(txid, pairID string, isSwapin bool) (err error) {
	status := mongodb.MatchTxStable
	timestamp := now()
	memo := "" // unchange
	err = mongodb.UpdateSwapResultStatus(isSwapin, txid, pairID, status, timestamp, memo)
	if err != nil {
		logWorkerError("stable", "markSwapResultStable", err, "txid", txid, "pairID", pairID, "isSwapin", isSwapin)
	} else {
		logWorker("stable", "markSwapResultStable", "txid", txid, "pairID", pairID, "isSwapin", isSwapin)
	}
	return err
}

func markSwapResultFailed(txid, pairID string, isSwapin bool) (err error) {
	status := mongodb.MatchTxFailed
	timestamp := now()
	memo := "" // unchange
	err = mongodb.UpdateSwapResultStatus(isSwapin, txid, pairID, status, timestamp, memo)
	if err != nil {
		logWorkerError("stable", "markSwapResultFailed", err, "txid", txid, "pairID", pairID, "isSwapin", isSwapin)
	} else {
		logWorker("stable", "markSwapResultFailed", "txid", txid, "pairID", pairID, "isSwapin", isSwapin)
	}
	return err
}

func dcrmSignTransaction(bridge tokens.CrossChainBridge, rawTx interface{}, args *tokens.BuildTxArgs) (signedTx interface{}, txHash string, err error) {
	maxRetryDcrmSignCount := 5
	for i := 0; i < maxRetryDcrmSignCount; i++ {
		signedTx, txHash, err = bridge.DcrmSignTransaction(rawTx, args.GetExtraArgs())
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, "", err
	}
	return signedTx, txHash, nil
}

func sendSignedTransaction(bridge tokens.CrossChainBridge, signedTx interface{}, txid, pairID string, isSwapin bool) (err error) {
	var (
		txHash              string
		retrySendTxCount    = 3
		retrySendTxInterval = 1 * time.Second
	)
	for i := 0; i < retrySendTxCount; i++ {
		txHash, err = bridge.SendTransaction(signedTx)
		if tx, _ := bridge.GetTransaction(txHash); tx != nil {
			err = nil
			break
		}
		time.Sleep(retrySendTxInterval)
	}
	if err != nil {
		logWorkerError("sendtx", "update swap status to TxSwapFailed", err, "txid", txid, "isSwapin", isSwapin)
		_ = mongodb.UpdateSwapStatus(isSwapin, txid, pairID, mongodb.TxSwapFailed, now(), err.Error())
		_ = mongodb.UpdateSwapResultStatus(isSwapin, txid, pairID, mongodb.TxSwapFailed, now(), err.Error())
		return err
	}
	bridge.IncreaseNonce(pairID, 1)
	return nil
}
