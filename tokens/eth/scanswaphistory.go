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
	maxScanHeight          = uint64(100)
	maxFirstScanHeight     = uint64(1000)
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
	log.Infof("[swaphistory] start scan %v swap history job", b.TokenConfig.BlockChain)

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
	latest := tools.LoopGetLatestBlockNumber(b)
	minHeight := b.TokenConfig.InitialHeight
	if minHeight+maxFirstScanHeight < latest {
		minHeight = latest - maxFirstScanHeight
	}
	chainName := b.TokenConfig.BlockChain
	log.Infof("[scanFirstLoop] start %v first scan loop to min height %v", chainName, minHeight)
	for height := latest; height >= minHeight; {
		logs, err := b.getSwapLogs(height)
		if err != nil {
			log.Errorf("[scanFirstLoop] get %v swap logs error. height=%v err=%v", chainName, height, err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		for _, swaplog := range logs {
			txid := swaplog.TxHash.String()
			if !isProcessed(txid) {
				log.Tracef("[scanFirstLoop] process %v tx. txid=%v height=%v", chainName, txid, height)
				b.processTransaction(txid)
			}
		}
		if height > 0 {
			height--
		} else {
			break
		}
	}

	log.Infof("[scanFirstLoop] finish %v first scan loop to min height %v", chainName, minHeight)
}

func (b *Bridge) scanTransactionHistory(isProcessed func(string) bool) {
	chainName := b.TokenConfig.BlockChain
	latest := tools.LoopGetLatestBlockNumber(b)
	minHeight := b.TokenConfig.InitialHeight
	if minHeight+maxScanHeight < latest {
		minHeight = latest - maxScanHeight
	}
	log.Infof("[scanhistory] start %v scan swap history loop from height %v", chainName, minHeight)
	if latest > minHeight {
		go b.quickSyncHistory(minHeight, latest+1)
	}

	scannedHistoryTxs = tools.NewCachedScannedTxs(500)
	stable := latest
	errorSubject := fmt.Sprintf("[scanhistory] get %v swap logs failed", chainName)
	scanSubject := fmt.Sprintf("[scanhistory] scanned %v block", chainName)
	for {
		latest := tools.LoopGetLatestBlockNumber(b)
		for h := stable; h <= latest; {
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
			if h != stable {
				log.Trace(scanSubject, "height", h)
			}
			h++
		}
		stable = latest
		time.Sleep(restIntervalInScanJob)
	}
}

func (b *Bridge) quickSyncHistory(start, end uint64) {
	chainName := b.TokenConfig.BlockChain
	log.Printf("[scanhistory] begin %v syncRange job. start=%v end=%v", chainName, start, end)
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
	log.Printf("[scanhistory] finish %v syncRange job. start=%v end=%v", chainName, start, end)
}

func (b *Bridge) quickSyncHistoryRange(idx, start, end uint64, wg *sync.WaitGroup) {
	defer wg.Done()
	chainName := b.TokenConfig.BlockChain
	log.Printf("[scanhistory] id=%v begin %v syncRange start=%v end=%v", idx, chainName, start, end)

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

	log.Printf("[scanhistory] id=%v finish %v syncRange start=%v end=%v", idx, chainName, start, end)
}
