package worker

import (
	"sync"
	"time"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var (
	swapinStableStarter  sync.Once
	swapoutStableStarter sync.Once

	treatAsNoncePassedInterval = int64(600) // seconds
)

// StartStableJob stable job
func StartStableJob() {
	mongodb.MgoWaitGroup.Add(2)
	go startSwapinStableJob()
	go startSwapoutStableJob()
}

func startSwapinStableJob() {
	swapinStableStarter.Do(func() {
		logWorker("stable", "start update swapin stable job")
		defer mongodb.MgoWaitGroup.Done()
		for {
			res, err := findSwapinResultsToStable()
			if err != nil {
				logWorkerError("stable", "find swapin results error", err)
			}
			if len(res) > 0 {
				logWorker("stable", "find swapin results to stable", "count", len(res))
			}
			for _, swap := range res {
				if utils.IsCleanuping() {
					logWorker("stable", "stop update swapin stable job")
					return
				}
				err = processSwapinStable(swap)
				if err != nil {
					logWorkerError("stable", "process swapin stable error", err)
				}
				time.Sleep(3 * time.Second) // in case of too frequently rpc calling
			}
			if utils.IsCleanuping() {
				logWorker("stable", "stop update swapin stable job")
				return
			}
			restInJob(restIntervalInStableJob)
		}
	})
}

func startSwapoutStableJob() {
	swapoutStableStarter.Do(func() {
		logWorker("stable", "start update swapout stable job")
		defer mongodb.MgoWaitGroup.Done()
		for {
			res, err := findSwapoutResultsToStable()
			if err != nil {
				logWorkerError("stable", "find swapout results error", err)
			}
			if len(res) > 0 {
				logWorker("stable", "find swapout results to stable", "count", len(res))
			}
			for _, swap := range res {
				if utils.IsCleanuping() {
					logWorker("stable", "stop update swapout stable job")
					return
				}
				err = processSwapoutStable(swap)
				if err != nil {
					logWorkerError("stable", "process swapout stable error", err)
				}
			}
			if utils.IsCleanuping() {
				logWorker("stable", "stop update swapout stable job")
				return
			}
			restInJob(restIntervalInStableJob)
		}
	})
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

func processSwapinStable(swap *mongodb.MgoSwapResult) error {
	logWorker("stable", "start processSwapinStable", "swaptxid", swap.SwapTx, "txid", swap.TxID, "bind", swap.Bind, "status", swap.Status)
	return processSwapStable(swap, true)
}

func processSwapoutStable(swap *mongodb.MgoSwapResult) (err error) {
	logWorker("stable", "start processSwapoutStable", "swaptxid", swap.SwapTx, "txid", swap.TxID, "bind", swap.Bind, "status", swap.Status)
	return processSwapStable(swap, false)
}

func getSwapTxStatus(resBridge tokens.CrossChainBridge, swap *mongodb.MgoSwapResult) *tokens.TxStatus {
	txStatus, err := resBridge.GetTransactionStatus(swap.SwapTx)
	if err == nil && txStatus != nil && txStatus.BlockHeight > 0 {
		return txStatus
	}
	for i, oldSwapTx := range swap.OldSwapTxs {
		if swap.SwapTx == oldSwapTx {
			continue
		}
		txStatus, err = resBridge.GetTransactionStatus(oldSwapTx)
		if err == nil && txStatus != nil && txStatus.BlockHeight > 0 {
			swap.SwapTx = oldSwapTx
			swap.SwapValue = swap.OldSwapVals[i]
			return txStatus
		}
	}
	return nil
}

func processSwapStable(swap *mongodb.MgoSwapResult, isSwapin bool) (err error) {
	oldSwapTx := swap.SwapTx
	resBridge := tokens.GetCrossChainBridge(!isSwapin)
	txStatus := getSwapTxStatus(resBridge, swap)
	if txStatus == nil || txStatus.BlockHeight == 0 {
		if swap.SwapHeight == 0 {
			return processUpdateSwapHeight(resBridge, swap)
		}
		return nil
	}

	if swap.SwapHeight != 0 {
		if txStatus.Confirmations < *resBridge.GetChainConfig().Confirmations {
			return nil
		}
		if swap.SwapTx != oldSwapTx {
			_ = updateSwapResultTx(swap.TxID, swap.PairID, swap.Bind, swap.SwapTx, swap.SwapValue, isSwapin, mongodb.KeepStatus)
		}
		if txStatus.IsSwapTxOnChainAndFailed(resBridge.GetTokenConfig(swap.PairID)) {
			logWorkerWarn("stable", "mark swap result failed with wrong status", "pairID", swap.PairID, "txid", swap.TxID, "bind", swap.Bind, "isSwapin", isSwapin, "swaptime", swap.Timestamp, "nowtime", now(), "confirmations", txStatus.Confirmations)
			return markSwapResultFailed(swap.TxID, swap.PairID, swap.Bind, isSwapin)
		}
		return markSwapResultStable(swap.TxID, swap.PairID, swap.Bind, isSwapin)
	}

	return updateSwapResultHeight(swap, txStatus.BlockHeight, txStatus.BlockTime, swap.SwapTx != oldSwapTx)
}

func processUpdateSwapHeight(resBridge tokens.CrossChainBridge, swap *mongodb.MgoSwapResult) (err error) {
	nonceSetter, ok := resBridge.(tokens.NonceSetter)
	if !ok {
		return nil
	}

	oldSwapTx := swap.SwapTx
	blockHeight, blockTime := nonceSetter.GetTxBlockInfo(swap.SwapTx)
	if blockHeight == 0 {
		for i, oldSwapTx := range swap.OldSwapTxs {
			if swap.SwapTx == oldSwapTx {
				continue
			}
			blockHeight, blockTime = nonceSetter.GetTxBlockInfo(oldSwapTx)
			if blockHeight > 0 {
				swap.SwapTx = oldSwapTx
				swap.SwapValue = swap.OldSwapVals[i]
				break
			}
		}
	}
	if blockHeight == 0 && swap.SwapNonce > 0 {
		return checkIfSwapNonceHasPassed(resBridge, swap, false)
	}
	return updateSwapResultHeight(swap, blockHeight, blockTime, swap.SwapTx != oldSwapTx)
}
