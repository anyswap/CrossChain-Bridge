package eth

import (
	"math/big"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

var (
	scannedBlocks = tools.NewCachedScannedBlocks(67)
)

// StartChainTransactionScanJob scan job
func (b *Bridge) StartChainTransactionScanJob() {
	log.Info("[scanchain] start scan chain job", "isSrc", b.IsSrc)

	startHeight := tools.GetLatestScanHeight(b.IsSrc)
	confirmations := *b.TokenConfig.Confirmations
	initialHeight := b.TokenConfig.InitialHeight

	var height uint64
	switch {
	case startHeight != 0:
		height = startHeight
	case initialHeight != 0:
		height = initialHeight
	default:
		latest := tools.LoopGetLatestBlockNumber(b)
		if latest > confirmations {
			height = latest - confirmations
		}
	}
	if height < initialHeight {
		height = initialHeight
	}
	_ = tools.UpdateLatestScanInfo(b.IsSrc, height)
	log.Info("[scanchain] start scan chain loop", "isSrc", b.IsSrc, "start", height)

	for {
		latest := tools.LoopGetLatestBlockNumber(b)
		for h := height + 1; h <= latest; {
			block, err := b.GetBlockByNumber(new(big.Int).SetUint64(h))
			if err != nil {
				log.Error("[scanchain] get block failed", "isSrc", b.IsSrc, "height", h, "err", err)
				time.Sleep(retryIntervalInScanJob)
				continue
			}
			blockHash := block.Hash.String()
			if scannedBlocks.IsBlockScanned(blockHash) {
				h++
				continue
			}
			for _, tx := range block.Transactions {
				b.processTransaction(tx.String())
			}
			scannedBlocks.CacheScannedBlock(blockHash, h)
			log.Info("[scanchain] scanned block", "isSrc", b.IsSrc, "blockHash", blockHash, "height", h, "txs", len(block.Transactions))
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
