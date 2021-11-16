package worker

import (
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// StartCheckFailedSwapJob check failed swap job
func StartCheckFailedSwapJob() {
	mongodb.MgoWaitGroup.Add(2)
	go doCheckFailedSwapinJob()
	go doCheckFailedSwapoutJob()
}

func doCheckFailedSwapinJob() {
	defer mongodb.MgoWaitGroup.Done()
	logWorker("checkfailedswap", "start check failed swapin job")
	for {
		septime := getSepTimeInFind(maxCheckFailedSwapLifetime)
		res, err := mongodb.FindSwapinResultsWithStatus(mongodb.MatchTxFailed, septime)
		if err != nil {
			logWorkerError("checkfailedswap", "find failed swapin error", err)
		}
		if len(res) > 0 {
			logWorker("checkfailedswap", "find failed swapin to check", "count", len(res))
		}
		for _, swap := range res {
			if utils.IsCleanuping() {
				logWorker("checkfailedswap", "stop check failed swapin job")
				return
			}
			err = checkFailedSwap(swap, true)
			if err != nil {
				logWorkerError("checkfailedswap", "check failed swapin error", err, "txid", swap.TxID, "pairID", swap.PairID)
			}
		}
		if utils.IsCleanuping() {
			logWorker("checkfailedswap", "stop check failed swapin job")
			return
		}
		restInJob(restIntervalInCheckFailedSwapJob)
	}
}

func doCheckFailedSwapoutJob() {
	defer mongodb.MgoWaitGroup.Done()
	logWorker("checkfailedswap", "start check failed swapout job")
	for {
		septime := getSepTimeInFind(maxCheckFailedSwapLifetime)
		res, err := mongodb.FindSwapoutResultsWithStatus(mongodb.MatchTxFailed, septime)
		if err != nil {
			logWorkerError("checkfailedswap", "find failed swapout error", err)
		}
		if len(res) > 0 {
			logWorker("checkfailedswap", "find failed swapout to check", "count", len(res))
		}
		for _, swap := range res {
			if utils.IsCleanuping() {
				logWorker("checkfailedswap", "stop check failed swapout job")
				return
			}
			err = checkFailedSwap(swap, false)
			if err != nil {
				logWorkerError("checkfailedswap", "check failed swapout error", err, "txid", swap.TxID, "pairID", swap.PairID)
			}
		}
		if utils.IsCleanuping() {
			logWorker("checkfailedswap", "stop check failed swapout job")
			return
		}
		restInJob(restIntervalInCheckFailedSwapJob)
	}
}

func checkFailedSwap(swap *mongodb.MgoSwapResult, isSwapin bool) error {
	if swap.SwapNonce == 0 || swap.SwapTx == "" {
		return nil
	}

	txid, pairID, bind := swap.TxID, swap.PairID, swap.Bind

	resBridge := tokens.GetCrossChainBridge(!isSwapin)
	if resBridge == nil {
		logWorkerWarn("checkfailedswap", "bridge not exist", "txid", swap.TxID, "pairID", swap.PairID, "bind", swap.Bind)
		return nil
	}
	nonceSetter, ok := resBridge.(tokens.NonceSetter)
	if !ok {
		return nil
	}
	tokenCfg := resBridge.GetTokenConfig(pairID)
	if tokenCfg == nil {
		return fmt.Errorf("no token config for pairID '%v'", pairID)
	}

	txStatus := getSwapTxStatus(resBridge, swap)
	if txStatus != nil && txStatus.IsSwapTxOnChainAndFailed(tokenCfg) {
		return nil
	}

	if txStatus != nil && txStatus.BlockHeight > 0 {
		logWorker("checkfailedswap", "do checking with height", "swap", swap, "swapheight", txStatus.BlockHeight, "confirmations", txStatus.Confirmations)
		if txStatus.Confirmations < *resBridge.GetChainConfig().Confirmations {
			return markSwapResultUnstable(txid, pairID, bind, isSwapin)
		}
		return markSwapResultStable(txid, pairID, bind, isSwapin)
	}

	nonce, err := nonceSetter.GetPoolNonce(tokenCfg.DcrmAddress, "latest")
	if err != nil {
		return errGetNonceFailed
	}

	logWorker("checkfailedswap", "do checking without height", "swap", swap, "swapnonce", swap.SwapNonce, "latestnonce", nonce)
	if nonce <= swap.SwapNonce {
		return markSwapResultUnstable(txid, pairID, bind, isSwapin)
	}
	return nil
}
