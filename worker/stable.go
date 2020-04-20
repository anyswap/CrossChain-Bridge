package worker

import (
	"sync"

	"github.com/fsn-dev/crossChain-Bridge/mongodb"
)

var (
	swapinStableStarter  sync.Once
	swapoutStableStarter sync.Once
)

func StartStableJob() error {
	go startSwapinStableJob()
	go startSwapoutStableJob()
	return nil
}

func startSwapinStableJob() error {
	swapinStableStarter.Do(func() {
		logWorker("stable", "start update swapin stable job")
		for {
			res, err := findSwapinResultsToStable()
			if err != nil {
				logWorkerError("stable", "find swapin results error", err)
			}
			for _, swap := range res {
				err = processSwapinStable(swap)
				if err != nil {
					logWorkerError("stable", "process swapin stable error", err)
				}
			}
			restInJob(restIntervalInStableJob)
		}
	})
	return nil
}

func startSwapoutStableJob() error {
	swapoutStableStarter.Do(func() {
		logWorker("stable", "start update swapout stable job")
		for {
			res, err := findSwapoutResultsToStable()
			if err != nil {
				logWorkerError("stable", "find swapout results error", err)
			}
			for _, swap := range res {
				err = processSwapoutStable(swap)
				if err != nil {
					logWorkerError("recall", "process swapout stable error", err)
				}
			}
			restInJob(restIntervalInStableJob)
		}
	})
	return nil
}

func findSwapinResultsToStable() ([]*mongodb.MgoSwapResult, error) {
	status := mongodb.MatchTxNotStable
	septime := getSepTimeInFind(maxStableLifetime)
	return mongodb.FindSwapinResultsWithStatus(status, septime)
}

func findSwapoutResultsToStable() ([]*mongodb.MgoSwapResult, error) {
	status := mongodb.MatchTxNotStable
	septime := getSepTimeInFind(maxStableLifetime)
	return mongodb.FindSwapoutResultsWithStatus(status, septime)
}

func processSwapinStable(swap *mongodb.MgoSwapResult) (err error) {
	isStable := false
	txid := swap.TxId
	// TODO
	// get txid tx
	// verify tx
	// update database

	if !isStable {
		return nil
	}

	err = markSwapinResultStable(txid)
	return err
}

func processSwapoutStable(swap *mongodb.MgoSwapResult) (err error) {
	isStable := false
	txid := swap.TxId
	// TODO
	// get txid tx
	// verify tx
	// update database

	if !isStable {
		return nil
	}

	err = markSwapoutResultStable(txid)
	return err
}
