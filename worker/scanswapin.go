package worker

import (
	"sync"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/mongodb"
	"github.com/fsn-dev/crossChain-Bridge/params"
	"github.com/fsn-dev/crossChain-Bridge/rpc/client"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	"github.com/fsn-dev/crossChain-Bridge/tokens/btc"
	"github.com/fsn-dev/crossChain-Bridge/tokens/btc/electrs"
)

var (
	swapinScanStarter    sync.Once
	swapServerApiAddress string
)

func StartScanJob(isServer bool) error {
	go StartSwapinScanJob(isServer)
	go StartSwapoutScanJob(isServer)
	return nil
}

func StartSwapinScanJob(isServer bool) error {
	if isServer {
		return StartSwapinScanJobOnServer()
	}
	return StartSwapinScanJobOnOracle()
}

func StartSwapinScanJobOnServer() error {
	swapinScanStarter.Do(func() {
		logWorker("scanswapin", "start scan swapin job")
		bridge, ok := tokens.SrcBridge.(*btc.BtcBridge)
		if !ok {
			panic("StartSwapinScanJob: require btc bridge")
		}
		token, _ := bridge.GetTokenAndGateway()
		nowTime := time.Now().Unix()
		var lastSeenTxid string
		// first loop process all tx history no matter whether processed before
		logWorker("scanswapin", "start first scan loop")
	FIRST_LOOP:
		for {
			txHistory, err := bridge.GetTransactionHistory(token.DcrmAddress, lastSeenTxid)
			if err != nil {
				logWorkerError("scanswapin", "get tx history error", err)
				time.Sleep(retryIntervalInScanJob)
				continue
			}
			if len(txHistory) == 0 {
				break
			}
			for _, tx := range txHistory {
				if tx.Status == nil || !*tx.Status.Confirmed {
					logWorker("scanswapin", "get tx history error", "err", "tx status is not confirmed", "tx", *tx.Txid)
					continue
				}
				if int64(*tx.Status.Block_time)+maxScanLifetime < nowTime { // too old
					break FIRST_LOOP
				}
				if swap, _ := mongodb.FindSwapin(*tx.Txid); swap == nil {
					registerSwapin(tx) // add if not exist
				}
			}
			lastSeenTxid = *txHistory[len(txHistory)-1].Txid
		}

		// second loop only process unprocessed tx history
		logWorker("scanswapin", "start second scan loop")
		lastSeenTxid = ""
		for {
			rescan := false
			txHistory, err := bridge.GetTransactionHistory(token.DcrmAddress, lastSeenTxid)
			if err != nil {
				logWorkerError("scanswapin", "get tx history error", err)
				time.Sleep(retryIntervalInScanJob)
				continue
			}
			if len(txHistory) == 0 {
				rescan = true
			}
			for _, tx := range txHistory {
				if tx.Status == nil || !*tx.Status.Confirmed {
					logWorker("scanswapin", "get tx history error", "err", "tx status is not confirmed", "tx", *tx.Txid)
					continue
				}
				if swap, _ := mongodb.FindSwapin(*tx.Txid); swap != nil {
					rescan = true
					break // rescan if found exist
				}
				registerSwapin(tx)
			}
			if rescan {
				lastSeenTxid = ""
				time.Sleep(restIntervalInScanJob)
			} else {
				lastSeenTxid = *txHistory[len(txHistory)-1].Txid
			}
		}
	})
	return nil
}

func registerSwapin(tx *electrs.ElectTx) error {
	txid := *tx.Txid
	logWorker("scanswapin", "register swapin", "tx", txid)
	swap := &mongodb.MgoSwap{
		Key:       txid,
		TxId:      txid,
		Status:    mongodb.TxNotStable,
		Timestamp: time.Now().Unix(),
	}
	return mongodb.AddSwapin(swap)
}

func StartSwapinScanJobOnOracle() error {
	swapinScanStarter.Do(func() {
		logWorker("scanswapin", "start scan swapin job")
		oracleCfg := params.GetConfig().Oracle
		if oracleCfg != nil {
			swapServerApiAddress = oracleCfg.ServerApiAddress
		}
		if swapServerApiAddress == "" {
			logWorker("scanswapin", "stop scan swapin job as no Oracle.ServerApiAddress configed")
			return
		}
		bridge, ok := tokens.SrcBridge.(*btc.BtcBridge)
		if !ok {
			panic("StartSwapinScanJob: require btc bridge")
		}

		var (
			token, _ = bridge.GetTokenAndGateway()

			latestProcessed string
			lastSeenTxid    string
			first           string
		)

		// init latestProcessed
		for {
			txHistory, err := bridge.GetTransactionHistory(token.DcrmAddress, lastSeenTxid)
			if err != nil {
				logWorkerError("scanswapin", "get tx history error", err)
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
			txHistory, err := bridge.GetTransactionHistory(token.DcrmAddress, lastSeenTxid)
			if err != nil {
				logWorkerError("scanswapin", "get tx history error", err)
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
					logWorker("scanswapin", "get tx history error", "err", "tx status is not confirmed", "tx", *tx.Txid)
					continue
				}
				if *tx.Txid == latestProcessed {
					rescan = true
					break
				}
				postRegisterSwapin(tx)
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
	})
	return nil
}

func postRegisterSwapin(tx *electrs.ElectTx) error {
	txid := *tx.Txid
	logWorker("scanswapin", "post register swapin", "tx", txid)
	var result interface{}
	return client.RpcPost(&result, swapServerApiAddress, "swap.Swapin", txid)
}
