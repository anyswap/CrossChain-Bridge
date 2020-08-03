package worker

import (
	"errors"
	"sync"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var (
	swapinRetryStarter  sync.Once
	swapoutRetryStarter sync.Once
)

const (
	defMinTimeToRetry = 3600 // unit senconds
)

// StartSwapRetryJob start retry for failed swaptx
func StartSwapRetryJob() {
	go startSwapinRetryJob()
	go startSwapoutRetryJob()
}

func startSwapinRetryJob() {
	swapinRetryStarter.Do(func() {
		logWorker("retryswap", "start swapin retry job")
		for {
			res, err := findSwapinsToRetry()
			if err != nil {
				logWorkerError("retryswap", "find swapins to retry error", err)
			}
			if len(res) > 0 {
				logWorker("retryswap", "find swapins to retry", "count", len(res))
			}
			for _, swap := range res {
				err = processRetrySwapin(swap)
				if err != nil {
					logWorkerError("retryswap", "process retry swapin error", err, "txid", swap.TxID)
				}
			}
			restInJob(restIntervalInRetryJob)
		}
	})
}

func startSwapoutRetryJob() {
	swapoutRetryStarter.Do(func() {
		logWorker("retryswap", "start swapout retry job")
		for {
			res, err := findSwapoutsToRetry()
			if err != nil {
				logWorkerError("retryswap", "find swapouts to retry error", err)
			}
			if len(res) > 0 {
				logWorker("retryswap", "find swapouts to retry", "count", len(res))
			}
			for _, swap := range res {
				err = processRetrySwapout(swap)
				if err == nil {
					logWorker("retryswap", "process retry swapout success", "txid", swap.TxID)
				}
			}
			restInJob(restIntervalInRetryJob)
		}
	})
}

func findSwapinsToRetry() ([]*mongodb.MgoSwap, error) {
	status := mongodb.TxSwapFailed
	septime := getSepTimeInFind(maxRetryLifetime)
	return mongodb.FindSwapinsWithStatus(status, septime)
}

func findSwapoutsToRetry() ([]*mongodb.MgoSwap, error) {
	status := mongodb.TxSwapFailed
	septime := getSepTimeInFind(maxRetryLifetime)
	return mongodb.FindSwapoutsWithStatus(status, septime)
}

func processRetrySwapin(swap *mongodb.MgoSwap) (err error) {
	txid := swap.TxID
	res, err := mongodb.FindSwapinResult(txid)
	if err != nil {
		return err
	}
	err = checkSwapCanRetry(res, tokens.DstBridge)
	if err != nil {
		return err
	}
	logWorker("retryswap", "update swapin status to TxNotSwapped to retry", "txid", txid, "swaptx", res.SwapTx)
	if res.SwapType == uint32(tokens.SwapRecallType) {
		_ = mongodb.UpdateSwapinStatus(txid, mongodb.TxToBeRecall, now(), "")
	} else {
		_ = mongodb.UpdateSwapinStatus(txid, mongodb.TxNotSwapped, now(), "")
	}
	_ = mongodb.UpdateSwapinResultStatus(txid, mongodb.MatchTxEmpty, now(), "")
	return nil
}

func processRetrySwapout(swap *mongodb.MgoSwap) (err error) {
	txid := swap.TxID
	res, err := mongodb.FindSwapoutResult(txid)
	if err != nil {
		return err
	}
	err = checkSwapCanRetry(res, tokens.SrcBridge)
	if err != nil {
		return err
	}
	logWorker("retryswap", "update swapout status to TxNotSwapped to retry", "txid", txid, "swaptx", res.SwapTx)
	_ = mongodb.UpdateSwapoutStatus(txid, mongodb.TxNotSwapped, now(), "")
	_ = mongodb.UpdateSwapoutResultStatus(txid, mongodb.MatchTxEmpty, now(), "")
	return nil
}

func checkSwapCanRetry(res *mongodb.MgoSwapResult, bridge tokens.CrossChainBridge) error {
	return checkSwapCanRetryWithStatus(mongodb.TxSwapFailed, res, bridge)
}

func checkSwapCanRetryWithStatus(status mongodb.SwapStatus, res *mongodb.MgoSwapResult, bridge tokens.CrossChainBridge) error {
	if res.SwapType == uint32(tokens.NoSwapType) {
		return errors.New("swap type is no swap")
	}
	if res.Status != status {
		return errors.New("swap result status can not retry")
	}
	if res.SwapTx == "" {
		return errors.New("swap without swaptx")
	}
	if res.SwapHeight != 0 {
		return errors.New("swaptx with non zero height")
	}
	_, err := bridge.GetTransaction(res.SwapTx)
	if err == nil {
		return errors.New("swaptx exist in chain or pool")
	}
	passedTime := getPassedTimeSince(res.Timestamp)
	tokenCfg := tokens.GetTokenConfig(bridge.IsSrcEndpoint())
	minTimeToRetry := tokenCfg.MinTimeToRetry
	if minTimeToRetry == 0 {
		minTimeToRetry = defMinTimeToRetry
	}
	if passedTime < minTimeToRetry {
		return errors.New("should wait some time")
	}
	nonceGetter, ok := bridge.(tokens.NonceGetter)
	if !ok {
		return nil
	}
	// eth enhanced, if we fail at nonce a, we should retry after nonce a
	// to ensure tx with nonce a is on blockchain to prevent double swapping
	var nonce uint64
	retryGetNonceCount := 3
	for i := 0; i < retryGetNonceCount; i++ {
		nonce, err = nonceGetter.GetPoolNonce(tokenCfg.DcrmAddress, "latest")
		if err == nil {
			break
		}
		log.Warn("get account nonce failed", "address", tokenCfg.DcrmAddress)
		time.Sleep(time.Second)
	}
	if nonce < res.SwapNonce {
		return errors.New("can not retry swap with lower nonce")
	}
	return nil
}
