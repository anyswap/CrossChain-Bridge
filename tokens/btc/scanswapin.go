package btc

import (
	"sync"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/mongodb"
	"github.com/fsn-dev/crossChain-Bridge/params"
	"github.com/fsn-dev/crossChain-Bridge/rpc/client"
	"github.com/fsn-dev/crossChain-Bridge/tokens/btc/electrs"
)

var (
	swapinScanStarter    sync.Once
	swapServerApiAddress string

	maxScanLifetime        = int64(3 * 24 * 3600)
	restIntervalInScanJob  = 10 * time.Second
	retryIntervalInScanJob = 10 * time.Second
)

func (b *BtcBridge) StartSwapinScanJob(isServer bool) error {
	swapinScanStarter.Do(func() {
		if isServer {
			b.StartSwapinScanJobOnServer()
		} else {
			b.StartSwapinScanJobOnOracle()
		}
	})
	return nil
}

func (b *BtcBridge) StartSwapinScanJobOnServer() error {
	log.Info("[scanswapin] start scan swapin job")
	token := b.TokenConfig
	nowTime := time.Now().Unix()
	var lastSeenTxid string
	// first loop process all tx history no matter whether processed before
	log.Info("[scanswapin] start first scan loop")
FIRST_LOOP:
	for {
		txHistory, err := b.GetTransactionHistory(token.DcrmAddress, lastSeenTxid)
		if err != nil {
			log.Error("[scanswapin] get tx history error", "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		if len(txHistory) == 0 {
			break
		}
		for _, tx := range txHistory {
			if tx.Status == nil || !*tx.Status.Confirmed {
				log.Info("[scanswapin] get tx history error", "err", "tx status is not confirmed", "tx", *tx.Txid)
				continue
			}
			if int64(*tx.Status.Block_time)+maxScanLifetime < nowTime { // too old
				break FIRST_LOOP
			}
			if swap, _ := mongodb.FindSwapin(*tx.Txid); swap == nil {
				b.registerSwapin(tx) // add if not exist
			}
		}
		lastSeenTxid = *txHistory[len(txHistory)-1].Txid
	}

	// second loop only process unprocessed tx history
	log.Info("[scanswapin] start second scan loop")
	lastSeenTxid = ""
	for {
		rescan := false
		txHistory, err := b.GetTransactionHistory(token.DcrmAddress, lastSeenTxid)
		if err != nil {
			log.Error("[scanswapin] get tx history error", "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		if len(txHistory) == 0 {
			rescan = true
		}
		for _, tx := range txHistory {
			if tx.Status == nil || !*tx.Status.Confirmed {
				log.Info("[scanswapin] get tx history error", "err", "tx status is not confirmed", "tx", *tx.Txid)
				continue
			}
			if swap, _ := mongodb.FindSwapin(*tx.Txid); swap != nil {
				rescan = true
				break // rescan if found exist
			}
			b.registerSwapin(tx)
		}
		if rescan {
			lastSeenTxid = ""
			time.Sleep(restIntervalInScanJob)
		} else {
			lastSeenTxid = *txHistory[len(txHistory)-1].Txid
		}
	}
	return nil
}

func (b *BtcBridge) registerSwapin(tx *electrs.ElectTx) error {
	txid := *tx.Txid
	log.Info("[scanswapin] register swapin", "tx", txid)
	swap := &mongodb.MgoSwap{
		Key:       txid,
		TxId:      txid,
		Status:    mongodb.TxNotStable,
		Timestamp: time.Now().Unix(),
	}
	return mongodb.AddSwapin(swap)
}

func (b *BtcBridge) StartSwapinScanJobOnOracle() error {
	log.Info("[scanswapin] start scan swapin job")
	oracleCfg := params.GetConfig().Oracle
	if oracleCfg != nil {
		swapServerApiAddress = oracleCfg.ServerApiAddress
	}
	if swapServerApiAddress == "" {
		log.Info("[scanswapin] stop scan swapin job as no Oracle.ServerApiAddress configed")
		return nil
	}

	var (
		token = b.TokenConfig

		latestProcessed string
		lastSeenTxid    string
		first           string
	)

	// init latestProcessed
	for {
		txHistory, err := b.GetTransactionHistory(token.DcrmAddress, lastSeenTxid)
		if err != nil {
			log.Error("[scanswapin] get tx history error", "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		if len(txHistory) == 0 {
			time.Sleep(restIntervalInScanJob)
		} else {
			latestProcessed = *txHistory[len(txHistory)-1].Txid
			break
		}
	}

	for {
		rescan := false
		txHistory, err := b.GetTransactionHistory(token.DcrmAddress, lastSeenTxid)
		if err != nil {
			log.Error("[scanswapin] get tx history error", "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		if len(txHistory) == 0 {
			rescan = true
		} else if first == "" {
			first = *txHistory[0].Txid
		}
		for _, tx := range txHistory {
			if tx.Status == nil || !*tx.Status.Confirmed {
				log.Info("[scanswapin] get tx history error", "err", "tx status is not confirmed", "tx", *tx.Txid)
				continue
			}
			if *tx.Txid == latestProcessed {
				rescan = true
				break
			}
			b.postRegisterSwapin(tx)
		}
		if rescan {
			lastSeenTxid = ""
			if first != "" {
				latestProcessed = first
				first = ""
			}
			time.Sleep(restIntervalInScanJob)
		} else {
			lastSeenTxid = *txHistory[len(txHistory)-1].Txid
		}
	}
	return nil
}

func (b *BtcBridge) postRegisterSwapin(tx *electrs.ElectTx) error {
	txid := *tx.Txid
	log.Info("[scanswapin] post register swapin", "tx", txid)
	var result interface{}
	return client.RpcPost(&result, swapServerApiAddress, "swap.Swapin", txid)
}
