package solana

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	bin "github.com/dfuse-io/binary"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
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

func (b *Bridge) StartChainTransactionScanJob() {
	chainName := b.ChainConfig.BlockChain
	log.Infof("[scanchain] start %v scan chain job", chainName)

	// get addresses
	pairIDs := tokens.GetAllPairIDs()
	if len(pairIDs) == 0 {
		return
	}

	for _, pairID := range pairIDs {
		tokenCfg := tokens.GetTokenConfig(pairID, b.IsSrc)
		depositAddress := tokenCfg.DepositAddress

		// For every address, start a scanner
		go func(depositAddress string) {
			// get scanned tx
			scanned := tools.GetLatestScannedSolanaTxid(depositAddress)
			for {
				txs, err := b.SearchTxs(depositAddress, scanned, "")
				if err != nil {
					log.Warn("Scan solana tx error", "address", depositAddress, "error", err)
					continue
				}
				if len(txs) > 0 {
					scanned = txs[0]
					err := tools.UpdateLatestScannedSolanaTxid(depositAddress, scanned)
					if err != nil {
						log.Warn("UpdateLatestScannedSolanaTxid error", "address", depositAddress, "txid", scanned, "error", err)
					}
				}
				for _, txid := range txs {
					go b.processTransactionWithTxid(txid)
				}
			}
		}(depositAddress)
	}
}

// StartChainTransactionScanJob2 scan job
func (b *Bridge) StartChainTransactionScanJob2() {
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
	errorSubject := fmt.Sprintf("[scanchain] get %v block failed", chainName)
	scanSubject := fmt.Sprintf("[scanchain] scanned %v block", chainName)

	scannedBlocks := tools.NewCachedScannedBlocks(67)
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
		for h := stable; h <= latest; {
			block, err := b.GetBlockByNumber(new(big.Int).SetUint64(h))
			if err != nil {
				log.Error(errorSubject, "height", h, "err", err)
				time.Sleep(retryIntervalInScanJob)
				continue
			}
			blockHash := block.Blockhash.String()
			if scannedBlocks.IsBlockScanned(blockHash) {
				h++
				continue
			}
			for _, entry := range block.Transactions {
				tx := &GetConfirmedTransactonResult{
					Transaction: entry.Transaction,
					Meta:        entry.Meta,
					Slot:        bin.Uint64(h),
					BlockTime:   block.BlockTime,
				}
				b.processTransaction(tx)
			}
			scannedBlocks.CacheScannedBlock(blockHash, h)
			log.Info(scanSubject, "blockHash", blockHash, "height", h, "txs", len(block.Transactions))
			h++
		}
		stable = latest
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

	for h := start; h < end; {
		select {
		case <-ctx.Done():
			break
		default:
		}
		block, err := b.GetBlockByNumber(new(big.Int).SetUint64(h))
		if err != nil {
			log.Errorf("[scanchain] id=%v get %v block failed at height %v. err=%v", idx, chainName, h, err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		for _, entry := range block.Transactions {
			tx := &GetConfirmedTransactonResult{
				Transaction: entry.Transaction,
				Meta:        entry.Meta,
				Slot:        bin.Uint64(h),
				BlockTime:   block.BlockTime,
			}
			b.processTransaction(tx)
		}
		log.Tracef("[scanchain] id=%v scanned %v block, height=%v hash=%v txs=%v", idx, chainName, h, block.Blockhash.String(), len(block.Transactions))
		h++
	}

	log.Printf("[scanchain] id=%v finish %v syncRange start=%v end=%v", idx, chainName, start, end)
}
