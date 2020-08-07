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
	swapTxID := swap.SwapTx
	logWorker("stable", "start processSwapinStable", "swaptxid", swapTxID, "status", swap.Status)
	var (
		token         *tokens.TokenConfig
		txStatus      *tokens.TxStatus
		confirmations uint64
	)
	if swap.SwapType == uint32(tokens.SwapRecallType) {
		txStatus = tokens.SrcBridge.GetTransactionStatus(swapTxID)
		token, _ = tokens.SrcBridge.GetTokenAndGateway()
		confirmations = *token.Confirmations
	} else {
		txStatus = tokens.DstBridge.GetTransactionStatus(swapTxID)
		token, _ = tokens.DstBridge.GetTokenAndGateway()
		confirmations = *token.Confirmations
	}

	if txStatus == nil || txStatus.BlockHeight == 0 {
		return nil
	}

	if swap.SwapHeight != 0 {
		if txStatus.Confirmations < confirmations {
			return nil
		}
		receipt, ok := txStatus.Receipt.(*types.RPCTxReceipt)
		txFailed := !ok || receipt == nil || *receipt.Status != 1
		if !txFailed && token.ContractAddress != "" && len(receipt.Logs) == 0 {
			txFailed = true
		}
		if txFailed {
			return markSwapinResultFailed(swap.Key)
		}
		return markSwapinResultStable(swap.Key)
	}

	matchTx := &MatchTx{
		SwapHeight: txStatus.BlockHeight,
		SwapTime:   txStatus.BlockTime,
		SwapType:   tokens.SwapinType,
	}
	return updateSwapinResult(swap.Key, matchTx)
}

func processSwapoutStable(swap *mongodb.MgoSwapResult) (err error) {
	swapTxID := swap.SwapTx
	logWorker("stable", "start processSwapoutStable", "swaptxid", swapTxID, "status", swap.Status)

	var txStatus *tokens.TxStatus
	var confirmations uint64

	txStatus = tokens.SrcBridge.GetTransactionStatus(swapTxID)
	token, _ := tokens.SrcBridge.GetTokenAndGateway()
	confirmations = *token.Confirmations

	if txStatus == nil || txStatus.BlockHeight == 0 {
		return nil
	}

	if swap.SwapHeight != 0 {
		if txStatus.Confirmations < confirmations {
			return nil
		}
		receipt, ok := txStatus.Receipt.(*types.RPCTxReceipt)
		txFailed := !ok || receipt == nil || *receipt.Status != 1
		if !txFailed && token.ContractAddress != "" && len(receipt.Logs) == 0 {
			txFailed = true
		}
		if txFailed {
			return markSwapoutResultFailed(swap.Key)
		}
		return markSwapoutResultStable(swap.Key)
	}

	matchTx := &MatchTx{
		SwapHeight: txStatus.BlockHeight,
		SwapTime:   txStatus.BlockTime,
		SwapType:   tokens.SwapoutType,
	}
	return updateSwapoutResult(swap.Key, matchTx)
}
