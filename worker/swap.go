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

func isSwapInBlacklist(swap *mongodb.MgoSwapResult) (isBlacked bool, err error) {
	isBlacked, err = mongodb.QueryBlacklist(swap.From, swap.PairID)
	if err != nil {
		return isBlacked, err
	}
	if !isBlacked && swap.Bind != swap.From {
		isBlacked, err = mongodb.QueryBlacklist(swap.Bind, swap.PairID)
		if err != nil {
			return isBlacked, err
		}
	}
	return isBlacked, nil
}

func processSwapinSwap(swap *mongodb.MgoSwap) (err error) {
	return processSwap(swap, true)
}

func processSwapoutSwap(swap *mongodb.MgoSwap) (err error) {
	return processSwap(swap, false)
}

func processSwap(swap *mongodb.MgoSwap, isSwapin bool) (err error) {
	txid := swap.TxID
	logWorker("swap", "start process swap", "txid", txid, "status", swap.Status, "isSwapin", isSwapin)

	var resBridge tokens.CrossChainBridge
	var swapType tokens.SwapType
	if isSwapin {
		resBridge = tokens.DstBridge
		swapType = tokens.SwapinType
	} else {
		resBridge = tokens.SrcBridge
		swapType = tokens.SwapoutType
	}

	res, err := mongodb.FindSwapResult(isSwapin, txid)
	if err != nil {
		return err
	}
	if tokens.GetTokenConfig(swap.PairID, isSwapin).DisableSwap {
		logWorkerTrace("swap", "swap is disabled", "isSwapin", isSwapin)
		return nil
	}
	isBlacked, err := isSwapInBlacklist(res)
	if err != nil {
		return err
	}
	if isBlacked {
		logWorkerTrace("swap", "address is in blacklist", "txid", txid, "isSwapin", isSwapin)
		err = tokens.ErrAddressIsInBlacklist
		_ = mongodb.UpdateSwapStatus(isSwapin, txid, mongodb.SwapInBlacklist, now(), err.Error())
		return nil
	}
	if res.SwapTx != "" {
		_ = mongodb.UpdateSwapStatus(isSwapin, txid, mongodb.TxProcessed, now(), "")
		if res.Status != mongodb.MatchTxEmpty {
			return fmt.Errorf("%v already swapped to %v with status %v", txid, res.SwapTx, res.Status)
		}
		if _, err = resBridge.GetTransaction(res.SwapTx); err == nil {
			return fmt.Errorf("[warn] %v already swapped to %v but with status %v", txid, res.SwapTx, res.Status)
		}
	}

	history := getSwapHistory(txid, isSwapin)
	if history != nil {
		if _, err = resBridge.GetTransaction(history.matchTx); err == nil {
			matchTx := &MatchTx{
				SwapTx:    history.matchTx,
				SwapValue: tokens.CalcSwappedValue(swap.PairID, history.value, isSwapin).String(),
				SwapType:  swapType,
				SwapNonce: history.nonce,
			}
			_ = updateSwapResult(txid, matchTx)
			logWorker("swap", "ignore swapped swap", "txid", txid, "matchTx", history.matchTx, "isSwapin", isSwapin)
			return fmt.Errorf("found swapped in history, txid=%v, matchTx=%v", txid, history.matchTx)
		}
	}

	value, err := common.GetBigIntFromStr(res.Value)
	if err != nil {
		return fmt.Errorf("wrong value %v", res.Value)
	}

	args := &tokens.BuildTxArgs{
		SwapInfo: tokens.SwapInfo{
			SwapID:   txid,
			SwapType: swapType,
		},
		To:    res.Bind,
		Value: value,
	}

	if isSwapin {
		args.TxType = tokens.SwapTxType(swap.TxType)
		args.Bind = swap.Bind
	}

	return doSwap(resBridge, args, isSwapin)
}

func doSwap(resBridge tokens.CrossChainBridge, args *tokens.BuildTxArgs, isSwapin bool) (err error) {
	txid := args.SwapID
	swapType := args.SwapType
	originValue := args.Value
	if isSwapin != (swapType == tokens.SwapinType) {
		return fmt.Errorf("mismatch isSwapin=%v but swapType=%v", isSwapin, swapType.String())
	}
	rawTx, err := resBridge.BuildRawTransaction(args)
	if err != nil {
		logWorkerError("doSwap", "BuildRawTransaction failed", err, "txid", txid, "isSwapin", isSwapin)
		return err
	}

	signedTx, txHash, err := dcrmSignTransaction(resBridge, rawTx, args.GetExtraArgs())
	if err != nil {
		logWorkerError("doSwap", "DcrmSignTransaction failed", err, "txid", txid, "isSwapin", isSwapin)
		return err
	}

	swapTxNonce := args.GetTxNonce()

	// update database before sending transaction
	addSwapHistory(txid, originValue, txHash, swapTxNonce, isSwapin)
	matchTx := &MatchTx{
		SwapTx:    txHash,
		SwapValue: tokens.CalcSwappedValue(args.PairID, originValue, isSwapin).String(),
		SwapType:  swapType,
		SwapNonce: swapTxNonce,
	}
	err = updateSwapResult(txid, matchTx)
	if err != nil {
		logWorkerError("doSwap", "update swap result failed", err, "txid", txid, "isSwapin", isSwapin)
		return err
	}

	err = mongodb.UpdateSwapStatus(isSwapin, txid, mongodb.TxProcessed, now(), "")
	if err != nil {
		logWorkerError("doSwap", "update swap status failed", err, "txid", txid, "isSwapin", isSwapin)
		return err
	}

	return sendSignedTransaction(args.PairID, resBridge, signedTx, txid, isSwapin)
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
