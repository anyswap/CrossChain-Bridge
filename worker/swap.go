package worker

import (
	"sync"

	"github.com/fsn-dev/crossChain-Bridge/mongodb"
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
	swapFail := true
	txid := swap.TxId
	// TODO
	// build rawtx of swap
	// dcrm sign rawtx
	// broadcast signtx
	// update database

	if swapFail {
		err = mongodb.UpdateSwapinStatus(txid, mongodb.TxSwapFailed, now(), "")
		return err
	}

	err = mongodb.UpdateSwapinStatus(txid, mongodb.TxProcessed, now(), "")

	matchTx := &MatchTx{}
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
