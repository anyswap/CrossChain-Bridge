package worker

import (
	"fmt"
	"sync"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/mongodb"
	. "github.com/fsn-dev/crossChain-Bridge/tokens"
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

	args := &BuildTxArgs{
		IsSwapin: false,
		To:       res.Bind,
		Value:    value,
		Memo:     res.TxId,
	}
	rawTx, err := SrcBridge.BuildRawTransaction(args)
	if err != nil {
		return err
	}

	signedTx, err := SrcBridge.DcrmSignTransaction(rawTx)
	if err != nil {
		return err
	}

	txHash, err := SrcBridge.SendTransaction(signedTx)

	if err != nil {
		err = mongodb.UpdateSwapinStatus(txid, mongodb.TxRecallFailed, now(), "")
		return err
	}

	mongodb.UpdateSwapinStatus(txid, mongodb.TxProcessed, now(), "")

	matchTx := &MatchTx{
		SwapTx:        txHash,
		SetRecallMemo: true,
	}
	return updateSwapinResult(txid, matchTx)
}
