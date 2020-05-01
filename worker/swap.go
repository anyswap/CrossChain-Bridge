package worker

import (
	"fmt"
	"sync"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/mongodb"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
)

var (
	swapinSwapStarter  sync.Once
	swapoutSwapStarter sync.Once
)

func StartSwapJob() error {
	go startSwapinSwapJob()
	go startSwapoutSwapJob()
	return nil
}

func startSwapinSwapJob() error {
	swapinSwapStarter.Do(func() {
		logWorker("swap", "start swapin swap job")
		for {
			res, err := findSwapinsToSwap()
			if err != nil {
				logWorkerError("swap", "find swapins error", err)
			}
			for _, swap := range res {
				err = processSwapinSwap(swap)
				if err != nil {
					logWorkerError("swap", "process swapin swap error", err)
				}
			}
			restInJob(restIntervalInDoSwapJob)
		}
	})
	return nil
}

func startSwapoutSwapJob() error {
	swapoutSwapStarter.Do(func() {
		logWorker("swap", "start swapout swap job")
		for {
			res, err := findSwapoutsToSwap()
			if err != nil {
				logWorkerError("swap", "find swapouts error", err)
			}
			for _, swap := range res {
				err = processSwapoutSwap(swap)
				if err != nil {
					logWorkerError("swap", "process swapout swap error", err)
				}
			}
			restInJob(restIntervalInDoSwapJob)
		}
	})
	return nil
}

func findSwapinsToSwap() ([]*mongodb.MgoSwap, error) {
	status := mongodb.TxNotSwapped
	septime := getSepTimeInFind(maxDoSwapLifetime)
	return mongodb.FindSwapinsWithStatus(status, septime)
}

func findSwapoutsToSwap() ([]*mongodb.MgoSwap, error) {
	status := mongodb.TxNotSwapped
	septime := getSepTimeInFind(maxDoSwapLifetime)
	return mongodb.FindSwapoutsWithStatus(status, septime)
}

func processSwapinSwap(swap *mongodb.MgoSwap) (err error) {
	txid := swap.TxId
	res, err := mongodb.FindSwapinResult(txid)
	if err != nil {
		return err
	}
	if res.SwapTx != "" {
		if res.Status == mongodb.TxNotSwapped {
			mongodb.UpdateSwapinStatus(txid, mongodb.TxProcessed, now(), "")
		}
		return fmt.Errorf("%v already swapped to %v", txid, res.SwapTx)
	}

	value, err := common.GetBigIntFromStr(res.Value)
	if err != nil {
		return fmt.Errorf("wrong value %v", res.Value)
	}

	args := &tokens.BuildTxArgs{
		IsSwapin: true,
		To:       res.Bind,
		Value:    value,
		Memo:     res.TxId,
	}
	rawTx, err := tokens.DstBridge.BuildRawTransaction(args)
	if err != nil {
		return err
	}

	signedTx, err := tokens.DstBridge.DcrmSignTransaction(rawTx)
	if err != nil {
		return err
	}

	txHash, err := tokens.DstBridge.SendTransaction(signedTx)

	if err != nil {
		err = mongodb.UpdateSwapinStatus(txid, mongodb.TxSwapFailed, now(), "")
		return err
	}

	mongodb.UpdateSwapinStatus(txid, mongodb.TxProcessed, now(), "")

	matchTx := &MatchTx{
		SwapTx: txHash,
	}
	return updateSwapinResult(txid, matchTx)
}

func processSwapoutSwap(swap *mongodb.MgoSwap) (err error) {
	txid := swap.TxId
	res, err := mongodb.FindSwapoutResult(txid)
	if err != nil {
		return err
	}
	if res.SwapTx != "" {
		if res.Status == mongodb.TxNotSwapped {
			mongodb.UpdateSwapoutStatus(txid, mongodb.TxProcessed, now(), "")
		}
		return fmt.Errorf("%v already swapped to %v", txid, res.SwapTx)
	}

	value, err := common.GetBigIntFromStr(res.Value)
	if err != nil {
		return fmt.Errorf("wrong value %v", res.Value)
	}

	args := &tokens.BuildTxArgs{
		IsSwapin: false,
		To:       res.Bind,
		Value:    value,
		Memo:     res.TxId,
	}
	rawTx, err := tokens.SrcBridge.BuildRawTransaction(args)
	if err != nil {
		return err
	}

	signedTx, err := tokens.SrcBridge.DcrmSignTransaction(rawTx)
	if err != nil {
		return err
	}

	txHash, err := tokens.SrcBridge.SendTransaction(signedTx)

	if err != nil {
		err = mongodb.UpdateSwapoutStatus(txid, mongodb.TxSwapFailed, now(), "")
		return err
	}

	mongodb.UpdateSwapoutStatus(txid, mongodb.TxProcessed, now(), "")

	matchTx := &MatchTx{
		SwapTx: txHash,
	}
	return updateSwapoutResult(txid, matchTx)
}
