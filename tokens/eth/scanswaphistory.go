package eth

import (
	"time"

	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	"github.com/fsn-dev/crossChain-Bridge/tokens/tools"
	"github.com/fsn-dev/crossChain-Bridge/types"
)

var (
	maxScanHeight          = uint64(15000)
	retryIntervalInScanJob = 3 * time.Second
	restIntervalInScanJob  = 3 * time.Second
)

// StartSwapHistoryScanJob scan job
func (b *Bridge) StartSwapHistoryScanJob() {
	log.Info("[swaphistory] start scan swap history job", "isSrc", b.IsSrc)

	isProcessed := func(txid string) bool {
		if b.IsSrc {
			return tools.IsSwapinExist(txid)
		}
		return tools.IsSwapoutExist(txid)
	}

	go b.scanFirstLoop(isProcessed)

	b.scanTransactionHistory(isProcessed)
}

func (b *Bridge) getSwapoutLogs(blockHeight uint64) ([]*types.RPCLog, error) {
	token := b.TokenConfig
	contractAddress := token.ContractAddress
	logTopic := tokens.LogSwapoutTopic
	return b.GetContractLogs(contractAddress, logTopic, blockHeight)
}

func (b *Bridge) scanFirstLoop(isProcessed func(string) bool) {
	// first loop process all tx history no matter whether processed before
	log.Info("[swaphistory] start first scan loop", "isSrc", b.IsSrc)
	latest := tools.LoopGetLatestBlockNumber(b)
	for height := latest; height+maxScanHeight > latest; {
		logs, err := b.getSwapoutLogs(height)
		if err != nil {
			//log.Trace("[swaphistory] first scan get swapout logs error", "isSrc", b.IsSrc, "height", height, "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		for _, log := range logs {
			txid := log.TxHash.String()
			if !isProcessed(txid) {
				b.processTransaction(txid)
			}
		}
		height--
	}

	log.Info("[scanhistory] finish first scan loop", "isSrc", b.IsSrc)
}

func (b *Bridge) scanTransactionHistory(isProcessed func(string) bool) {
	log.Info("[scanhistory] start scan swap history loop")
	var (
		height uint64
		rescan = true
	)
	for {
		if rescan {
			height = tools.LoopGetLatestBlockNumber(b)
		}
		logs, err := b.getSwapoutLogs(height)
		if err != nil {
			log.Error("[swaphistory] get swapout logs error", "isSrc", b.IsSrc, "height", height, "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		log.Info("[scanhistory] scan swap history", "isSrc", b.IsSrc, "height", height, "count", len(logs))
		for _, log := range logs {
			txid := log.TxHash.String()
			if isProcessed(txid) {
				rescan = true
				break // rescan if already processed
			}
			b.processTransaction(txid)
		}
		if rescan {
			time.Sleep(restIntervalInScanJob)
		} else {
			height--
		}
	}
}
