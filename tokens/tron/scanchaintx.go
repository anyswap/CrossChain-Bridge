package tron

import (
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

var (
	step int = 100
	longSleep = time.Second * 2
	shortSleep = time.Millisecond * 500

	maxScanHeight          = uint64(100)
	retryIntervalInScanJob = 3 * time.Second
	restIntervalInScanJob  = 3 * time.Second
)

// StartChainTransactionScanJob scan job
func (b *Bridge) StartChainTransactionScanJob() {
	chainName := b.ChainConfig.BlockChain
	log.Infof("[scanchain] start %v scan chain job", chainName)

	chainCfg := b.GetChainConfig()
	var start, end int64
	start = int64(tools.GetLatestScanHeight(b.IsSrc))
	if start == 0 {
		start = int64(*chainCfg.InitialHeight)
	}
	log.Infof("[scanchain] latest scan height is %v", start)
	end = start + int64(step)
	for {
		res, err := b.GetBlockByLimitNext(start, end)
		if err != nil {
			log.Warn("Get block failed", "start", start, "end", end)
			continue
		}

		for _, tx := range res.Block[0].Transactions {
			b.processTransaction(tx)
		}

		latest := start + int64(len(res.Block)) - 1
		err = tools.UpdateLatestScanInfo(b.IsSrc, uint64(latest))
		if err != nil {
			log.Warn("[scanchain] update latest scan info", "error", err)
		}
		start = start + int64(len(res.Block))
		end = start + int64(step)
		if len(res.Block) < step {
			time.Sleep(longSleep)
		} else {
			time.Sleep(shortSleep)
		}
	}
}

// StartPoolTransactionScanJob not implemented for tron
func (b *Bridge) StartPoolTransactionScanJob() {
	return
}