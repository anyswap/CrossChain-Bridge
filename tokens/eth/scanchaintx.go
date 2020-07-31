package eth

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

var (
	scannedBlocks = tools.NewCachedScannedBlocks(67)

	quickSyncFinish  bool
	quickSyncWorkers = uint64(4)
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

	latest := tools.LoopGetLatestBlockNumber(b)
	if latest > height {
		go b.quickSync(height, latest)
	}

	chainName := b.TokenConfig.BlockChain
	stable := latest
	errorSubject := fmt.Sprintf("[scanchain] get %v block failed", chainName)
	scanSubject := fmt.Sprintf("[scanchain] scanned %v block", chainName)
	for {
		latest = tools.LoopGetLatestBlockNumber(b)
		for h := stable + 1; h <= latest; {
			block, err := b.GetBlockByNumber(new(big.Int).SetUint64(h))
			if err != nil {
				log.Error(errorSubject, "height", h, "err", err)
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
			log.Info(scanSubject, "blockHash", blockHash, "height", h, "txs", len(block.Transactions))
			h++
		}
		if stable+confirmations < latest {
			stable = latest - confirmations
			if quickSyncFinish {
				_ = tools.UpdateLatestScanInfo(b.IsSrc, stable)
			}
		}
		time.Sleep(restIntervalInScanJob)
	}
}

func (b *Bridge) quickSync(start, end uint64) {
	count := end - start
	workers := quickSyncWorkers
	if count < 10 {
		workers = 1
	}
	step := count / workers
	wg := new(sync.WaitGroup)
	wg.Add(int(workers))
	for i := uint64(0); i < workers; i++ {
		wstt := start + i*step
		wend := start + (i+1)*step
		if i+1 == workers {
			wend = end + 1
		}
		go b.quickSyncRange(i+1, wstt, wend, wg)
	}
	wg.Wait()
	quickSyncFinish = true
}

func (b *Bridge) quickSyncRange(idx, start, end uint64, wg *sync.WaitGroup) {
	defer wg.Done()

	chainName := b.TokenConfig.BlockChain
	for h := start; h < end; {
		block, err := b.GetBlockByNumber(new(big.Int).SetUint64(h))
		if err != nil {
			log.Errorf("[scanchain] id=%v get %v block failed at height %v. err=%v", idx, chainName, h, err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		for _, tx := range block.Transactions {
			b.processTransaction(tx.String())
		}
		log.Printf("[scanchain] id=%v scanned %v block, height=%v hash=%v txs=%v", idx, chainName, h, block.Hash.String(), len(block.Transactions))
		h++
	}
}
