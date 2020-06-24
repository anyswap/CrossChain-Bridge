package btc

import (
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

var (
	maxScanLifetime        = int64(3 * 24 * 3600)
	retryIntervalInScanJob = 3 * time.Second
	restIntervalInScanJob  = 3 * time.Second
)

// StartSwapHistoryScanJob scan job
func (b *Bridge) StartSwapHistoryScanJob() {
	log.Info("[scanhistory] start scan swap history job", "isSrc", b.IsSrc)

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
	log.Info("[scanhistory] start first scan loop", "isSrc", b.IsSrc)
	var (
		nowTime       = time.Now().Unix()
		lastSeenTxid  = ""
		initialHeight = b.TokenConfig.InitialHeight
	)

	isTooOld := func(time *uint64) bool {
		return time != nil && int64(*time)+maxScanLifetime < nowTime
	}

FIRST_LOOP:
	for {
		txHistory, err := b.GetTransactionHistory(b.TokenConfig.DcrmAddress, lastSeenTxid)
		if err != nil {
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		if len(txHistory) == 0 {
			break
		}
		for _, tx := range txHistory {
			if tx.Status.BlockHeight != nil && *tx.Status.BlockHeight < initialHeight {
				break FIRST_LOOP
			}
			if isTooOld(tx.Status.BlockTime) {
				break FIRST_LOOP
			}
			txid := *tx.Txid
			if !isProcessed(txid) {
				_ = b.processSwapin(txid)
			}
		}
		lastSeenTxid = *txHistory[len(txHistory)-1].Txid
	}

	log.Info("[scanhistory] finish first scan loop", "isSrc", b.IsSrc)
}

func (b *Bridge) scanTransactionHistory(isProcessed func(string) bool) {
	log.Info("[scanhistory] start scan swap history loop", "isSrc", b.IsSrc)
	var (
		lastSeenTxid  = ""
		rescan        = true
		initialHeight = b.TokenConfig.InitialHeight
	)

	for {
		txHistory, err := b.GetTransactionHistory(b.TokenConfig.DcrmAddress, lastSeenTxid)
		if err != nil {
			log.Error("[scanhistory] get tx history error", "isSrc", b.IsSrc, "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		if len(txHistory) == 0 {
			rescan = true
		} else if rescan {
			rescan = false
		}
		log.Info("[scanhistory] scan swap history", "isSrc", b.IsSrc, "count", len(txHistory))
		for _, tx := range txHistory {
			if tx.Status.BlockHeight != nil && *tx.Status.BlockHeight < initialHeight {
				rescan = true
				break
			}
			txid := *tx.Txid
			if isProcessed(txid) {
				rescan = true
				break // rescan if already processed
			}
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
