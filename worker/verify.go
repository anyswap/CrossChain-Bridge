package worker

import (
	"sync"

	"github.com/fsn-dev/crossChain-Bridge/mongodb"
)

var (
	swapinVerifyStarter  sync.Once
	swapoutVerifyStarter sync.Once
)

func StartVerifyJob() error {
	go startSwapinVerifyJob()
	go startSwapoutVerifyJob()
	return nil
}

func startSwapinVerifyJob() error {
	swapinVerifyStarter.Do(func() {
		logWorker("verify", "start swapin verify job")
		for {
			res, err := findSwapinsToVerify()
			if err != nil {
				logWorkerError("verify", "find swapins error", err)
			}
			for _, swap := range res {
				err = processSwapinVerify(swap)
				if err != nil {
					logWorkerError("verify", "process swapin verify error", err)
				}
			}
			restInJob(restIntervalInVerifyJob)
		}
	})
	return nil
}

func startSwapoutVerifyJob() error {
	swapoutVerifyStarter.Do(func() {
		logWorker("verify", "start swapout verify job")
		for {
			res, err := findSwapoutsToVerify()
			if err != nil {
				logWorkerError("verify", "find swapouts error", err)
			}
			for _, swap := range res {
				err = processSwapoutVerify(swap)
				if err != nil {
					logWorkerError("recall", "process swapout verify error", err)
				}
			}
			restInJob(restIntervalInVerifyJob)
		}
	})
	return nil
}

func findSwapinsToVerify() ([]*mongodb.MgoSwap, error) {
	status := mongodb.TxNotStable
	septime := getSepTimeInFind(maxVerifyLifetime)
	return mongodb.FindSwapinsWithStatus(status, septime)
}

func findSwapoutsToVerify() ([]*mongodb.MgoSwap, error) {
	status := mongodb.TxNotStable
	septime := getSepTimeInFind(maxVerifyLifetime)
	return mongodb.FindSwapoutsWithStatus(status, septime)
}

func processSwapinVerify(swap *mongodb.MgoSwap) (err error) {
	var (
		verifyFail = false
		canRecall  = false
	)
	txid := swap.TxId
	// TODO
	// get txid tx
	// verify tx
	// update database

	if verifyFail {
		err = mongodb.UpdateSwapinStatus(txid, mongodb.TxVerifyFailed, now(), "")
		return err
	}

	if canRecall {
		err = mongodb.UpdateSwapinStatus(txid, mongodb.TxCanRecall, now(), "")
	} else {
		err = mongodb.UpdateSwapinStatus(txid, mongodb.TxNotSwapped, now(), "")
	}

	initialTx := &InitialTx{}
	addInitialSwapinResult(initialTx)
	return err
}

func processSwapoutVerify(swap *mongodb.MgoSwap) (err error) {
	verifyFail := true
	canRecall := false
	txid := swap.TxId
	// TODO
	// get txid tx
	// verify tx
	// update database

	if verifyFail {
		err = mongodb.UpdateSwapinStatus(txid, mongodb.TxVerifyFailed, now(), "")
		return err
	}

	if canRecall {
		err = mongodb.UpdateSwapinStatus(txid, mongodb.TxCanRecall, now(), "")
	} else {
		err = mongodb.UpdateSwapinStatus(txid, mongodb.TxNotSwapped, now(), "")
	}

	initialTx := &InitialTx{}
	addInitialSwapinResult(initialTx)
	return err
}
