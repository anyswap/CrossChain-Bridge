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
	var (
		token   = b.TokenConfig
		nowTime = time.Now().Unix()
		rescan  = true

		lastSeenTxid string
		txHistory    []*electrs.ElectTx
		err          error
	)
	// first loop process all tx history no matter whether processed before
	log.Info("[scanswapin] start first scan loop")
FIRST_LOOP:
	for {
		txHistory, err = b.GetTransactionHistory(token.DcrmAddress, lastSeenTxid)
		if err != nil {
			log.Error("[scanswapin] get tx history error", "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		if len(txHistory) == 0 {
			break
		}
		for _, tx := range txHistory {
			if tx.Status.Block_time != nil &&
				int64(*tx.Status.Block_time)+maxScanLifetime < nowTime { // too old
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
		if rescan {
			txHistory, err = b.GetPoolTransactions(token.DcrmAddress)
		} else {
			txHistory, err = b.GetTransactionHistory(token.DcrmAddress, lastSeenTxid)
		}
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

func getSwapServerApiAddress() string {
	oracleCfg := params.GetConfig().Oracle
	if oracleCfg != nil {
		return oracleCfg.ServerApiAddress
	}
	return ""
}

func (b *BtcBridge) StartSwapinScanJobOnOracle() error {
	log.Info("[scanswapin] start scan swapin job")

	swapServerApiAddress = getSwapServerApiAddress()
	if swapServerApiAddress == "" {
		log.Info("[scanswapin] stop scan swapin job as no Oracle.ServerApiAddress configed")
		return nil
	}

	var (
		token  = b.TokenConfig
		rescan = true

		txHistory       []*electrs.ElectTx
		latestProcessed string
		lastSeenTxid    string
		first           string
		err             error
	)

	for {
		if rescan {
			txHistory, err = b.GetPoolTransactions(token.DcrmAddress)
		} else {
			txHistory, err = b.GetTransactionHistory(token.DcrmAddress, lastSeenTxid)
			if latestProcessed == "" && len(txHistory) > 0 {
				latestProcessed = *txHistory[len(txHistory)-1].Txid
			}
		}
		if err != nil {
			log.Error("[scanswapin] get tx history error", "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		if len(txHistory) == 0 {
			rescan = true
		} else {
			if rescan {
				rescan = false
			}
			if first == "" {
				first = *txHistory[0].Txid
			}
		}
		for _, tx := range txHistory {
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
