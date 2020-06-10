package btc

import (
	"sync"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/mongodb"
	"github.com/fsn-dev/crossChain-Bridge/params"
	"github.com/fsn-dev/crossChain-Bridge/rpc/client"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
)

var (
	swapinScanStarter    sync.Once
	swapServerAPIAddress string
	oracleLatestSeenTx   string

	maxScanLifetime        = int64(3 * 24 * 3600)
	retryIntervalInScanJob = 3 * time.Second
	restIntervalInScanJob  = 3 * time.Second
)

// StartSwapinScanJob scann job
func (b *Bridge) StartSwapinScanJob(isServer bool) {
	swapinScanStarter.Do(func() {
		if isServer {
			b.startSwapinScanJobOnServer()
		} else {
			b.startSwapinScanJobOnOracle()
		}
	})
}

func (b *Bridge) startSwapinScanJobOnServer() {
	log.Info("[scanswapin] server start scan swapin job")

	isProcessed := func(txid string) bool {
		swap, _ := mongodb.FindSwapin(txid)
		return swap != nil
	}

	go b.scanTransactionPool(true)

	go b.scanFirstLoop(true, isProcessed)

	log.Info("[scanswapin] server start second scan loop")
	b.scanTransactionHistory(true, isProcessed)
}

func (b *Bridge) processSwapin(txid string, isServer bool) error {
	swapInfo, err := b.VerifyTransaction(txid, true)
	if !tokens.ShouldRegisterSwapForError(err) {
		return err
	}
	if isServer {
		return b.registerSwapin(txid, swapInfo.Bind)
	}
	if !b.isSwapinExistByQuery(txid) {
		return b.postRegisterSwapin(txid)
	}
	return nil
}

func (b *Bridge) isSwapinExistByQuery(txid string) bool {
	var result interface{}
	_ = client.RPCPost(&result, swapServerAPIAddress, "swap.GetSwapin", txid)
	return result != nil
}

func (b *Bridge) registerSwapin(txid string, bind string) error {
	log.Info("[scanswapin] register swapin", "tx", txid, "bind", bind)
	swap := &mongodb.MgoSwap{
		Key:       txid,
		TxType:    uint32(tokens.SwapinTx),
		Bind:      bind,
		TxID:      txid,
		Status:    mongodb.TxNotStable,
		Timestamp: time.Now().Unix(),
	}
	return mongodb.AddSwapin(swap)
}

func (b *Bridge) postRegisterSwapin(txid string) error {
	log.Info("[scanswapin] post register swapin", "tx", txid)
	var result interface{}
	return client.RPCPost(&result, swapServerAPIAddress, "swap.Swapin", txid)
}

func getSwapServerAPIAddress() string {
	oracleCfg := params.GetConfig().Oracle
	if oracleCfg != nil {
		return oracleCfg.ServerAPIAddress
	}
	return ""
}

func (b *Bridge) startSwapinScanJobOnOracle() {
	log.Info("[scanswapin] oracle start scan swapin job")

	// init swapServerAPIAddress
	swapServerAPIAddress = getSwapServerAPIAddress()
	if swapServerAPIAddress == "" {
		log.Info("[scanswapin] stop scan swapin job as no Oracle.ServerAPIAddress configed")
		return
	}

	go b.scanTransactionPool(false)

	// init oracleLatestSeenTx
	for {
		txHistory, err := b.GetTransactionHistory(b.TokenConfig.DcrmAddress, "")
		if err != nil {
			log.Error("[scanswapin] get tx history error", "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		if len(txHistory) != 0 {
			oracleLatestSeenTx = *txHistory[len(txHistory)-1].Txid
			break
		}
		time.Sleep(restIntervalInScanJob)
	}

	isProcessed := func(txid string) bool {
		return txid == oracleLatestSeenTx
	}
	b.scanTransactionHistory(false, isProcessed)
}

func (b *Bridge) scanFirstLoop(isServer bool, isProcessed func(string) bool) {
	// first loop process all tx history no matter whether processed before
	log.Info("[scanswapin] start first scan loop")
	var (
		nowTime      = time.Now().Unix()
		lastSeenTxid = ""
	)

	isTooOld := func(time *uint64) bool {
		return time != nil && int64(*time)+maxScanLifetime < nowTime
	}

	for {
		txHistory, err := b.GetTransactionHistory(b.TokenConfig.DcrmAddress, lastSeenTxid)
		if err != nil {
			log.Error("[scanswapin] get tx history error", "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		if len(txHistory) == 0 {
			break
		}
		for _, tx := range txHistory {
			if isTooOld(tx.Status.BlockTime) {
				return
			}
			txid := *tx.Txid
			if !isProcessed(txid) {
				_ = b.processSwapin(txid, isServer)
			}
		}
		lastSeenTxid = *txHistory[len(txHistory)-1].Txid
	}
}

func (b *Bridge) scanTransactionHistory(isServer bool, isProcessed func(string) bool) {
	log.Info("[scanswapin] start scan tx history loop")
	var (
		lastSeenTxid  = ""
		firstSeenTxid = ""
		rescan        = true
	)

	for {
		txHistory, err := b.GetTransactionHistory(b.TokenConfig.DcrmAddress, lastSeenTxid)
		if err != nil {
			log.Error("[scanswapin] get tx history error", "err", err)
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
			if !isServer && firstSeenTxid == "" {
				firstSeenTxid = txid
			}
			if isProcessed(txid) {
				rescan = true
				break // rescan if already processed
			}
			_ = b.processSwapin(txid, isServer)
		}
		if rescan {
			lastSeenTxid = ""
			if !isServer && firstSeenTxid != "" {
				oracleLatestSeenTx = firstSeenTxid
			}
			time.Sleep(restIntervalInScanJob)
		} else {
			lastSeenTxid = *txHistory[len(txHistory)-1].Txid
		}
	}
}

func (b *Bridge) scanTransactionPool(isServer bool) {
	log.Info("[scanswapin] start scan tx pool loop")
	for {
		txids, err := b.GetPoolTxidList()
		if err != nil {
			log.Error("[scanswapin] get pool tx list error", "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		for _, txid := range txids {
			_ = b.processSwapin(txid, isServer)
		}
		time.Sleep(restIntervalInScanJob)
	}
}
