package worker

import (
	"fmt"
	"sync"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/mongodb"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
)

var (
	swapinRecallStarter sync.Once
)

func StartRecallJob() error {
	go startSwapinRecallJob()
	return nil
}

func startSwapinRecallJob() error {
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
					logWorkerError("recall", "process recall error", err)
				}
			}
			restInJob(restIntervalInRecallJob)
		}
	})
	return nil
}

func findSwapinsToRecall() ([]*mongodb.MgoSwap, error) {
	status := mongodb.TxToBeRecall
	septime := getSepTimeInFind(maxRecallLifetime)
	return mongodb.FindSwapinsWithStatus(status, septime)
}

func processRecallSwapin(swap *mongodb.MgoSwap) (err error) {
	txid := swap.TxId
	res, err := mongodb.FindSwapinResult(txid)
	if err != nil {
		return err
	}
	if res.SwapTx != "" {
		if res.Status == mongodb.TxToBeRecall {
			mongodb.UpdateSwapinStatus(txid, mongodb.TxProcessed, now(), "")
		}
		return fmt.Errorf("%v already swapped to %v", txid, res.SwapTx)
	}

	value, err := common.GetBigIntFromStr(res.Value)
	if err != nil {
		return fmt.Errorf("wrong value %v", res.Value)
	}

	args := &tokens.BuildTxArgs{
		SwapInfo: &tokens.SwapInfo{
			SwapID:   res.TxId,
			SwapType: tokens.Swap_Recall,
		},
		To:    res.Bind,
		Value: value,
	}
	bridge := tokens.SrcBridge
	rawTx, err := bridge.BuildRawTransaction(args)
	if err != nil {
		return err
	}

	signedTx, err := bridge.DcrmSignTransaction(rawTx, args)
	if err != nil {
		return err
	}

	txHash, err := bridge.SendTransaction(signedTx)

	if err != nil {
		err = mongodb.UpdateSwapinStatus(txid, mongodb.TxRecallFailed, now(), "")
		return err
	}

	mongodb.UpdateSwapinStatus(txid, mongodb.TxProcessed, now(), "")

	matchTx := &MatchTx{
		SwapTx:    txHash,
		SwapValue: tokens.CalcSwappedValue(value, bridge.IsSrcEndpoint()).String(),
		SwapType:  tokens.Swap_Recall,
	}
	return updateSwapinResult(txid, matchTx)
}
