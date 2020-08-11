package worker

import (
	"fmt"
	"sync"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var (
	swapinRecallStarter sync.Once
)

// StartRecallJob recall job
func StartRecallJob() {
	go startSwapinRecallJob()
}

func startSwapinRecallJob() {
	swapinRecallStarter.Do(func() {
		logWorker("recall", "start swapin recall job")
		for {
			res, err := findSwapinsToRecall()
			if err != nil {
				logWorkerError("recall", "find recalls error", err)
			}
			if len(res) > 0 {
				logWorker("recall", "find recalls to recall", "count", len(res))
			}
			for _, swap := range res {
				err = processRecallSwapin(swap)
				if err != nil {
					logWorkerError("recall", "process recall error", err, "txid", swap.TxID)
				}
			}
			restInJob(restIntervalInRecallJob)
		}
	})
}

func findSwapinsToRecall() ([]*mongodb.MgoSwap, error) {
	status := mongodb.TxToBeRecall
	septime := getSepTimeInFind(maxRecallLifetime)
	return mongodb.FindSwapinsWithStatus(status, septime)
}

func processRecallSwapin(swap *mongodb.MgoSwap) (err error) {
	txid := swap.TxID
	res, err := mongodb.FindSwapinResult(txid)
	if err != nil {
		return err
	}
	if tokens.GetTokenConfig(true).DisableSwap {
		logWorkerTrace("recall", "swapin recall is disabled")
		return nil
	}
	if res.SwapTx != "" {
		if res.Status == mongodb.TxToBeRecall {
			_ = mongodb.UpdateSwapinStatus(txid, mongodb.TxProcessed, now(), "")
		}
		return fmt.Errorf("%v already swapped to %v", txid, res.SwapTx)
	}

	value, err := common.GetBigIntFromStr(res.Value)
	if err != nil {
		return fmt.Errorf("wrong value %v", res.Value)
	}

	args := &tokens.BuildTxArgs{
		SwapInfo: tokens.SwapInfo{
			SwapID:   res.TxID,
			SwapType: tokens.SwapRecallType,
			TxType:   tokens.SwapTxType(swap.TxType),
			Bind:     swap.Bind,
		},
		To:    res.Bind,
		Value: value,
		Memo:  fmt.Sprintf("%s%s", tokens.RecallMemoPrefix, res.TxID),
	}
	bridge := tokens.SrcBridge
	rawTx, err := bridge.BuildRawTransaction(args)
	if err != nil {
		logWorkerError("recall", "BuildRawTransaction failed", err, "txid", txid)
		return err
	}

	signedTx, txHash, err := dcrmSignTransaction(bridge, rawTx, args.GetExtraArgs())
	if err != nil {
		logWorkerError("recall", "DcrmSignTransaction failed", err, "txid", txid)
		return err
	}

	swapTxNonce := args.GetTxNonce()

	// update database before sending transaction
	matchTx := &MatchTx{
		SwapTx:    txHash,
		SwapValue: tokens.CalcSwappedValue(value, false).String(),
		SwapType:  tokens.SwapRecallType,
		SwapNonce: swapTxNonce,
	}
	err = updateSwapinResult(txid, matchTx)
	if err != nil {
		return err
	}
	err = mongodb.UpdateSwapinStatus(txid, mongodb.TxProcessed, now(), "")
	if err != nil {
		return err
	}

	return sendSignedTransaction(bridge, signedTx, txid, true)
}
