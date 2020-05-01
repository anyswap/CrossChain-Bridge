package worker

import (
	"sync"

	"github.com/fsn-dev/crossChain-Bridge/mongodb"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
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

func processSwapinStable(swap *mongodb.MgoSwapResult) error {
	txid := swap.SwapTx
	var txStatus *tokens.TxStatus
	var confirmations uint64
	if swap.Memo == RecallTxMemo {
		txStatus = tokens.SrcBridge.GetTransactionStatus(txid)
		token, _ := tokens.SrcBridge.GetTokenAndGateway()
		confirmations = *token.Confirmations
	} else {
		txStatus = tokens.DstBridge.GetTransactionStatus(txid)
		token, _ := tokens.DstBridge.GetTokenAndGateway()
		confirmations = *token.Confirmations
	}

	if txStatus.Block_height == 0 {
		return nil
	}

	if swap.SwapHeight != 0 {
		if txStatus.Confirmations >= confirmations {
			return markSwapinResultStable(txid)
		}
		return nil
	}

	matchTx := &MatchTx{
		SwapTx:     txid,
		SwapHeight: txStatus.Block_height,
		SwapTime:   txStatus.Block_time,
	}
	return updateSwapinResult(txid, matchTx)
}

func processSwapoutStable(swap *mongodb.MgoSwapResult) (err error) {
	txid := swap.SwapTx
	var txStatus *tokens.TxStatus
	var confirmations uint64

	txStatus = tokens.SrcBridge.GetTransactionStatus(txid)
	token, _ := tokens.SrcBridge.GetTokenAndGateway()
	confirmations = *token.Confirmations

	if txStatus.Block_height == 0 {
		return nil
	}

	if swap.SwapHeight != 0 {
		if txStatus.Confirmations >= confirmations {
			return markSwapoutResultStable(txid)
		}
		return nil
	}

	matchTx := &MatchTx{
		SwapTx:     txid,
		SwapHeight: txStatus.Block_height,
		SwapTime:   txStatus.Block_time,
	}
	return updateSwapoutResult(txid, matchTx)
}
