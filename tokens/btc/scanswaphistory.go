package btc

import (
	"time"

	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/mongodb"
)

var (
	maxScanLifetime        = int64(3 * 24 * 3600)
	retryIntervalInScanJob = 3 * time.Second
	restIntervalInScanJob  = 3 * time.Second
)

func (b *Bridge) StartSwapHistoryScanJob() {
	log.Info("[scanhistory] start scan history job", "isSrc", b.IsSrc)

	isProcessed := func(txid string) bool {
		swap, _ := mongodb.FindSwapin(txid)
		return swap != nil
	}

	go b.scanFirstLoop(isProcessed)

	b.scanTransactionHistory(isProcessed)
}

func (b *Bridge) scanFirstLoop(isProcessed func(string) bool) {
	// first loop process all tx history no matter whether processed before
	log.Info("[scanhistory] start first scan loop", "isSrc", b.IsSrc)
	var (
		nowTime      = time.Now().Unix()
		lastSeenTxid = ""
	)

	isTooOld := func(time *uint64) bool {
		return time != nil && int64(*time)+maxScanLifetime < nowTime
	}

FIRST_LOOP:
	for {
		txHistory, err := b.GetTransactionHistory(b.TokenConfig.DcrmAddress, lastSeenTxid)
		if err != nil {
			//log.Trace("[scanhistory] get tx history error",  "isSrc", b.IsSrc,"err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		if len(txHistory) == 0 {
			break
		}
		for _, tx := range txHistory {
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
	log.Info("[scanhistory] start scan tx history loop", "isSrc", b.IsSrc)
	var (
		lastSeenTxid = ""
		rescan       = true
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
		for _, tx := range txHistory {
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
