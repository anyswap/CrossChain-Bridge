package worker

import (
	"fmt"
	"sync"

	"github.com/fsn-dev/crossChain-Bridge/log"
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
			if len(res) > 0 {
				logWorker("stable", "find swapout results to stable", "count", len(res))
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
	swapTxId := swap.SwapTx
	log.Debug("start processSwapinStable", "swaptxid", swapTxId, "status", swap.Status)
	var (
		txStatus      *tokens.TxStatus
		confirmations uint64
	)
	if swap.SwapType == uint32(tokens.Swap_Recall) {
		txStatus, _ = tokens.SrcBridge.GetTransactionStatus(swapTxId)
		token, _ := tokens.SrcBridge.GetTokenAndGateway()
		confirmations = *token.Confirmations
	} else {
		txStatus, _ = tokens.DstBridge.GetTransactionStatus(swapTxId)
		token, _ := tokens.DstBridge.GetTokenAndGateway()
		confirmations = *token.Confirmations
	}

	if txStatus == nil {
		return fmt.Errorf("[processSwapinStable] tx status is empty, swapTxId=%v", swapTxId)
	}

	if txStatus.Block_height == 0 {
		return nil
	}

	if swap.SwapHeight != 0 {
		if txStatus.Confirmations >= confirmations {
			return markSwapinResultStable(swap.Key)
		}
		return nil
	}

	matchTx := &MatchTx{
		SwapHeight: txStatus.Block_height,
		SwapTime:   txStatus.Block_time,
		SwapType:   tokens.Swap_Swapin,
	}
	return updateSwapinResult(swap.Key, matchTx)
}

func processSwapoutStable(swap *mongodb.MgoSwapResult) (err error) {
	swapTxId := swap.SwapTx
	log.Debug("start processSwapoutStable", "swaptxid", swapTxId, "status", swap.Status)

	var txStatus *tokens.TxStatus
	var confirmations uint64

	txStatus, _ = tokens.SrcBridge.GetTransactionStatus(swapTxId)
	token, _ := tokens.SrcBridge.GetTokenAndGateway()
	confirmations = *token.Confirmations

	if txStatus == nil {
		return fmt.Errorf("[processSwapoutStable] tx status is empty, swapTxId=%v", swapTxId)
	}

	if txStatus.Block_height == 0 {
		return nil
	}

	if swap.SwapHeight != 0 {
		if txStatus.Confirmations >= confirmations {
			return markSwapoutResultStable(swap.Key)
		}
		return nil
	}

	matchTx := &MatchTx{
		SwapHeight: txStatus.Block_height,
		SwapTime:   txStatus.Block_time,
		SwapType:   tokens.Swap_Swapout,
	}
	return updateSwapoutResult(swap.Key, matchTx)
}
