package xrp

import (
	"fmt"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

var (
	scannedBlocks = tools.NewCachedScannedBlocks(13)

	maxScanHeight          = uint64(100)
	retryIntervalInScanJob = 3 * time.Second
	restIntervalInScanJob  = 3 * time.Second
)

func (b *Bridge) getStartAndLatestHeight() (start, latest uint64) {
	startHeight := tools.GetLatestScanHeight(b.IsSrc)

	chainCfg := b.GetChainConfig()
	confirmations := *chainCfg.Confirmations
	initialHeight := *chainCfg.InitialHeight

	latest = tools.LoopGetLatestBlockNumber(b)

	switch {
	case startHeight != 0:
		start = startHeight
	case initialHeight != 0:
		start = initialHeight
	default:
		if latest > confirmations {
			start = latest - confirmations
		}
	}
	if start < initialHeight {
		start = initialHeight
	}
	if start+maxScanHeight < latest {
		start = latest - maxScanHeight
	}
	return start, latest
}

// StartChainTransactionScanJob scan job
func (b *Bridge) StartChainTransactionScanJob() {
	log.Info("====== StartChainTransactionScanJob ======")
	go b.StartPoolTransactionScanJob()

	chainName := b.ChainConfig.BlockChain
	log.Infof("[scanchain] start %v scan chain job", chainName)

	start, latest := b.getStartAndLatestHeight()
	log.Info("====== 111111 ======", "start", start, "latest", latest)
	_ = tools.UpdateLatestScanInfo(b.IsSrc, start)
	log.Infof("[scanchain] start %v scan chain loop from %v latest=%v", chainName, start, latest)

	chainCfg := b.GetChainConfig()
	confirmations := *chainCfg.Confirmations

	stable := start
	log.Info("====== 222222 ======", "stable", stable)
	errorSubject := fmt.Sprintf("[scanchain] get %v block failed", chainName)
	scanSubject := fmt.Sprintf("[scanchain] scanned %v block", chainName)
	for {
		log.Info("====== 333333 ======")
		latest := tools.LoopGetLatestBlockNumber(b)
		log.Info("====== 444444 ======", "latest", latest)
		log.Info("Scan chain", "latest block number", latest)
		for h := stable + 1; h <= latest; {
			log.Info("====== 555555 ======", "h", h)
			blockHash, err := b.GetBlockHash(h)
			if err != nil {
				log.Error(errorSubject, "height", h, "err", err)
				time.Sleep(retryIntervalInScanJob)
				log.Info("====== 666666 ======")
				continue
			}
			if scannedBlocks.IsBlockScanned(blockHash) {
				log.Info("====== 777777 ======", "blockhash", blockHash)
				continue
			}
			log.Info("Scan chain, get block hash", "", blockHash)
			txids, err := b.GetBlockTxids(h)
			if err != nil {
				log.Error(errorSubject, "height", h, "blockHash", blockHash, "ledger index", h, "err", err)
				time.Sleep(retryIntervalInScanJob)
				continue
			}
			log.Info("Scan chain, get tx ids", "", txids)
			for _, txid := range txids {
				b.processTransaction(txid)
			}
			scannedBlocks.CacheScannedBlock(blockHash, h)
			log.Info(scanSubject, "blockHash", blockHash, "height", h, "txs", len(txids))
			h++
		}
		if stable+confirmations < latest {
			stable = latest - confirmations
			_ = tools.UpdateLatestScanInfo(b.IsSrc, stable)
		}
		time.Sleep(restIntervalInScanJob)
	}
}
