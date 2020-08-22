package btc

import (
	"fmt"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

var (
	scannedBlocks = tools.NewCachedScannedBlocks(13)
)

// StartChainTransactionScanJob scan job
func (b *Bridge) StartChainTransactionScanJob() {
	chainName := b.TokenConfig.BlockChain
	log.Infof("[scanchain] start %v scan chain job", chainName)

	startHeight := tools.GetLatestScanHeight(b.IsSrc)
	confirmations := *b.TokenConfig.Confirmations
	initialHeight := b.TokenConfig.InitialHeight

	latest := tools.LoopGetLatestBlockNumber(b)

	var height uint64
	switch {
	case startHeight != 0:
		height = startHeight
	case initialHeight != 0:
		height = initialHeight
	default:
		if latest > confirmations {
			height = latest - confirmations
		}
	}
	if height < initialHeight {
		height = initialHeight
	}
	if height+maxScanHeight < latest {
		height = latest - maxScanHeight
	}
	_ = tools.UpdateLatestScanInfo(b.IsSrc, height)
	log.Infof("[scanchain] start %v scan chain loop from %v latest=%v", chainName, height, latest)

	stable := height
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
