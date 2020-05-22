package worker

import (
	"sync"

	"github.com/fsn-dev/crossChain-Bridge/mongodb"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
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
			if len(res) > 0 {
				logWorker("verify", "find swapins to verify", "count", len(res))
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
			if len(res) > 0 {
				logWorker("verify", "find swapouts to verify", "count", len(res))
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

func processSwapinVerify(swap *mongodb.MgoSwap) error {
	txid := swap.TxId
	swapInfo, err := tokens.SrcBridge.VerifyTransaction(txid, false)

	resultStatus := mongodb.MatchTxEmpty

	switch err {
	case tokens.ErrTxNotStable, tokens.ErrTxNotFound:
		logWorkerError("verify", "processSwapinVerify", err, "txid", txid)
		return err
	case tokens.ErrTxWithWrongMemo:
		resultStatus = mongodb.TxWithWrongMemo
		err = mongodb.UpdateSwapinStatus(txid, mongodb.TxCanRecall, now(), "")
	case nil:
		err = mongodb.UpdateSwapinStatus(txid, mongodb.TxNotSwapped, now(), "")
	default:
		return mongodb.UpdateSwapinStatus(txid, mongodb.TxVerifyFailed, now(), "")
	}

	if err != nil {
		logWorkerError("verify", "processSwapinVerify", err, "txid", txid)
		return err
	}
	return addInitialSwapinResult(swapInfo, resultStatus)
}

func processSwapoutVerify(swap *mongodb.MgoSwap) error {
	txid := swap.TxId
	swapInfo, err := tokens.DstBridge.VerifyTransaction(txid, false)

	resultStatus := mongodb.MatchTxEmpty

	switch err {
	case tokens.ErrTxNotStable, tokens.ErrTxNotFound:
		logWorkerError("verify", "processSwapoutVerify", err, "txid", txid)
		return err
	case tokens.ErrTxWithWrongMemo:
		resultStatus = mongodb.TxWithWrongMemo
		err = mongodb.UpdateSwapoutStatus(txid, mongodb.TxCanRecall, now(), "")
	case nil:
		err = mongodb.UpdateSwapoutStatus(txid, mongodb.TxNotSwapped, now(), "")
	default:
		return mongodb.UpdateSwapoutStatus(txid, mongodb.TxVerifyFailed, now(), "")
	}

	if err != nil {
		logWorkerError("verify", "processSwapoutVerify", err, "txid", txid)
		return err
	}
	return addInitialSwapoutResult(swapInfo, resultStatus)
}
