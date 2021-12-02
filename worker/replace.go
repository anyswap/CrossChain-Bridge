package worker

import (
	"strings"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var (
	defWaitTimeToReplace = int64(900) // seconds
	defMaxReplaceCount   = 20

	// key is signer address
	swapinReplaceChanMap  = make(map[string]chan *mongodb.MgoSwapResult)
	swapoutReplaceChanMap = make(map[string]chan *mongodb.MgoSwapResult)
)

// StartReplaceJob replace job
func StartReplaceJob() {
	if tokens.DstNonceSetter != nil {
		mongodb.MgoWaitGroup.Add(1)
		go startReplaceSwapinJob()
	}

	if tokens.SrcNonceSetter != nil {
		mongodb.MgoWaitGroup.Add(1)
		go startReplaceSwapoutJob()
	}
}

func startReplaceSwapinJob() {
	logWorker("replace", "start replace swapin job")
	defer mongodb.MgoWaitGroup.Done()
	if !tokens.DstBridge.GetChainConfig().EnableReplaceSwap {
		logWorker("replace", "stop replace swapin job as disabled")
		return
	}
	for {
		res, err := findSwapinsToReplace()
		if err != nil {
			logWorkerError("replace", "find swapins error", err)
		}
		logWorker("replace", "find swapins to replace", "count", len(res))
		for _, swap := range res {
			if utils.IsCleanuping() {
				logWorker("replace", "stop replace swapin job")
				return
			}
			processReplaceSwap(swap, true)
		}
		if utils.IsCleanuping() {
			logWorker("replace", "stop replace swapin job")
			return
		}
		restInJob(restIntervalInReplaceSwapJob)
	}
}

func startReplaceSwapoutJob() {
	logWorker("replace", "start replace swapout job")
	defer mongodb.MgoWaitGroup.Done()
	if !tokens.SrcBridge.GetChainConfig().EnableReplaceSwap {
		logWorker("replace", "stop replace swapout job as disabled")
		return
	}
	for {
		res, err := findSwapoutsToReplace()
		if err != nil {
			logWorkerError("replace", "find swapouts error", err)
		}
		logWorker("replace", "find swapouts to replace", "count", len(res))
		for _, swap := range res {
			if utils.IsCleanuping() {
				logWorker("replace", "stop replace swapout job")
				return
			}
			processReplaceSwap(swap, false)
		}
		if utils.IsCleanuping() {
			logWorker("replace", "stop replace swapout job")
			return
		}
		restInJob(restIntervalInReplaceSwapJob)
	}
}

func findSwapinsToReplace() ([]*mongodb.MgoSwapResult, error) {
	status := mongodb.MatchTxNotStable
	septime := getSepTimeInFind(maxReplaceSwapLifetime)
	return mongodb.FindSwapResultsToReplace(status, septime, true)
}

func findSwapoutsToReplace() ([]*mongodb.MgoSwapResult, error) {
	status := mongodb.MatchTxNotStable
	septime := getSepTimeInFind(maxReplaceSwapLifetime)
	return mongodb.FindSwapResultsToReplace(status, septime, false)
}

func getReplaceConfigs(isSwapin bool) (waitTimeToReplace int64, maxReplaceCount int) {
	var chainCfg *tokens.ChainConfig
	if isSwapin {
		chainCfg = tokens.DstBridge.GetChainConfig()
	} else {
		chainCfg = tokens.SrcBridge.GetChainConfig()
	}
	waitTimeToReplace = chainCfg.WaitTimeToReplace
	maxReplaceCount = chainCfg.MaxReplaceCount
	return waitTimeToReplace, maxReplaceCount
}

func processReplaceSwap(swap *mongodb.MgoSwapResult, isSwapin bool) {
	if swap.SwapNonce == 0 || swap.SwapHeight != 0 {
		return
	}
	if swap.Status != mongodb.MatchTxNotStable {
		return
	}
	waitTimeToReplace, maxReplaceCount := getReplaceConfigs(isSwapin)
	if waitTimeToReplace == 0 {
		waitTimeToReplace = defWaitTimeToReplace
	}
	if maxReplaceCount == 0 {
		maxReplaceCount = defMaxReplaceCount
	}
	if len(swap.OldSwapTxs) > maxReplaceCount {
		return
	}
	if getSepTimeInFind(waitTimeToReplace) < swap.Timestamp {
		return
	}
	bridge := tokens.GetCrossChainBridge(!isSwapin)
	err := checkIfSwapNonceHasPassed(bridge, swap, true)
	if err != nil {
		return
	}
	_ = updateSwapTimestamp(swap.TxID, swap.PairID, swap.Bind, isSwapin)
	dispatchReplaceTask(swap)
}

func dispatchReplaceTask(swap *mongodb.MgoSwapResult) {
	logWorker("replace", "dispatch task", "swap", swap)
	pairID := strings.ToLower(swap.PairID)
	pairCfg := tokens.GetTokenPairConfig(pairID)
	isSwapin := tokens.SwapType(swap.SwapType) == tokens.SwapinType
	if isSwapin {
		swapinDcrmAddr := strings.ToLower(pairCfg.DestToken.DcrmAddress)
		if _, exist := swapinReplaceChanMap[swapinDcrmAddr]; !exist {
			swapinReplaceChanMap[swapinDcrmAddr] = make(chan *mongodb.MgoSwapResult, swapChanSize)
			go processReplaceSwapTask(swapinReplaceChanMap[swapinDcrmAddr])
		}
		swapinReplaceChanMap[swapinDcrmAddr] <- swap
	} else {
		swapoutDcrmAddr := strings.ToLower(pairCfg.SrcToken.DcrmAddress)
		if _, exist := swapoutReplaceChanMap[swapoutDcrmAddr]; !exist {
			swapoutReplaceChanMap[swapoutDcrmAddr] = make(chan *mongodb.MgoSwapResult, swapChanSize)
			go processReplaceSwapTask(swapoutReplaceChanMap[swapoutDcrmAddr])
		}
		swapoutReplaceChanMap[swapoutDcrmAddr] <- swap
	}
}

func processReplaceSwapTask(swapChan <-chan *mongodb.MgoSwapResult) {
	for {
		select {
		case <-utils.CleanupChan:
			return
		case swap := <-swapChan:
			doReplaceSwap(swap)
		}
	}
}

func doReplaceSwap(swap *mongodb.MgoSwapResult) {
	if swap.SwapNonce == 0 || swap.SwapHeight != 0 {
		return
	}
	isSwapin := tokens.SwapType(swap.SwapType) == tokens.SwapinType
	nonceSetter := tokens.GetNonceSetter(!isSwapin)
	if nonceSetter == nil {
		logWorkerWarn("replace", "not nonce support chain", "isSwapin", isSwapin)
		return
	}
	logWorker("replace", "process task", "swap", swap)

	var txHash string
	var err error
	if isSwapin {
		txHash, err = ReplaceSwapin(swap.TxID, swap.PairID, swap.Bind, "")
	} else {
		txHash, err = ReplaceSwapout(swap.TxID, swap.PairID, swap.Bind, "")
	}
	if err != nil {
		logWorker("replace", "replace swap error", "pairID", swap.PairID, "txid", swap.TxID, "bind", swap.Bind, "isSwapin", isSwapin, "swapNonce", swap.SwapNonce, "err", err)
	} else {
		logWorker("replace", "replace swap finished", "pairID", swap.PairID, "txid", swap.TxID, "bind", swap.Bind, "isSwapin", isSwapin, "txHash", txHash, "swapNonce", swap.SwapNonce)
	}
}

func isTransactionOnChain(bridge tokens.NonceSetter, txHash string) bool {
	if txHash == "" {
		return false
	}
	blockHeight, _ := bridge.GetTxBlockInfo(txHash)
	return blockHeight > 0
}

func isSwapResultTxOnChain(bridge tokens.NonceSetter, res *mongodb.MgoSwapResult) bool {
	if isTransactionOnChain(bridge, res.SwapTx) {
		return true
	}
	for _, tx := range res.OldSwapTxs {
		if isTransactionOnChain(bridge, tx) {
			return true
		}
	}
	return false
}
