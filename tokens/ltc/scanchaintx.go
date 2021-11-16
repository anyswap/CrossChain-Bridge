package ltc

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
	go b.StartPoolTransactionScanJob()

	chainName := b.ChainConfig.BlockChain
	log.Infof("[scanchain] start %v scan chain job", chainName)

	start, latest := b.getStartAndLatestHeight()
	_ = tools.UpdateLatestScanInfo(b.IsSrc, start)
	log.Infof("[scanchain] start %v scan chain loop from %v latest=%v", chainName, start, latest)

	chainCfg := b.GetChainConfig()
	confirmations := *chainCfg.Confirmations

	stable := start
	errorSubject := fmt.Sprintf("[scanchain] get %v block failed", chainName)
	scanSubject := fmt.Sprintf("[scanchain] scanned %v block", chainName)
	for {
		latest := tools.LoopGetLatestBlockNumber(b)
		for h := stable + 1; h <= latest; {
			blockHash, err := b.GetBlockHash(h)
			if err != nil {
				log.Error(errorSubject, "height", h, "err", err)
				time.Sleep(retryIntervalInScanJob)
				continue
			}
			if scannedBlocks.IsBlockScanned(blockHash) {
				h++
				continue
			}
			txids, err := b.GetBlockTxids(blockHash)
			if err != nil {
				log.Error(errorSubject, "height", h, "blockHash", blockHash, "err", err)
				time.Sleep(retryIntervalInScanJob)
				continue
			}
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
