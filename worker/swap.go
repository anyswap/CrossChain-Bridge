package worker

import (
	"container/ring"
	"fmt"
	"math/big"
	"sync"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var (
	swapinSwapStarter  sync.Once
	swapoutSwapStarter sync.Once

	swapRing        *ring.Ring
	swapRingLock    sync.RWMutex
	swapRingMaxSize = 1000
)

// StartSwapJob swap job
func StartSwapJob() {
	go startSwapinSwapJob()
	go startSwapoutSwapJob()
}

func startSwapinSwapJob() {
	swapinSwapStarter.Do(func() {
		logWorker("swap", "start swapin swap job")
		for {
			res, err := findSwapinsToSwap()
			if err != nil {
				logWorkerError("swapin", "find swapins error", err)
			}
			if len(res) > 0 {
				logWorker("swapin", "find swapins to swap", "count", len(res))
			}
			for _, swap := range res {
				err = processSwapinSwap(swap)
				if err != nil {
					logWorkerError("swapin", "process swapin swap error", err, "txid", swap.TxID)
				}
			}
			restInJob(restIntervalInDoSwapJob)
		}
	})
}

func startSwapoutSwapJob() {
	swapoutSwapStarter.Do(func() {
		logWorker("swapout", "start swapout swap job")
		for {
			res, err := findSwapoutsToSwap()
			if err != nil {
				logWorkerError("swapout", "find swapouts error", err)
			}
			if len(res) > 0 {
				logWorker("swapout", "find swapouts to swap", "count", len(res))
			}
			for _, swap := range res {
				err = processSwapoutSwap(swap)
				if err != nil {
					logWorkerError("swapout", "process swapout swap error", err)
				}
			}
			restInJob(restIntervalInDoSwapJob)
		}
	})
}

func findSwapinsToSwap() ([]*mongodb.MgoSwap, error) {
	status := mongodb.TxNotSwapped
	septime := getSepTimeInFind(maxDoSwapLifetime)
	return mongodb.FindSwapinsWithStatus(status, septime)
}

func findSwapoutsToSwap() ([]*mongodb.MgoSwap, error) {
	status := mongodb.TxNotSwapped
	septime := getSepTimeInFind(maxDoSwapLifetime)
	return mongodb.FindSwapoutsWithStatus(status, septime)
}

func processSwapinSwap(swap *mongodb.MgoSwap) (err error) {
	txid := swap.TxID
	bridge := tokens.DstBridge
	logWorker("swapin", "start processSwapinSwap", "txid", txid, "status", swap.Status)
	res, err := mongodb.FindSwapinResult(txid)
	if err != nil {
		return err
	}
	if tokens.GetTokenConfig(false).DisableSwap {
		logWorkerTrace("swapin", "swapin is disabled")
		return nil
	}
	if res.SwapTx != "" {
		if res.Status == mongodb.TxNotSwapped {
			_ = mongodb.UpdateSwapinStatus(txid, mongodb.TxProcessed, now(), "")
		}
		return fmt.Errorf("%v already swapped to %v", txid, res.SwapTx)
	}

	history := getSwapHistory(txid, true)
	if history != nil {
		if _, err = bridge.GetTransaction(history.matchTx); err == nil {
			matchTx := &MatchTx{
				SwapTx:    history.matchTx,
				SwapValue: tokens.CalcSwappedValue(history.value, true).String(),
				SwapType:  tokens.SwapinType,
				SwapNonce: history.nonce,
			}
			_ = updateSwapinResult(txid, matchTx)
			logWorker("swapin", "ignore swapped swapin", "txid", txid, "matchTx", history.matchTx)
			return fmt.Errorf("found swapped in history, txid=%v, matchTx=%v", txid, history.matchTx)
		}
	}

	value, err := common.GetBigIntFromStr(res.Value)
	if err != nil {
		return fmt.Errorf("wrong value %v", res.Value)
	}

	args := &tokens.BuildTxArgs{
		SwapInfo: tokens.SwapInfo{
			SwapID:   res.TxID,
			SwapType: tokens.SwapinType,
			TxType:   tokens.SwapTxType(swap.TxType),
			Bind:     swap.Bind,
		},
		To:    res.Bind,
		Value: value,
	}
	rawTx, err := bridge.BuildRawTransaction(args)
	if err != nil {
		logWorkerError("swapin", "BuildRawTransaction failed", err, "txid", txid)
		return err
	}

	signedTx, txHash, err := bridge.DcrmSignTransaction(rawTx, args.GetExtraArgs())
	if err != nil {
		logWorkerError("swapin", "DcrmSignTransaction failed", err, "txid", txid)
		return err
	}

	swapTxNonce := args.GetTxNonce()

	// update database before sending transaction
	addSwapHistory(txid, value, txHash, swapTxNonce, true)
	matchTx := &MatchTx{
		SwapTx:    txHash,
		SwapValue: tokens.CalcSwappedValue(value, true).String(),
		SwapType:  tokens.SwapinType,
		SwapNonce: swapTxNonce,
	}
	err = updateSwapinResult(txid, matchTx)
	if err != nil {
		logWorkerError("swapin", "updateSwapinResult failed", err, "txid", txid)
		return err
	}
	err = mongodb.UpdateSwapinStatus(txid, mongodb.TxProcessed, now(), "")
	if err != nil {
		logWorkerError("swapin", "UpdateSwapinStatus failed", err, "txid", txid)
		return err
	}

	return sendSignedTransaction(bridge, signedTx, txid, true)
}

func processSwapoutSwap(swap *mongodb.MgoSwap) (err error) {
	txid := swap.TxID
	bridge := tokens.SrcBridge
	logWorker("swapout", "start processSwapoutSwap", "txid", txid, "status", swap.Status)
	res, err := mongodb.FindSwapoutResult(txid)
	if err != nil {
		return err
	}
	if tokens.GetTokenConfig(true).DisableSwap {
		logWorkerTrace("swapout", "swapout is disabled")
		return nil
	}
	if res.SwapTx != "" {
		if res.Status == mongodb.TxNotSwapped {
			_ = mongodb.UpdateSwapoutStatus(txid, mongodb.TxProcessed, now(), "")
		}
		return fmt.Errorf("%v already swapped to %v", txid, res.SwapTx)
	}

	history := getSwapHistory(txid, false)
	if history != nil {
		if _, err = bridge.GetTransaction(history.matchTx); err == nil {
			matchTx := &MatchTx{
				SwapTx:    history.matchTx,
				SwapValue: tokens.CalcSwappedValue(history.value, false).String(),
				SwapType:  tokens.SwapoutType,
				SwapNonce: history.nonce,
			}
			_ = updateSwapoutResult(txid, matchTx)
			logWorker("swapout", "ignore swapped swapout", "txid", txid, "matchTx", history.matchTx)
			return fmt.Errorf("found swapped out history, txid=%v, matchTx=%v", txid, history.matchTx)
		}
	}

	value, err := common.GetBigIntFromStr(res.Value)
	if err != nil {
		return fmt.Errorf("wrong value %v", res.Value)
	}

	args := &tokens.BuildTxArgs{
		SwapInfo: tokens.SwapInfo{
			SwapID:   res.TxID,
			SwapType: tokens.SwapoutType,
		},
		To:    res.Bind,
		Value: value,
	}
	rawTx, err := bridge.BuildRawTransaction(args)
	if err != nil {
		logWorkerError("swapout", "BuildRawTransaction failed", err, "txid", txid)
		return err
	}

	signedTx, txHash, err := bridge.DcrmSignTransaction(rawTx, args.GetExtraArgs())
	if err != nil {
		logWorkerError("swapout", "DcrmSignTransaction failed", err, "txid", txid)
		return err
	}

	swapTxNonce := args.GetTxNonce()

	// update database before sending transaction
	addSwapHistory(txid, value, txHash, swapTxNonce, false)
	matchTx := &MatchTx{
		SwapTx:    txHash,
		SwapValue: tokens.CalcSwappedValue(value, false).String(),
		SwapType:  tokens.SwapoutType,
		SwapNonce: swapTxNonce,
	}
	err = updateSwapoutResult(txid, matchTx)
	if err != nil {
		logWorkerError("swapout", "updateSwapoutResult failed", err, "txid", txid)
		return err
	}
	err = mongodb.UpdateSwapoutStatus(txid, mongodb.TxProcessed, now(), "")
	if err != nil {
		logWorkerError("swapout", "UpdateSwapoutStatus failed", err, "txid", txid)
		return err
	}

	return sendSignedTransaction(bridge, signedTx, txid, false)
}

type swapInfo struct {
	txid     string
	value    *big.Int
	matchTx  string
	nonce    uint64
	isSwapin bool
}

func addSwapHistory(txid string, value *big.Int, matchTx string, nonce uint64, isSwapin bool) {
	// Create the new item as its own ring
	item := ring.New(1)
	item.Value = &swapInfo{
		txid:     txid,
		value:    value,
		matchTx:  matchTx,
		nonce:    nonce,
		isSwapin: isSwapin,
	}

	swapRingLock.Lock()
	defer swapRingLock.Unlock()

	if swapRing == nil {
		swapRing = item
	} else {
		if swapRing.Len() == swapRingMaxSize {
			swapRing = swapRing.Move(-1)
			swapRing.Unlink(1)
			swapRing = swapRing.Move(1)
		}
		swapRing.Move(-1).Link(item)
	}
}

func getSwapHistory(txid string, isSwapin bool) *swapInfo {
	swapRingLock.RLock()
	defer swapRingLock.RUnlock()

	if swapRing == nil {
		return nil
	}

	r := swapRing
	for i := 0; i < r.Len(); i++ {
		item := r.Value.(*swapInfo)
		if item.txid == txid && item.isSwapin == isSwapin {
			return item
		}
		r = r.Prev()
	}

	return nil
}
