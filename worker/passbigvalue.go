package worker

import (
	"errors"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

const (
	minTimeIntervalToPassBigValue = int64(300) // seconds
)

// StartPassBigValueJob pass big value job
func StartPassBigValueJob() {
	mongodb.MgoWaitGroup.Add(2)
	go startPassBigValSwapinJob()
	go startPassBigValSwapoutJob()
}

func startPassBigValSwapinJob() {
	logWorker("passbigval", "start pass big value swapin job")
	defer mongodb.MgoWaitGroup.Done()
	if !tokens.SrcBridge.GetChainConfig().EnablePassBigValue {
		logWorker("replace", "stop pass big value swapin job as disabled")
		return
	}
	for {
		res, err := findBigValSwapins()
		if err != nil {
			logWorkerError("passbigval", "find big value swapins error", err)
		}
		if len(res) > 0 {
			logWorker("passbigval", "find big value swapins to pass", "count", len(res))
		}
		for _, swap := range res {
			if utils.IsCleanuping() {
				logWorker("passbigval", "stop pass big value swapin job")
				return
			}
			err = processPassBigValSwapin(swap)
			switch {
			case err == nil,
				errors.Is(err, tokens.ErrTxNotStable),
				errors.Is(err, tokens.ErrTxNotFound):
			default:
				logWorkerError("passbigval", "process pass big value swapin error", err, "txid", swap.TxID)
			}
		}
		if utils.IsCleanuping() {
			logWorker("passbigval", "stop pass big value swapin job")
			return
		}
		restInJob(restIntervalInPassBigValJob)
	}
}

func startPassBigValSwapoutJob() {
	logWorker("passbigval", "start pass big value swapout job")
	defer mongodb.MgoWaitGroup.Done()
	if !tokens.DstBridge.GetChainConfig().EnablePassBigValue {
		logWorker("replace", "stop pass big value swapout job as disabled")
		return
	}
	for {
		res, err := findBigValSwapouts()
		if err != nil {
			logWorkerError("passbigval", "find big value swapouts error", err)
		}
		if len(res) > 0 {
			logWorker("passbigval", "find big value swapouts to pass", "count", len(res))
		}
		for _, swap := range res {
			if utils.IsCleanuping() {
				logWorker("passbigval", "stop pass big value swapout job")
				return
			}
			err = processPassBigValSwapout(swap)
			switch {
			case err == nil,
				errors.Is(err, tokens.ErrTxNotStable),
				errors.Is(err, tokens.ErrTxNotFound):
			default:
				logWorkerError("passbigval", "process pass big value swapout error", err, "txid", swap.TxID)
			}
		}
		if utils.IsCleanuping() {
			logWorker("passbigval", "stop pass big value swapout job")
			return
		}
		restInJob(restIntervalInPassBigValJob)
	}
}

func findBigValSwapins() ([]*mongodb.MgoSwap, error) {
	status := mongodb.TxWithBigValue
	septime := getSepTimeInFind(maxPassBigValueLifetime)
	return mongodb.FindSwapinsWithStatus(status, septime)
}

func findBigValSwapouts() ([]*mongodb.MgoSwap, error) {
	status := mongodb.TxWithBigValue
	septime := getSepTimeInFind(maxPassBigValueLifetime)
	return mongodb.FindSwapoutsWithStatus(status, septime)
}

func processPassBigValSwapin(swap *mongodb.MgoSwap) (err error) {
	return processPassBigValSwap(swap, true)
}

func processPassBigValSwapout(swap *mongodb.MgoSwap) error {
	return processPassBigValSwap(swap, false)
}

func processPassBigValSwap(swap *mongodb.MgoSwap, isSwapin bool) (err error) {
	if swap.Status != mongodb.TxWithBigValue {
		return nil
	}
	if swap.InitTime > getSepTimeInFind(passBigValueTimeRequired)*1000 { // init time is milli seconds
		return nil
	}
	if getSepTimeInFind(minTimeIntervalToPassBigValue) < swap.Timestamp {
		return nil
	}

	pairID := swap.PairID
	txid := swap.TxID
	bind := swap.Bind
	bridge := tokens.GetCrossChainBridge(isSwapin)

	_, err = verifySwapTransaction(bridge, pairID, txid, bind, tokens.SwapTxType(swap.TxType))
	if err != nil {
		return err
	}

	if isSwapin {
		return mongodb.PassSwapinBigValue(txid, pairID, bind)
	}
	return mongodb.PassSwapoutBigValue(txid, pairID, bind)
}
