package worker

import (
	"container/ring"
	"fmt"
	"sync"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/mongodb"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
)

var (
	swapinSwapStarter  sync.Once
	swapoutSwapStarter sync.Once

	swapRing        *ring.Ring
	swapRingLock    sync.RWMutex
	swapRingMaxSize = 1000
)

func StartSwapJob() error {
	go startSwapinSwapJob()
	go startSwapoutSwapJob()
	return nil
}

func startSwapinSwapJob() error {
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
					logWorkerError("swapin", "process swapin swap error", err, "txid", swap.TxId)
				}
			}
			restInJob(restIntervalInDoSwapJob)
		}
	})
	return nil
}

func startSwapoutSwapJob() error {
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
	return nil
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
	txid := swap.TxId
	log.Debug("start processSwapinSwap", "txid", txid, "status", swap.Status)
	history := getSwapHistory(txid, true)
	if history != nil {
		logWorker("swapin", "ignore swapped swapin", "txid", txid, "matchTx", history.matchTx)
		return fmt.Errorf("found swapped in history, txid=%v, matchTx=%v", txid, history.matchTx)
	}
	res, err := mongodb.FindSwapinResult(txid)
	if err != nil {
		return err
	}
	if res.SwapTx != "" {
		if res.Status == mongodb.TxNotSwapped {
			mongodb.UpdateSwapinStatus(txid, mongodb.TxProcessed, now(), "")
		}
		return fmt.Errorf("%v already swapped to %v", txid, res.SwapTx)
	}

	value, err := common.GetBigIntFromStr(res.Value)
	if err != nil {
		return fmt.Errorf("wrong value %v", res.Value)
	}

	args := &tokens.BuildTxArgs{
		SwapInfo: tokens.SwapInfo{
			SwapID:   res.TxId,
			SwapType: tokens.Swap_Swapin,
		},
		To:    res.Bind,
		Value: value,
	}
	bridge := tokens.DstBridge
	rawTx, err := bridge.BuildRawTransaction(args)
	if err != nil {
		return err
	}
	if rawTx == nil {
		return fmt.Errorf("build raw tx is empty, txid=%v", txid)
	}

	signedTx, err := bridge.DcrmSignTransaction(rawTx, args.GetExtraArgs())
	if err != nil {
		return err
	}
	if signedTx == nil {
		return fmt.Errorf("signed tx is empty, txid=%v", txid)
	}

	txHash, err := bridge.SendTransaction(signedTx)

	if err != nil {
		logWorkerError("swapin", "update swapin status to TxSwapFailed", err, "txid", txid)
		err = mongodb.UpdateSwapinStatus(txid, mongodb.TxSwapFailed, now(), "")
		return err
	}

	addSwapHistory(txid, txHash, true)

	mongodb.UpdateSwapinStatus(txid, mongodb.TxProcessed, now(), "")

	matchTx := &MatchTx{
		SwapTx:    txHash,
		SwapValue: tokens.CalcSwappedValue(value, bridge.IsSrcEndpoint()).String(),
	}
	return updateSwapinResult(txid, matchTx)
}

func processSwapoutSwap(swap *mongodb.MgoSwap) (err error) {
	txid := swap.TxId
	log.Debug("start processSwapoutSwap", "txid", txid, "status", swap.Status)
	history := getSwapHistory(txid, false)
	if history != nil {
		logWorker("swapout", "ignore swapped swapout", "txid", txid, "matchTx", history.matchTx)
		return fmt.Errorf("found swapped out history, txid=%v, matchTx=%v", txid, history.matchTx)
	}
	res, err := mongodb.FindSwapoutResult(txid)
	if err != nil {
		return err
	}
	if res.SwapTx != "" {
		if res.Status == mongodb.TxNotSwapped {
			mongodb.UpdateSwapoutStatus(txid, mongodb.TxProcessed, now(), "")
		}
		return fmt.Errorf("%v already swapped to %v", txid, res.SwapTx)
	}

	value, err := common.GetBigIntFromStr(res.Value)
	if err != nil {
		return fmt.Errorf("wrong value %v", res.Value)
	}

	args := &tokens.BuildTxArgs{
		SwapInfo: tokens.SwapInfo{
			SwapID:   res.TxId,
			SwapType: tokens.Swap_Swapout,
		},
		To:    res.Bind,
		Value: value,
		Memo:  fmt.Sprintf("%s%s", tokens.UnlockMemoPrefix, res.TxId),
	}
	bridge := tokens.SrcBridge
	rawTx, err := bridge.BuildRawTransaction(args)
	if err != nil {
		return err
	}
	if rawTx == nil {
		return fmt.Errorf("build raw tx is empty, txid=%v", txid)
	}

	signedTx, err := bridge.DcrmSignTransaction(rawTx, args.GetExtraArgs())
	if err != nil {
		return err
	}
	if signedTx == nil {
		return fmt.Errorf("signed tx is empty, txid=%v", txid)
	}

	txHash, err := bridge.SendTransaction(signedTx)

	if err != nil {
		logWorkerError("swapout", "update swapout status to TxSwapFailed", err, "txid", txid)
		err = mongodb.UpdateSwapoutStatus(txid, mongodb.TxSwapFailed, now(), "")
		return err
	}

	addSwapHistory(txid, txHash, false)

	mongodb.UpdateSwapoutStatus(txid, mongodb.TxProcessed, now(), "")

	matchTx := &MatchTx{
		SwapTx:    txHash,
		SwapValue: tokens.CalcSwappedValue(value, bridge.IsSrcEndpoint()).String(),
	}
	return updateSwapoutResult(txid, matchTx)
}

type swapInfo struct {
	txid     string
	matchTx  string
	isSwapin bool
}

func addSwapHistory(txid, matchTx string, isSwapin bool) {
	// Create the new item as its own ring
	item := ring.New(1)
	item.Value = &swapInfo{
		txid:     txid,
		matchTx:  matchTx,
		isSwapin: isSwapin,
	}

	swapRingLock.Lock()
	defer swapRingLock.Unlock()

	if swapRing == nil {
		swapRing = item
	} else {
		if swapRing.Len() == swapRingMaxSize {
			// Drop the block out of the ring
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
