package cosmos

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

var (
	quickSyncFinish  bool
	quickSyncWorkers = uint64(4)

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
	chainName := b.ChainConfig.BlockChain
	log.Infof("[scanchain] start %v scan chain job", chainName)

	start, latest := b.getStartAndLatestHeight()
	_ = tools.UpdateLatestScanInfo(b.IsSrc, start)
	log.Infof("[scanchain] start %v scan chain loop from %v latest=%v", chainName, start, latest)

	if latest > start {
		go b.quickSync(context.Background(), nil, start, latest+1)
	} else {
		quickSyncFinish = true
	}

	stable := latest
	//errorSubject := fmt.Sprintf("[scanchain] get %v block failed", chainName)
	scanSubject := fmt.Sprintf("[scanchain] scanned %v block", chainName)

	scannedRange := tools.NewCachedScannedBlocks(67)
	var quickSyncCtx context.Context
	var quickSyncCancel context.CancelFunc
	for {
		latest = tools.LoopGetLatestBlockNumber(b)
		if stable+maxScanHeight < latest {
			if quickSyncCancel != nil {
				select {
				case <-quickSyncCtx.Done():
				default:
					log.Warn("cancel quick sync range", "stable", stable, "latest", latest)
					quickSyncCancel()
				}
			}
			quickSyncCtx, quickSyncCancel = context.WithCancel(context.Background())
			go b.quickSync(quickSyncCtx, quickSyncCancel, stable+1, latest)
			stable = latest
		}

		for h := stable; h < latest; {
			start := h / 100 * 100
			end := start
			if latest-start > 100 {
				end = start + 99
			} else {
				stable = end + 1
				break
			}
			blockRange := fmt.Sprintf("%v-%v", start, end)
			log.Debug("cosmos scan loop", "range", blockRange)
			if scannedRange.IsBlockScanned(blockRange) {
				h = end + 1
				continue
			}
			txs, err := b.SearchTxs(big.NewInt(int64(start)), big.NewInt(int64(end)))
			if err != nil {
				log.Warn("Search txs in range error", "range", blockRange, "error", err)
				continue
			}
			for _, tx := range txs {
				b.processTransaction(tx)
			}
			scannedRange.CacheScannedBlock(blockRange, end)
			log.Info(scanSubject, "blockRange", blockRange, "txs", len(txs))
			h = end + 1
			stable = end + 1
		}
		if quickSyncFinish {
			_ = tools.UpdateLatestScanInfo(b.IsSrc, stable)
		}
		time.Sleep(restIntervalInScanJob)
	}
}

func (b *Bridge) quickSync(ctx context.Context, cancel context.CancelFunc, start, end uint64) {
	chainName := b.ChainConfig.BlockChain
	log.Printf("[scanchain] begin %v syncRange job. start=%v end=%v", chainName, start, end)
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
			wend = end
		}
		go b.quickSyncRange(ctx, i+1, wstt, wend, wg)
	}
	wg.Wait()
	if cancel != nil {
		cancel()
	} else {
		quickSyncFinish = true
	}
	log.Printf("[scanchain] finish %v syncRange job. start=%v end=%v", chainName, start, end)
}

func (b *Bridge) quickSyncRange(ctx context.Context, idx, start, end uint64, wg *sync.WaitGroup) {
	defer wg.Done()
	chainName := b.ChainConfig.BlockChain
	log.Printf("[scanchain] id=%v begin %v syncRange start=%v end=%v", idx, chainName, start, end)

	select {
	case <-ctx.Done():
		break
	default:
	}

	for h := start; h < end; {
		h2 := h
		if end-h > 100 {
			h2 = h + 100

			h = h2 + 1
		} else {
			h2 = end
		}
		txs, err := b.SearchTxs(big.NewInt(int64(h)), big.NewInt(int64(h2)))
		if err != nil {
			log.Warn("Search txs in range error", "range", fmt.Sprintf("%v-%v", h, h2), "error", err)
			continue
		}
		for _, tx := range txs {
			b.processTransaction(tx)
		}
		h = h2 + 1
	}

	log.Printf("[scanchain] id=%v finish %v syncRange start=%v end=%v", idx, chainName, start, end)
}
