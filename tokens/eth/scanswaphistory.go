package eth

import (
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
	"github.com/anyswap/CrossChain-Bridge/types"
)

var (
	maxScanHeight          = uint64(15000)
	retryIntervalInScanJob = 3 * time.Second
	restIntervalInScanJob  = 3 * time.Second
)

// StartSwapHistoryScanJob scan job
func (b *Bridge) StartSwapHistoryScanJob() {
	if b.TokenConfig.ContractAddress == "" {
		return
	}
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

func (b *Bridge) getSwapLogs(blockHeight uint64) ([]*types.RPCLog, error) {
	token := b.TokenConfig
	contractAddress := token.ContractAddress
	var logTopic string
	if b.IsSrc {
		logTopic = common.ToHex(getLogSwapinTopic())
	} else {
		logTopic = common.ToHex(getLogSwapoutTopic())
	}
	return b.GetContractLogs(contractAddress, logTopic, blockHeight)
}

func (b *Bridge) scanFirstLoop(isProcessed func(string) bool) {
	// first loop process all tx history no matter whether processed before
	log.Info("[scanhistory] start first scan loop", "isSrc", b.IsSrc)
	initialHeight := b.TokenConfig.InitialHeight
	latest := tools.LoopGetLatestBlockNumber(b)
	for height := latest; height+maxScanHeight > latest && height >= initialHeight; {
		logs, err := b.getSwapLogs(height)
		if err != nil {
			log.Error("[scanhistory] get swap logs error", "isSrc", b.IsSrc, "height", height, "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		for _, swaplog := range logs {
			txid := swaplog.TxHash.String()
			if !isProcessed(txid) {
				log.Info("[scanhistory] first scan loop", "isSrc", b.IsSrc, "txid", txid, "height", height)
				b.processTransaction(txid)
			}
		}
		if height > 0 {
			height--
		} else {
			break
		}
	}

	log.Info("[scanhistory] finish first scan loop", "isSrc", b.IsSrc)
}

func (b *Bridge) scanTransactionHistory(isProcessed func(string) bool) {
	log.Info("[scanhistory] start scan swap history loop")
	var (
		height        uint64
		rescan        = true
		initialHeight = b.TokenConfig.InitialHeight
	)
	for {
		if rescan || height < initialHeight || height == 0 {
			height = tools.LoopGetLatestBlockNumber(b)
		}
		logs, err := b.getSwapLogs(height)
		if err != nil {
			log.Error("[swaphistory] get swap logs error", "isSrc", b.IsSrc, "height", height, "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		for _, swaplog := range logs {
			txid := swaplog.TxHash.String()
			if isProcessed(txid) {
				rescan = true
				break // rescan if already processed
			}
			log.Info("[scanhistory] scanned tx", "isSrc", b.IsSrc, "txid", txid, "height", height)
			b.processTransaction(txid)
		}
		if rescan {
			time.Sleep(restIntervalInScanJob)
		} else if height > 0 {
			height--
		}
	}
}
