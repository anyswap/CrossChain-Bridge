package eth

import (
	"time"

	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/mongodb"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	"github.com/fsn-dev/crossChain-Bridge/types"
)

var (
	maxScanHeight          = uint64(15000)
	retryIntervalInScanJob = 3 * time.Second
	restIntervalInScanJob  = 3 * time.Second
)

// StartSwapHistoryScanJob scan job
func (b *Bridge) StartSwapHistoryScanJob() {
	log.Info("[swaphistory] start scan history job", "isSrc", b.IsSrc)

	isProcessed := func(txid string, txheight uint64) bool {
		swap, _ := mongodb.FindSwapout(txid)
		return swap != nil
	}

	go b.scanFirstLoop(isProcessed)

	b.scanTransactionHistory(isProcessed)
}

func (b *Bridge) getLatestHeight() uint64 {
	for {
		latest, err := b.GetLatestBlockNumber()
		if err == nil {
			return latest
		}
		log.Error("[swaphistory] get latest block number error", "isSrc", b.IsSrc, "err", err)
		time.Sleep(retryIntervalInScanJob)
	}
}

func (b *Bridge) getSwapoutLogs(blockHeight uint64) ([]*types.RPCLog, error) {
	token := b.TokenConfig
	contractAddress := token.ContractAddress
	logTopic := tokens.LogSwapoutTopic
	return b.GetContractLogs(contractAddress, logTopic, blockHeight)
}

func (b *Bridge) scanFirstLoop(isProcessed func(string, uint64) bool) {
	// first loop process all tx history no matter whether processed before
	log.Info("[swaphistory] start first scan loop", "isSrc", b.IsSrc)
	latest := b.getLatestHeight()
	for height := latest; height+maxScanHeight > latest; {
		logs, err := b.getSwapoutLogs(height)
		if err != nil {
			//log.Trace("[swaphistory] first scan get swapout logs error", "isSrc", b.IsSrc, "height", height, "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		for _, log := range logs {
			txid := log.TxHash.String()
			if !isProcessed(txid, height) {
				b.processTransaction(txid)
			}
		}
		height--
	}
}

func (b *Bridge) scanTransactionHistory(isProcessed func(string, uint64) bool) {
	log.Info("[scanhistory] start scan tx history loop")
	var (
		height uint64
		rescan = true
	)
	for {
		if rescan {
			height = b.getLatestHeight()
		}
		logs, err := b.getSwapoutLogs(height)
		if err != nil {
			log.Error("[swaphistory] get swapout logs error", "isSrc", b.IsSrc, "height", height, "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		for _, log := range logs {
			txid := log.TxHash.String()
			if isProcessed(txid, height) {
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
