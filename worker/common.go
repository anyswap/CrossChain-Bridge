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
		logWorkerError("addSwap", "addInitialSwapResult failed", err, "pairID", swapInfo.PairID, "txid", txid)
	} else {
		logWorker("addSwap", "addInitialSwapResult success", "pairID", swapInfo.PairID, "txid", txid)
	}
	return err
}

func updateSwapResult(txid, pairID, bind string, mtx *MatchTx) (err error) {
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
		err = mongodb.UpdateSwapinResult(txid, pairID, bind, updates)
	case tokens.SwapoutType:
		err = mongodb.UpdateSwapoutResult(txid, pairID, bind, updates)
	default:
		err = tokens.ErrUnknownSwapType
	}
	if err != nil {
		logWorkerError("updateSwap", "updateSwapResult", err,
			"txid", txid, "pairID", pairID, "bind", bind,
			"swaptx", mtx.SwapTx, "swapheight", mtx.SwapHeight,
			"swaptime", mtx.SwapTime, "swapvalue", mtx.SwapValue,
			"swaptype", mtx.SwapType, "swapnonce", mtx.SwapNonce)
	} else {
		logWorker("updateSwap", "updateSwapResult",
			"txid", txid, "pairID", pairID, "bind", bind,
			"swaptx", mtx.SwapTx, "swapheight", mtx.SwapHeight,
			"swaptime", mtx.SwapTime, "swapvalue", mtx.SwapValue,
			"swaptype", mtx.SwapType, "swapnonce", mtx.SwapNonce)
	}
	return err
}

func markSwapResultStable(txid, pairID, bind string, isSwapin bool) (err error) {
	status := mongodb.MatchTxStable
	timestamp := now()
	memo := "" // unchange
	err = mongodb.UpdateSwapResultStatus(isSwapin, txid, pairID, bind, status, timestamp, memo)
	if err != nil {
		logWorkerError("stableSwap", "markSwapResultStable", err, "txid", txid, "pairID", pairID, "bind", bind, "isSwapin", isSwapin)
	} else {
		logWorker("stableSwap", "markSwapResultStable", "txid", txid, "pairID", pairID, "bind", bind, "isSwapin", isSwapin)
	}
	return err
}

func markSwapResultFailed(txid, pairID, bind string, isSwapin bool) (err error) {
	status := mongodb.MatchTxFailed
	timestamp := now()
	memo := "" // unchange
	err = mongodb.UpdateSwapResultStatus(isSwapin, txid, pairID, bind, status, timestamp, memo)
	if err != nil {
		logWorkerError("stableSwap", "markSwapResultFailed", err, "txid", txid, "pairID", pairID, "bind", bind, "isSwapin", isSwapin)
	} else {
		logWorker("stableSwap", "markSwapResultFailed", "txid", txid, "pairID", pairID, "bind", bind, "isSwapin", isSwapin)
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
				logWorker("sendtx", "send tx success", "pairID", pairID, "txid", txid, "isSwapin", isSwapin, "txHash", txHash)
				err = nil
				break
			}
		}
		time.Sleep(retrySendTxInterval)
	}
	if err != nil {
		logWorkerError("sendtx", "update swap status to TxSwapFailed", err, "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin)
		_ = mongodb.UpdateSwapStatus(isSwapin, txid, pairID, bind, mongodb.TxSwapFailed, now(), err.Error())
		_ = mongodb.UpdateSwapResultStatus(isSwapin, txid, pairID, bind, mongodb.TxSwapFailed, now(), err.Error())
		return err
	}
	if nonceSetter, ok := bridge.(tokens.NonceSetter); ok {
		nonceSetter.IncreaseNonce(pairID, 1)
	}
	return nil
}
