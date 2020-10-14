package btc

import (
	"fmt"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

var (
	maxScanHeight          = uint64(50)
	maxFirstScanHeight     = uint64(1000)
	retryIntervalInScanJob = 3 * time.Second
	restIntervalInScanJob  = 3 * time.Second
)

// StartSwapHistoryScanJob scan job
func (b *Bridge) StartSwapHistoryScanJob() {
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

func (b *Bridge) scanFirstLoop(isProcessed func(string) bool) {
	// first loop process all tx history no matter whether processed before
	latest := tools.LoopGetLatestBlockNumber(b)
	minHeight := b.TokenConfig.InitialHeight
	if minHeight+maxFirstScanHeight < latest {
		minHeight = latest - maxFirstScanHeight
	}
	chainName := b.TokenConfig.BlockChain
	log.Infof("[scanFirstLoop] start %v first scan loop to min height %v", chainName, minHeight)

	lastSeenTxid := ""

FIRST_LOOP:
	for {
		txHistory, err := b.GetTransactionHistory(b.TokenConfig.DepositAddress, lastSeenTxid)
		if err != nil {
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		if len(txHistory) == 0 {
			break
		}
		for _, tx := range txHistory {
			if tx.Status.BlockHeight == nil {
				continue
			}
			height := *tx.Status.BlockHeight
			if height < minHeight {
				break FIRST_LOOP
			}
			txid := *tx.Txid
			if !isProcessed(txid) {
				log.Tracef("[scanFirstLoop] process %v tx. txid=%v height=%v", chainName, txid, height)
				_ = b.processSwapin(txid)
			}
		}
		lastSeenTxid = *txHistory[len(txHistory)-1].Txid
	}

	log.Infof("[scanFirstLoop] finish %v first scan loop to min height %v", chainName, minHeight)
}

func (b *Bridge) scanTransactionHistory(isProcessed func(string) bool) {
	var (
		lastSeenTxid = ""
		rescan       = true
	)

	latest := tools.LoopGetLatestBlockNumber(b)
	minHeight := b.TokenConfig.InitialHeight
	if minHeight+maxScanHeight < latest {
		minHeight = latest - maxScanHeight
	}

	chainName := b.TokenConfig.BlockChain
	errorSubject := fmt.Sprintf("[scanhistory] get %v tx history failed", chainName)
	scanSubject := fmt.Sprintf("[scanhistory] scanned %v tx", chainName)
	log.Infof("[scanhistory] start %v scan swap history loop from height %v", chainName, minHeight)

	for {
		txHistory, err := b.GetTransactionHistory(b.TokenConfig.DepositAddress, lastSeenTxid)
		if err != nil {
			log.Error(errorSubject, "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		if len(txHistory) == 0 {
			rescan = true
		} else if rescan {
			rescan = false
		}
		for _, tx := range txHistory {
			if tx.Status.BlockHeight == nil {
				continue
			}
			height := *tx.Status.BlockHeight
			if height < minHeight {
				rescan = true
				break
			}
			txid := *tx.Txid
			if isProcessed(txid) {
				rescan = true
				break // rescan if already processed
			}
			log.Trace(scanSubject, "txid", txid, "height", height)
			_ = b.processSwapin(txid)
		}
		if rescan {
			lastSeenTxid = ""
			time.Sleep(restIntervalInScanJob)
		} else {
			lastSeenTxid = *txHistory[len(txHistory)-1].Txid
		}
	}
}
