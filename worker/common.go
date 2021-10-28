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
	SwapNonce  uint64
}

func getSwapType(isSwapin bool) tokens.SwapType {
	if isSwapin {
		return tokens.SwapinType
	}
	return tokens.SwapoutType
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
		Status:    mongodb.KeepStatus,
		Timestamp: now(),
	}
	if mtx.SwapHeight == 0 {
		updates.SwapValue = mtx.SwapValue
		updates.SwapNonce = mtx.SwapNonce
		updates.SwapHeight = 0
		updates.SwapTime = 0
		if mtx.SwapTx != "" {
			updates.SwapTx = mtx.SwapTx
			updates.Status = mongodb.MatchTxNotStable
		}
	} else {
		updates.SwapNonce = mtx.SwapNonce
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
			"swaptype", mtx.SwapType, "swapnonce", mtx.SwapNonce)
	} else {
		logWorker("update", "updateSwapResult",
			"txid", txid, "pairID", pairID, "bind", bind,
			"swaptx", mtx.SwapTx, "swapheight", mtx.SwapHeight,
			"swaptime", mtx.SwapTime, "swapvalue", mtx.SwapValue,
			"swaptype", mtx.SwapType, "swapnonce", mtx.SwapNonce)
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
		updates.SwapValue = swap.SwapValue
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

func updateSwapTimestamp(txid, pairID, bind string, isSwapin bool) (err error) {
	updates := &mongodb.SwapResultUpdateItems{
		Status:    mongodb.KeepStatus,
		Timestamp: now(),
	}
	if isSwapin {
		err = mongodb.UpdateSwapinResult(txid, pairID, bind, updates)
	} else {
		err = mongodb.UpdateSwapoutResult(txid, pairID, bind, updates)
	}
	if err != nil {
		logWorkerError("update", "updateSwapTimestamp", err, "txid", txid, "pairID", pairID, "bind", bind)
	} else {
		logWorker("update", "updateSwapTimestamp", "txid", txid, "pairID", pairID, "bind", bind)
	}
	return err
}

func updateSwapResultTx(txid, pairID, bind, swapTx, swapValue string, isSwapin bool, status mongodb.SwapStatus) (err error) {
	updates := &mongodb.SwapResultUpdateItems{
		Status:    status,
		SwapTx:    swapTx,
		SwapValue: swapValue,
		Timestamp: now(),
	}
	if isSwapin {
		err = mongodb.UpdateSwapinResult(txid, pairID, bind, updates)
	} else {
		err = mongodb.UpdateSwapoutResult(txid, pairID, bind, updates)
	}
	if err != nil {
		logWorkerError("update", "updateSwapResultTx", err, "txid", txid, "pairID", pairID, "bind", bind, "swaptx", swapTx)
	} else {
		logWorker("update", "updateSwapResultTx", "txid", txid, "pairID", pairID, "bind", bind, "swaptx", swapTx)
	}
	return err
}

func updateOldSwapTxs(txid, pairID, bind, swapTx string, oldSwapTxs, oldSwapVals []string, isSwapin bool) (err error) {
	if len(oldSwapTxs) != len(oldSwapVals) {
		return fmt.Errorf("update old swaptxs with different count of values")
	}
	updates := &mongodb.SwapResultUpdateItems{
		Status:      mongodb.KeepStatus,
		SwapTx:      swapTx,
		OldSwapTxs:  oldSwapTxs,
		OldSwapVals: oldSwapVals,
		Timestamp:   now(),
	}
	if isSwapin {
		err = mongodb.UpdateSwapinResult(txid, pairID, bind, updates)
	} else {
		err = mongodb.UpdateSwapoutResult(txid, pairID, bind, updates)
	}
	if err != nil {
		logWorkerError("update", "updateOldSwapTxs", err, "txid", txid, "pairID", pairID, "bind", bind, "swapTx", swapTx, "swaptxs", len(oldSwapTxs))
	} else {
		logWorker("update", "updateOldSwapTxs", "txid", txid, "pairID", pairID, "bind", bind, "swapTx", swapTx, "swaptxs", len(oldSwapTxs))
	}
	return err
}

func markSwapResultUnstable(txid, pairID, bind string, isSwapin bool) (err error) {
	status := mongodb.MatchTxNotStable
	timestamp := now()
	memo := "" // unchange
	err = mongodb.UpdateSwapResultStatus(isSwapin, txid, pairID, bind, status, timestamp, memo)
	if err != nil {
		logWorkerError("checkfailedswap", "markSwapResultUnstable", err, "txid", txid, "pairID", pairID, "bind", bind, "isSwapin", isSwapin)
	} else {
		logWorker("checkfailedswap", "markSwapResultUnstable", "txid", txid, "pairID", pairID, "bind", bind, "isSwapin", isSwapin)
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

func sendSignedTransaction(bridge tokens.CrossChainBridge, signedTx interface{}, args *tokens.BuildTxArgs) (txHash string, err error) {
	var (
		retrySendTxCount    = 3
		retrySendTxInterval = 1 * time.Second
		txid, pairID, bind  = args.SwapID, args.PairID, args.Bind
		isSwapin            = args.SwapType == tokens.SwapinType
	)
	for i := 0; i < retrySendTxCount; i++ {
		txHash, err = bridge.SendTransaction(signedTx)
		if err == nil {
			logWorker("sendtx", "send tx success", "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin, "txHash", txHash)
			break
		}
		time.Sleep(retrySendTxInterval)
	}
	if txHash != "" {
		addSwapHistory(isSwapin, txid, bind)
		_ = mongodb.AddSwapHistory(isSwapin, txid, bind, txHash)
	}
	if err != nil {
		logWorkerError("sendtx", "send tx failed", err, "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin, "txHash", txHash)
		return txHash, err
	}

	nonceSetter, _ := bridge.(tokens.NonceSetter)
	if nonceSetter == nil {
		return txHash, err
	}

	nonceSetter.SetNonce(pairID, args.GetTxNonce()+1) // increase for next usage

	// update swap result tx height in goroutine
	go func() {
		var blockHeight, blockTime uint64
		for i := int64(0); i < 10; i++ {
			blockHeight, blockTime = nonceSetter.GetTxBlockInfo(txHash)
			if blockHeight > 0 {
				break
			}
			time.Sleep(5 * time.Second)
		}
		if blockHeight > 0 {
			matchTx := &MatchTx{
				SwapTx:     txHash,
				SwapHeight: blockHeight,
				SwapTime:   blockTime,
			}
			_ = updateSwapResult(txid, pairID, bind, matchTx)
		}
	}()

	return txHash, err
}
