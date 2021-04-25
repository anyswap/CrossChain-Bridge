package worker

import (
	"fmt"
	"time"

	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var (
	defWaitTimeToReplace = int64(900) // seconds
	defMaxReplaceCount   = 20
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
	if !tokens.DstBridge.GetChainConfig().EnableReplaceSwap {
		logWorker("replace", "stop replace swapin job as disabled")
		return
	}
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
	if !tokens.SrcBridge.GetChainConfig().EnableReplaceSwap {
		logWorker("replace", "stop replace swapout job as disabled")
		return
	}
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
	return processReplaceSwap(swap, true)
}

func processSwapoutReplace(swap *mongodb.MgoSwapResult) error {
	return processReplaceSwap(swap, false)
}

func processReplaceSwap(swap *mongodb.MgoSwapResult, isSwapin bool) (err error) {
	var chainCfg *tokens.ChainConfig
	if isSwapin {
		chainCfg = tokens.DstBridge.GetChainConfig()
	} else {
		chainCfg = tokens.SrcBridge.GetChainConfig()
	}
	waitTimeToReplace := chainCfg.WaitTimeToReplace
	maxReplaceCount := chainCfg.MaxReplaceCount
	if waitTimeToReplace == 0 {
		waitTimeToReplace = defWaitTimeToReplace
	}
	if maxReplaceCount == 0 {
		maxReplaceCount = defMaxReplaceCount
	}
	if len(swap.OldSwapTxs) > maxReplaceCount {
		return fmt.Errorf("replace swap too many times (> %v)", maxReplaceCount)
	}
	if getSepTimeInFind(waitTimeToReplace) < swap.InitTime {
		return nil
	}
	bridge := tokens.GetCrossChainBridge(!isSwapin)
	nonceSetter, ok := bridge.(tokens.NonceSetter)
	if !ok {
		return nil
	}
	var txHash string
	for {
		if isSwapin {
			txHash, err = ReplaceSwapin(swap.TxID, swap.PairID, swap.Bind, "")
		} else {
			txHash, err = ReplaceSwapout(swap.TxID, swap.PairID, swap.Bind, "")
		}
		if txHash != "" {
			if checkTxIsPacked(nonceSetter, txHash, waitTimeToReplace/5+1) {
				return nil
			}
		} else {
			switch err {
			case errSwapTxWithHeight,
				errSwapTxIsOnChain,
				errWrongResultStatus,
				errSwapNoncePassed:
				logWorkerTrace("replace", "jump swap", "pairID", swap.PairID, "txid", swap.TxID, "bind", swap.Bind, "isSwapin", isSwapin, "err", err)
				return nil
			case errSwapWithoutSwapTx:
			default:
				logWorkerTrace("replace", "replace swap error", "pairID", swap.PairID, "txid", swap.TxID, "bind", swap.Bind, "isSwapin", isSwapin, "err", err)
			}
			time.Sleep(60 * time.Second)
		}
	}
}

func checkTxIsPacked(bridge tokens.NonceSetter, txHash string, loopCount int64) bool {
	for i := int64(0); i < loopCount; i++ {
		if isTransactionOnChain(bridge, txHash) {
			return true
		}
		time.Sleep(5 * time.Second)
	}
	return false
}

func isTransactionOnChain(bridge tokens.NonceSetter, txHash string) bool {
	blockHeight, _ := bridge.GetTxBlockInfo(txHash)
	return blockHeight > 0
}
