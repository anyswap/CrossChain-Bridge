package eth

import (
	"fmt"
	"sync"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
	"github.com/anyswap/CrossChain-Bridge/types"
)

var (
	maxScanHeight          = uint64(15000)
	retryIntervalInScanJob = 3 * time.Second
	restIntervalInScanJob  = 3 * time.Second

	quickSyncHistoryWorkers = uint64(4)

	scannedHistoryTxs *tools.CachedScannedTxs
)

// StartSwapHistoryScanJob scan job
func (b *Bridge) StartSwapHistoryScanJob() {
	if b.TokenConfig.ContractAddress == "" {
		return
	}
	log.Info("[swaphistory] start scan swap history job", "isSrc", b.IsSrc)

	isProcessed := func(txid string) bool {
		if b.IsSrc {
			return tools.IsSwapinExist(txid)
		}
		return tools.IsSwapoutExist(txid)
	}

	go b.scanFirstLoop(isProcessed)

	b.scanTransactionHistory(isProcessed)
}

func (b *Bridge) getSwapLogs(blockHeight uint64) ([]*types.RPCLog, error) {
	contractAddresses := []common.Address{common.HexToAddress(b.TokenConfig.ContractAddress)}
	var logTopics [][]common.Hash
	if b.IsSrc {
		logTopics = [][]common.Hash{getSwapinSrcLogTopics()}
	} else {
		logTopics = [][]common.Hash{getSwapoutLogTopics()}
	}
	return b.GetContractLogs(contractAddresses, logTopics, blockHeight)
}

func (b *Bridge) scanFirstLoop(isProcessed func(string) bool) {
	// first loop process all tx history no matter whether processed before
	log.Info("[scanhistory] start first scan loop", "isSrc", b.IsSrc)
	initialHeight := b.TokenConfig.InitialHeight
	latest := tools.LoopGetLatestBlockNumber(b)
	for height := latest; height+maxScanHeight > latest && height >= initialHeight; {
		logs, err := b.getSwapLogs(height)
		if err != nil {
			log.Error("[scanhistory] get swap logs error", "isSrc", b.IsSrc, "height", height, "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		for _, swaplog := range logs {
			txid := swaplog.TxHash.String()
			if !isProcessed(txid) {
				log.Debug("[scanhistory] first scan loop", "isSrc", b.IsSrc, "txid", txid, "height", height)
				b.processTransaction(txid)
			}
		}
		if height > 0 {
			height--
		} else {
			break
		}
	}

	log.Info("[scanhistory] finish first scan loop", "isSrc", b.IsSrc)
}

func (b *Bridge) scanTransactionHistory(isProcessed func(string) bool) {
	log.Info("[scanhistory] start scan swap history loop")
	height := tools.LoopGetLatestBlockNumber(b)
	intialHeight := b.TokenConfig.InitialHeight
	if height < intialHeight {
		height = intialHeight
	}
	latest := tools.LoopGetLatestBlockNumber(b)
	if latest > height {
		go b.quickSyncHistory(height, latest)
	}

	confirmations := *b.TokenConfig.Confirmations
	capacity := int(confirmations*200 + 500)
	maxCapacity := 20000
	if capacity > maxCapacity {
		capacity = maxCapacity
	}
	scannedHistoryTxs = tools.NewCachedScannedTxs(capacity)
	chainName := b.TokenConfig.BlockChain
	stable := latest
	errorSubject := fmt.Sprintf("[scanhistory] get %v swap logs failed", chainName)
	scanSubject := fmt.Sprintf("[scanhistory] scanned %v block", chainName)
	for {
		latest := tools.LoopGetLatestBlockNumber(b)
		for h := stable + 1; h <= latest; {
			logs, err := b.getSwapLogs(h)
			if err != nil {
				log.Error(errorSubject, "height", h, "err", err)
				time.Sleep(retryIntervalInScanJob)
				continue
			}
			for _, swaplog := range logs {
				txid := swaplog.TxHash.String()
				if scannedHistoryTxs.IsTxScanned(txid) || isProcessed(txid) {
					continue
				}
				b.processTransaction(txid)
				scannedHistoryTxs.CacheScannedTx(txid)
			}
			log.Info(scanSubject, "height", h)
			h++
		}
		if stable+confirmations < latest {
			stable = latest - confirmations
		}
	}
}

func (b *Bridge) quickSyncHistory(start, end uint64) {
	count := end - start
	workers := quickSyncHistoryWorkers
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
		go b.quickSyncHistoryRange(i+1, wstt, wend, wg)
	}
	wg.Wait()
}

func (b *Bridge) quickSyncHistoryRange(idx, start, end uint64, wg *sync.WaitGroup) {
	defer wg.Done()

	chainName := b.TokenConfig.BlockChain
	for h := start; h < end; {
		logs, err := b.getSwapLogs(h)
		if err != nil {
			log.Errorf("[scanhistory] id=%v get %v swap logs at height %v failed. err=%v", idx, chainName, h, err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		for _, swaplog := range logs {
			txid := swaplog.TxHash.String()
			b.processTransaction(txid)
		}
		log.Debugf("[scanhistory] id=%v scanned %v block, height=%v", idx, chainName, h)
		h++
	}
}
