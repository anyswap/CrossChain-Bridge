package worker

import (
	"fmt"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc"
)

// MatchTx struct
type MatchTx struct {
	SwapTx     string
	SwapHeight uint64
	SwapTime   uint64
	SwapValue  string
	SwapType   tokens.SwapType
}

func addInitialSwapResult(swapInfo *tokens.TxSwapInfo, status mongodb.SwapStatus, isSwapin bool) (err error) {
	txid := swapInfo.Hash
	var swapType tokens.SwapType
	if isSwapin {
		swapType = tokens.SwapinType
	} else {
		swapType = tokens.SwapoutType
	}
	swapResult := &mongodb.MgoSwapResult{
		PairID:     swapInfo.PairID,
		TxID:       txid,
		TxTo:       swapInfo.TxTo,
		TxHeight:   swapInfo.Height,
		TxTime:     swapInfo.Timestamp,
		From:       swapInfo.From,
		To:         swapInfo.To,
		Bind:       swapInfo.Bind,
		Value:      swapInfo.Value.String(),
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

func updateSwapResult(txid, pairID, bind string, mtx *MatchTx) (err error) {
	updates := &mongodb.SwapResultUpdateItems{
		Status:    mongodb.MatchTxNotStable,
		Timestamp: now(),
	}
	if mtx.SwapHeight == 0 {
		updates.SwapTx = mtx.SwapTx
		updates.SwapValue = mtx.SwapValue
		updates.SwapHeight = 0
		updates.SwapTime = 0
	} else {
		updates.SwapHeight = mtx.SwapHeight
		updates.SwapTime = mtx.SwapTime
		if mtx.SwapTx != "" {
			updates.SwapTx = mtx.SwapTx
		}
	}
	switch mtx.SwapType {
	case tokens.SwapinType:
		err = mongodb.UpdateSwapinResult(txid, pairID, bind, updates)
	case tokens.SwapoutType:
		err = mongodb.UpdateSwapoutResult(txid, pairID, bind, updates)
	default:
		err = tokens.ErrUnknownSwapType
	}
	if err != nil {
		logWorkerError("update", "updateSwapResult", err,
			"txid", txid, "pairID", pairID, "bind", bind,
			"swaptx", mtx.SwapTx, "swapheight", mtx.SwapHeight,
			"swaptime", mtx.SwapTime, "swapvalue", mtx.SwapValue,
			"swaptype", mtx.SwapType)
	} else {
		logWorker("update", "updateSwapResult",
			"txid", txid, "pairID", pairID, "bind", bind,
			"swaptx", mtx.SwapTx, "swapheight", mtx.SwapHeight,
			"swaptime", mtx.SwapTime, "swapvalue", mtx.SwapValue,
			"swaptype", mtx.SwapType)
	}
	return err
}

func updateSwapResultHeight(swap *mongodb.MgoSwapResult, blockHeight, blockTime uint64, updateSwapTx bool) (err error) {
	updates := &mongodb.SwapResultUpdateItems{
		Status:    mongodb.KeepStatus,
		Timestamp: now(),
	}
	updates.SwapHeight = blockHeight
	updates.SwapTime = blockTime
	if updateSwapTx {
		updates.SwapTx = swap.SwapTx
	}
	txid := swap.TxID
	pairID := swap.PairID
	bind := swap.Bind
	switch tokens.SwapType(swap.SwapType) {
	case tokens.SwapinType:
		err = mongodb.UpdateSwapinResult(txid, pairID, bind, updates)
	case tokens.SwapoutType:
		err = mongodb.UpdateSwapoutResult(txid, pairID, bind, updates)
	default:
		err = tokens.ErrUnknownSwapType
	}
	if err != nil {
		logWorkerError("update", "updateSwapResultHeight", err, "txid", txid, "pairID", pairID, "bind", bind, "swaptx", swap.SwapTx, "height", blockHeight)
	} else {
		logWorker("update", "updateSwapResultHeight", "txid", txid, "pairID", pairID, "bind", bind, "swaptx", swap.SwapTx, "height", blockHeight)
	}
	return err
}

func updateSwapTx(txid, pairID, bind, swapTx string, isSwapin bool) (err error) {
	updates := &mongodb.SwapResultUpdateItems{
		Status:    mongodb.KeepStatus,
		SwapTx:    swapTx,
		Timestamp: now(),
	}
	if isSwapin {
		err = mongodb.UpdateSwapinResult(txid, pairID, bind, updates)
	} else {
		err = mongodb.UpdateSwapoutResult(txid, pairID, bind, updates)
	}
	if err != nil {
		logWorkerError("update", "updateSwapTx", err, "txid", txid, "pairID", pairID, "bind", bind, "swaptx", swapTx)
	} else {
		logWorker("update", "updateSwapTx", "txid", txid, "pairID", pairID, "bind", bind, "swaptx", swapTx)
	}
	return err
}

func updateOldSwapTxs(txid, pairID, bind string, oldSwapTxs []string, isSwapin bool) (err error) {
	updates := &mongodb.SwapResultUpdateItems{
		Status:     mongodb.KeepStatus,
		OldSwapTxs: oldSwapTxs,
		Timestamp:  now(),
	}
	if isSwapin {
		err = mongodb.UpdateSwapinResult(txid, pairID, bind, updates)
	} else {
		err = mongodb.UpdateSwapoutResult(txid, pairID, bind, updates)
	}
	if err != nil {
		logWorkerError("update", "updateOldSwapTxs", err, "txid", txid, "pairID", pairID, "bind", bind, "swaptxs", len(oldSwapTxs))
	} else {
		logWorker("update", "updateOldSwapTxs", "txid", txid, "pairID", pairID, "bind", bind, "swaptxs", len(oldSwapTxs))
	}
	return err
}

func markSwapResultStable(txid, pairID, bind string, isSwapin bool) (err error) {
	status := mongodb.MatchTxStable
	timestamp := now()
	memo := "" // unchange
	err = mongodb.UpdateSwapResultStatus(isSwapin, txid, pairID, bind, status, timestamp, memo)
	if err != nil {
		logWorkerError("stable", "markSwapResultStable", err, "txid", txid, "pairID", pairID, "bind", bind, "isSwapin", isSwapin)
	} else {
		logWorker("stable", "markSwapResultStable", "txid", txid, "pairID", pairID, "bind", bind, "isSwapin", isSwapin)
	}
	return err
}

func markSwapResultFailed(txid, pairID, bind string, isSwapin bool) (err error) {
	status := mongodb.MatchTxFailed
	timestamp := now()
	memo := "" // unchange
	err = mongodb.UpdateSwapResultStatus(isSwapin, txid, pairID, bind, status, timestamp, memo)
	if err != nil {
		logWorkerError("stable", "markSwapResultFailed", err, "txid", txid, "pairID", pairID, "bind", bind, "isSwapin", isSwapin)
	} else {
		logWorker("stable", "markSwapResultFailed", "txid", txid, "pairID", pairID, "bind", bind, "isSwapin", isSwapin)
	}
	return err
}

func verifySwapTransaction(bridge tokens.CrossChainBridge, pairID, txid, bind string, swapTxType tokens.SwapTxType) (swapInfo *tokens.TxSwapInfo, err error) {
	switch swapTxType {
	case tokens.P2shSwapinTx:
		if btc.BridgeInstance == nil {
			return nil, tokens.ErrNoBtcBridge
		}
		swapInfo, err = btc.BridgeInstance.VerifyP2shTransaction(pairID, txid, bind, false)
	default:
		swapInfo, err = bridge.VerifyTransaction(pairID, txid, false)
	}
	if swapInfo == nil {
		return nil, fmt.Errorf("empty swapinfo after verify tx")
	}
	if swapInfo.Bind != "" && !strings.EqualFold(swapInfo.Bind, bind) {
		return nil, tokens.ErrBindAddressMismatch
	}
	return swapInfo, err
}

func sendSignedTransaction(bridge tokens.CrossChainBridge, signedTx interface{}, txid, pairID, bind string, isSwapin bool) (err error) {
	var (
		txHash              string
		retrySendTxCount    = 3
		retrySendTxInterval = 1 * time.Second
	)
	for i := 0; i < retrySendTxCount; i++ {
		txHash, err = bridge.SendTransaction(signedTx)
		if txHash != "" {
			if tx, _ := bridge.GetTransaction(txHash); tx != nil {
				logWorker("sendtx", "send tx success", "txHash", txHash)
				err = nil
				break
			}
		}
		time.Sleep(retrySendTxInterval)
	}
	if err != nil {
		logWorkerError("sendtx", "update swap status to TxSwapFailed", err, "txid", txid, "bind", bind, "isSwapin", isSwapin)
		_ = mongodb.UpdateSwapStatus(isSwapin, txid, pairID, bind, mongodb.TxSwapFailed, now(), err.Error())
		_ = mongodb.UpdateSwapResultStatus(isSwapin, txid, pairID, bind, mongodb.TxSwapFailed, now(), err.Error())
	}
	return err
}

func assignSwapNonce(res *mongodb.MgoSwapResult, isSwapin bool) (swapNonce uint64, err error) {
	resBridge := tokens.GetCrossChainBridge(!isSwapin)
	nonceSetter, ok := resBridge.(tokens.NonceSetter)
	if !ok {
		return 0, nil // ignore bridge which does not support nonce
	}
	pairID := res.PairID
	tokenCfg := resBridge.GetTokenConfig(pairID)
	if tokenCfg == nil {
		return 0, tokens.ErrUnknownPairID
	}
	dcrmAddress := tokenCfg.DcrmAddress
	for { // looop until success
		swapNonce, err = nonceSetter.GetPoolNonce(dcrmAddress, "pending")
		if err == nil {
			swapNonce = nonceSetter.AdjustNonce(pairID, swapNonce)
			break
		}
		time.Sleep(1 * time.Second)
	}
	err = mongodb.AssginSwapNonce(isSwapin, res.TxID, pairID, res.Bind, swapNonce)
	if err != nil {
		return 0, err
	}
	nonceSetter.SetNonce(pairID, swapNonce+1) // increase for next usage
	return swapNonce, nil
}
