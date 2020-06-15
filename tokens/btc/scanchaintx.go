package btc

import (
	"time"

	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/tokens/tools"
)

var (
	scannedBlocks = tools.NewCachedScannedBlocks(13)
)

// StartChainTransactionScanJob scan job
func (b *Bridge) StartChainTransactionScanJob() {
	log.Info("[scanchain] start scan chain tx job", "isSrc", b.IsSrc)

	startHeight := tools.GetLatestScanHeight(b.IsSrc)
	confirmations := *b.TokenConfig.Confirmations

	var height uint64
	if startHeight == 0 {
		latest := tools.LoopGetLatestBlockNumber(b)
		if latest > confirmations {
			height = latest - confirmations
			_ = tools.UpdateLatestScanInfo(b.IsSrc, height)
		}
	} else {
		height = startHeight
	}
	log.Info("[scanchain] start scan tx history loop", "isSrc", b.IsSrc, "start", height)

	for {
		latest := tools.LoopGetLatestBlockNumber(b)
		for h := height + 1; h <= latest; {
			blockHash, err := b.GetBlockHash(h)
			if err != nil {
				log.Error("[scanchain] get block hash failed", "isSrc", b.IsSrc, "height", h, "err", err)
				time.Sleep(retryIntervalInScanJob)
				continue
			}
			if scannedBlocks.IsBlockScanned(blockHash) {
				h++
				continue
			}
			txids, err := b.GetBlockTxids(blockHash)
			if err != nil {
				log.Error("[scanchain] get block txids failed", "isSrc", b.IsSrc, "height", h, "blockHash", blockHash, "err", err)
				time.Sleep(retryIntervalInScanJob)
				continue
			}
			for _, txid := range txids {
				b.processTransaction(txid)
			}
			scannedBlocks.CacheScannedBlock(blockHash, h)
			log.Info("[scanchain] scanned tx history", "isSrc", b.IsSrc, "blockHash", blockHash, "height", h, "txs", len(txids))
			h++
		}
		if latest > confirmations {
			latestStable := latest - confirmations
			if height < latestStable {
				height = latestStable
				_ = tools.UpdateLatestScanInfo(b.IsSrc, height)
			}
		}
		time.Sleep(restIntervalInScanJob)
	}
}
