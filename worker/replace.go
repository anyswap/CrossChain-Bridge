package worker

import (
	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var (
	waitTimeToReplace = int64(600) // seconds
)

// StartReplaceJob replace job
func StartReplaceJob() {
	var ok bool
	_, ok = tokens.DstBridge.(tokens.NonceSetter)
	if ok {
		go startReplaceSwapinJob()
	}

	_, ok = tokens.SrcBridge.(tokens.NonceSetter)
	if ok {
		go startReplaceSwapoutJob()
	}
}

func startReplaceSwapinJob() {
	logWorker("replace", "start replace swapin job")
	for {
		res, err := findSwapinsToReplace()
		if err != nil {
			logWorkerError("replace", "find swapins error", err)
		}
		for _, swap := range res {
			err = processSwapinReplace(swap)
			if err != nil {
				logWorkerError("replace", "process swapin replace error", err, "pairID", swap.PairID, "txid", swap.TxID, "bind", swap.Bind)
			}
		}
		restInJob(restIntervalInReplaceSwapJob)
	}
}

func startReplaceSwapoutJob() {
	logWorker("replace", "start replace swapout job")
	for {
		res, err := findSwapoutsToReplace()
		if err != nil {
			logWorkerError("replace", "find swapouts error", err)
		}
		for _, swap := range res {
			err = processSwapoutReplace(swap)
			if err != nil {
				logWorkerError("replace", "process swapout replace error", err, "pairID", swap.PairID, "txid", swap.TxID, "bind", swap.Bind)
			}
		}
		restInJob(restIntervalInReplaceSwapJob)
	}
}

func findSwapinsToReplace() ([]*mongodb.MgoSwapResult, error) {
	status := mongodb.MatchTxNotStable
	septime := getSepTimeInFind(maxReplaceSwapLifetime)
	return mongodb.FindSwapinResultsWithStatus(status, septime)
}

func findSwapoutsToReplace() ([]*mongodb.MgoSwapResult, error) {
	status := mongodb.MatchTxNotStable
	septime := getSepTimeInFind(maxReplaceSwapLifetime)
	return mongodb.FindSwapoutResultsWithStatus(status, septime)
}

func processSwapinReplace(swap *mongodb.MgoSwapResult) error {
	if getSepTimeInFind(waitTimeToReplace) < swap.Timestamp {
		return nil
	}
	return ReplaceSwapin(swap.TxID, swap.PairID, swap.Bind, "")
}

func processSwapoutReplace(swap *mongodb.MgoSwapResult) error {
	if getSepTimeInFind(waitTimeToReplace) < swap.Timestamp {
		return nil
	}
	return ReplaceSwapout(swap.TxID, swap.PairID, swap.Bind, "")
}
