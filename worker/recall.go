package worker

import (
	"sync"

	"github.com/fsn-dev/crossChain-Bridge/mongodb"
)

var (
	swapinRecallStarter sync.Once
)

func StartRecallJob() error {
	go startSwapinRecallJob()
	return nil
}

func startSwapinRecallJob() error {
	swapinRecallStarter.Do(func() {
		logWorker("recall", "start swapin recall job")
		for {
			res, err := findSwapinsToRecall()
			if err != nil {
				logWorkerError("recall", "find recalls error", err)
			}
			for _, swap := range res {
				err = processRecallSwapin(swap)
				if err != nil {
					logWorkerError("recall", "process recall error", err)
				}
			}
			restInJob(restIntervalInRecallJob)
		}
	})
	return nil
}

func findSwapinsToRecall() ([]*mongodb.MgoSwap, error) {
	status := mongodb.TxToBeRecall
	septime := getSepTimeInFind(maxRecallLifetime)
	return mongodb.FindSwapinsWithStatus(status, septime)
}

func processRecallSwapin(swap *mongodb.MgoSwap) (err error) {
	recallFailed := true
	txid := swap.TxId
	// TODO
	// build rawtx of swap
	// dcrm sign rawtx
	// broadcast signtx
	// update database

	if recallFailed {
		err = mongodb.UpdateSwapinStatus(txid, mongodb.TxRecallFailed, now(), "")
		return err
	}

	err = mongodb.UpdateSwapinStatus(txid, mongodb.TxProcessed, now(), "")

	matchTx := &MatchTx{
		IsRecall: true,
	}
	updateSwapinResult(txid, matchTx)
	return err
}
