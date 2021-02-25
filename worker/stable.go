package worker

import (
	"sync"

	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/types"
)

var (
	swapinStableStarter  sync.Once
	swapoutStableStarter sync.Once
)

// StartStableJob stable job
func StartStableJob() {
	go startSwapinStableJob()
	go startSwapoutStableJob()
}

func startSwapinStableJob() {
	swapinStableStarter.Do(func() {
		logWorker("stable", "start update swapin stable job")
		for {
			res, err := findSwapinResultsToStable()
			if err != nil {
				logWorkerError("stable", "find swapin results error", err)
			}
			if len(res) > 0 {
				logWorker("stable", "find swapin results to stable", "count", len(res))
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
}

func startSwapoutStableJob() {
	swapoutStableStarter.Do(func() {
		logWorker("stable", "start update swapout stable job")
		for {
			res, err := findSwapoutResultsToStable()
			if err != nil {
				logWorkerError("stable", "find swapout results error", err)
			}
			if len(res) > 0 {
				logWorker("stable", "find swapout results to stable", "count", len(res))
			}
			for _, swap := range res {
				err = processSwapoutStable(swap)
				if err != nil {
					logWorkerError("stable", "process swapout stable error", err)
				}
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
	logWorker("stable", "start processSwapinStable", "swaptxid", swap.SwapTx, "bind", swap.Bind, "status", swap.Status)
	return processSwapStable(swap, true)
}

func processSwapoutStable(swap *mongodb.MgoSwapResult) (err error) {
	logWorker("stable", "start processSwapoutStable", "swaptxid", swap.SwapTx, "bind", swap.Bind, "status", swap.Status)
	return processSwapStable(swap, false)
}

func isTxOnChain(bridge tokens.CrossChainBridge, txStatus *tokens.TxStatus) bool {
	if txStatus == nil || txStatus.BlockHeight == 0 {
		return false
	}
	// only consider eth like tx
	_, ok := bridge.(tokens.NonceSetter)
	if !ok {
		return true
	}
	return txStatus.Receipt != nil
}

func getSwapTxStatus(resBridge tokens.CrossChainBridge, swap *mongodb.MgoSwapResult) *tokens.TxStatus {
	txStatus := resBridge.GetTransactionStatus(swap.SwapTx)
	if isTxOnChain(resBridge, txStatus) {
		return txStatus
	}
	for _, oldSwapTx := range swap.OldSwapTxs {
		if swap.SwapTx == oldSwapTx {
			continue
		}
		txStatus2 := resBridge.GetTransactionStatus(oldSwapTx)
		if isTxOnChain(resBridge, txStatus2) {
			swap.SwapTx = oldSwapTx
			return txStatus2
		}
	}
	return txStatus
}

func processSwapStable(swap *mongodb.MgoSwapResult, isSwapin bool) (err error) {
	oldSwapTx := swap.SwapTx
	resBridge := tokens.GetCrossChainBridge(!isSwapin)
	txStatus := getSwapTxStatus(resBridge, swap)
	if txStatus == nil || txStatus.BlockHeight == 0 {
		return nil
	}

	if swap.SwapHeight != 0 {
		if txStatus.Confirmations < *resBridge.GetChainConfig().Confirmations {
			return nil
		}
		if swap.SwapTx != oldSwapTx {
			_ = updateSwapTx(swap.TxID, swap.PairID, swap.Bind, swap.SwapTx, isSwapin)
		}
		if txStatus.Receipt != nil {
			receipt, ok := txStatus.Receipt.(*types.RPCTxReceipt)
			txFailed := !ok || receipt == nil || *receipt.Status != 1
			token := resBridge.GetTokenConfig(swap.PairID)
			if !txFailed && token != nil && token.ContractAddress != "" && len(receipt.Logs) == 0 {
				txFailed = true
			}
			if txFailed {
				return markSwapResultFailed(swap.TxID, swap.PairID, swap.Bind, isSwapin)
			}
		}
		return markSwapResultStable(swap.TxID, swap.PairID, swap.Bind, isSwapin)
	}

	matchTx := &MatchTx{
		SwapHeight: txStatus.BlockHeight,
		SwapTime:   txStatus.BlockTime,
		SwapType:   getSwapType(isSwapin),
	}
	if swap.SwapTx != oldSwapTx {
		matchTx.SwapTx = swap.SwapTx
	}
	return updateSwapResult(swap.TxID, swap.PairID, swap.Bind, matchTx)
}
