package worker

import (
	"fmt"
	"sync"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/mongodb"
	. "github.com/fsn-dev/crossChain-Bridge/tokens"
)

var (
	swapinSwapStarter  sync.Once
	swapoutSwapStarter sync.Once
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
				logWorkerError("swap", "find swapins error", err)
			}
			for _, swap := range res {
				err = processSwapinSwap(swap)
				if err != nil {
					logWorkerError("swap", "process swapin swap error", err)
				}
			}
			restInJob(restIntervalInDoSwapJob)
		}
	})
	return nil
}

func startSwapoutSwapJob() error {
	swapoutSwapStarter.Do(func() {
		logWorker("swap", "start swapout swap job")
		for {
			res, err := findSwapoutsToSwap()
			if err != nil {
				logWorkerError("swap", "find swapouts error", err)
			}
			for _, swap := range res {
				err = processSwapoutSwap(swap)
				if err != nil {
					logWorkerError("swap", "process swapout swap error", err)
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
	res, err := mongodb.FindSwapinResult(txid)
	if err != nil {
		return err
	}

	value, err := common.GetBigIntFromStr(res.Value)
	if err != nil {
		return fmt.Errorf("wrong value %v", res.Value)
	}

	args := &BuildTxArgs{
		IsSwapin: true,
		To:       res.Bind,
		Value:    value,
		Memo:     res.TxId,
	}
	rawTx, err := DstBridge.BuildRawTransaction(args)
	if err != nil {
		return err
	}

	signedTx, err := DstBridge.DcrmSignTransaction(rawTx)
	if err != nil {
		return err
	}

	txHash, err := DstBridge.SendTransaction(signedTx)

	if err != nil {
		err = mongodb.UpdateSwapinStatus(txid, mongodb.TxSwapFailed, now(), "")
		return err
	}

	err = mongodb.UpdateSwapinStatus(txid, mongodb.TxProcessed, now(), "")

	matchTx := &MatchTx{
		SwapTx:     txHash,
		SwapHeight: 0,
		SwapTime:   0,
		IsRecall:   false,
	}
	updateSwapinResult(txid, matchTx)
	return err
}

func processSwapoutSwap(swap *mongodb.MgoSwap) (err error) {
	swapFail := true
	txid := swap.TxId
	// TODO
	// build rawtx of swap
	// dcrm sign rawtx
	// broadcast signtx
	// update database

	if swapFail {
		err = mongodb.UpdateSwapoutStatus(txid, mongodb.TxSwapFailed, now(), "")
		return err
	}

	err = mongodb.UpdateSwapoutStatus(txid, mongodb.TxProcessed, now(), "")

	matchTx := &MatchTx{}
	updateSwapoutResult(txid, matchTx)
	return err
}
