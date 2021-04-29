package worker

import (
	"sync"
	"time"

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
				time.Sleep(3 * time.Second) // in case of too frequently rpc calling
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

func getSwapTxStatus(resBridge tokens.CrossChainBridge, swap *mongodb.MgoSwapResult) *tokens.TxStatus {
	txStatus := resBridge.GetTransactionStatus(swap.SwapTx)
	//if txStatus != nil && txStatus.BlockHeight > 0 {
	if txStatus != nil {
		return txStatus
	}
	for _, oldSwapTx := range swap.OldSwapTxs {
		if swap.SwapTx == oldSwapTx {
			continue
		}
		txStatus = resBridge.GetTransactionStatus(oldSwapTx)
		if txStatus != nil && txStatus.BlockHeight > 0 {
			swap.SwapTx = oldSwapTx
			return txStatus
		}
	}
	return nil
}

func processSwapStable(swap *mongodb.MgoSwapResult, isSwapin bool) (err error) {
	oldSwapTx := swap.SwapTx
	resBridge := tokens.GetCrossChainBridge(!isSwapin)
	txStatus := getSwapTxStatus(resBridge, swap)
	if txStatus.CustomeCheckStable != nil {
		// Custome check stable logic
		// For blockchains like Tron, which can tell tx is conditionally finailized
		// but only cannot provide an Eth style RPCTxReceipt
		switch txStatus.CustomeCheckStable(*resBridge.GetChainConfig().Confirmations) {
		case 0:
			// stable
			logWorker("transaction is stable", "txid", swap.TxID)
			return markSwapResultStable(swap.TxID, swap.PairID, swap.Bind, isSwapin)
		case  1:
			// fail
			logWorker("transaction failed", "txid", swap.TxID)
			return markSwapResultFailed(swap.TxID, swap.PairID, swap.Bind, isSwapin)
		default:
			// unstable
			logWorker("transaction is unstable", "txid", swap.TxID)
			return nil
		}
	} 
	if txStatus == nil || txStatus.BlockHeight == 0 {
		if swap.SwapHeight == 0 {
			return processUpdateSwapHeight(resBridge, swap)
		}
		return nil
	}
	if txStatus != nil && txStatus.PrioriFinalized {
		// For blockchains like Cosmos, which can tell tx is finailized at this stage
		return markSwapResultStable(swap.TxID, swap.PairID, swap.Bind, isSwapin)
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
		for _, oldSwapTx := range swap.OldSwapTxs {
			if swap.SwapTx == oldSwapTx {
				continue
			}
			blockHeight, blockTime = nonceSetter.GetTxBlockInfo(oldSwapTx)
			if blockHeight > 0 {
				swap.SwapTx = oldSwapTx
				break
			}
		}
	}
	if blockHeight == 0 {
		return nil
	}
	return updateSwapResultHeight(swap, blockHeight, blockTime, swap.SwapTx != oldSwapTx)
}
