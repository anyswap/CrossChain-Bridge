package eth

import (
	"sync"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/mongodb"
	"github.com/fsn-dev/crossChain-Bridge/params"
	"github.com/fsn-dev/crossChain-Bridge/rpc/client"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	"github.com/fsn-dev/crossChain-Bridge/types"
)

var (
	swapoutScanStarter   sync.Once
	swapServerApiAddress string
	oracleLatestScanned  uint64

	maxScanHeight          = uint64(15000)
	retryIntervalInScanJob = 3 * time.Second
	restIntervalInScanJob  = 3 * time.Second
)

func (b *EthBridge) StartSwapoutScanJob(isServer bool) error {
	swapoutScanStarter.Do(func() {
		if isServer {
			b.StartSwapoutScanJobOnServer()
		} else {
			b.StartSwapoutScanJobOnOracle()
		}
	})
	return nil
}

func (b *EthBridge) StartSwapoutScanJobOnServer() error {
	log.Info("[scanswapout] start scan swapout job")

	isProcessed := func(txid string, txheight uint64) bool {
		swap, _ := mongodb.FindSwapout(txid)
		return swap != nil
	}

	go b.scanTransactionPool(true)

	go b.scanFirstLoop(true, isProcessed)

	log.Info("[scanswapout] start second scan loop")
	return b.scanTransactionHistory(true, isProcessed)
}

func (b *EthBridge) processSwapout(txid string, isServer bool) error {
	swapInfo, err := b.VerifyTransaction(txid, true)
	if !tokens.ShouldRegisterSwapForError(err) {
		return err
	}
	if isServer {
		return b.registerSwapout(txid, swapInfo.Bind)
	}
	if !b.IsSwapoutExistByQuery(txid) {
		return b.postRegisterSwapout(txid)
	}
	return nil
}

func (b *EthBridge) IsSwapoutExistByQuery(txid string) bool {
	var result interface{}
	client.RpcPost(&result, swapServerApiAddress, "swap.GetSwapout", txid)
	return result != nil
}

func (b *EthBridge) registerSwapout(txid string, bind string) error {
	log.Info("[scanswapout] register swapout", "tx", txid)
	swap := &mongodb.MgoSwap{
		Key:       txid,
		TxId:      txid,
		TxType:    mongodb.SwapoutTx,
		Bind:      bind,
		Status:    mongodb.TxNotStable,
		Timestamp: time.Now().Unix(),
	}
	return mongodb.AddSwapout(swap)
}

func (b *EthBridge) postRegisterSwapout(txid string) error {
	log.Info("[scanswapout] post register swapout", "tx", txid)
	var result interface{}
	return client.RpcPost(&result, swapServerApiAddress, "swap.Swapout", txid)
}

func getSwapServerApiAddress() string {
	oracleCfg := params.GetConfig().Oracle
	if oracleCfg != nil {
		return oracleCfg.ServerApiAddress
	}
	return ""
}

func (b *EthBridge) getSwapoutLogs(blockHeight uint64) ([]*types.RPCLog, error) {
	token := b.TokenConfig
	contractAddress := token.ContractAddress
	logTopic := tokens.LogSwapoutTopic
	return b.GetContractLogs(contractAddress, logTopic, blockHeight)
}

func (b *EthBridge) getLatestHeight() uint64 {
	for {
		latest, err := b.GetLatestBlockNumber()
		if err == nil {
			return latest
		}
		log.Error("[scanswapout] get latest block number error", "err", err)
		time.Sleep(retryIntervalInScanJob)
	}
	return 0
}

func (b *EthBridge) StartSwapoutScanJobOnOracle() error {
	log.Info("[scanswapout] start scan swapout job")

	// init swapServerApiAddress
	swapServerApiAddress = getSwapServerApiAddress()
	if swapServerApiAddress == "" {
		log.Info("[scanswapout] stop scan swapout job as no Oracle.ServerApiAddress configed")
		return nil
	}

	go b.scanTransactionPool(false)

	latest := b.getLatestHeight()
	confirmations := *b.TokenConfig.Confirmations
	oracleLatestScanned = latest - confirmations

	isProcessed := func(txid string, txheight uint64) bool {
		return txheight <= oracleLatestScanned
	}
	return b.scanTransactionHistory(false, isProcessed)
}

func (b *EthBridge) scanFirstLoop(isServer bool, isProcessed func(string, uint64) bool) error {
	// first loop process all tx history no matter whether processed before
	log.Info("[scanswapout] start first scan loop")
	latest := b.getLatestHeight()
	for height := latest; height+maxScanHeight > latest; {
		logs, err := b.getSwapoutLogs(height)
		if err != nil {
			log.Error("[scanswapout] first scan get swapout logs error", "height", height, "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		for _, log := range logs {
			txid := log.TxHash.String()
			if !isProcessed(txid, height) {
				b.processSwapout(txid, isServer)
			}
		}
		height--
	}
	return nil
}

func (b *EthBridge) scanTransactionHistory(isServer bool, isProcessed func(string, uint64) bool) error {
	log.Info("[scanswapout] start scan tx history loop")
	var (
		confirmations = *b.TokenConfig.Confirmations
		oracleLatest  uint64
		height        uint64
		rescan        = true
	)
	for {
		if rescan {
			height = b.getLatestHeight()
			if !isServer {
				oracleLatest = height - confirmations
			}
		}
		logs, err := b.getSwapoutLogs(height)
		if err != nil {
			log.Error("[scanswapout] get swapout logs error", "height", height, "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		for _, log := range logs {
			txid := log.TxHash.String()
			if isProcessed(txid, height) {
				if !isServer {
					oracleLatestScanned = oracleLatest
				}
				rescan = true
				break // rescan if already processed
			}
			b.processSwapout(txid, isServer)
		}
		if rescan {
			time.Sleep(restIntervalInScanJob)
		} else {
			height--
		}
	}
	return nil
}

func (b *EthBridge) scanTransactionPool(isServer bool) error {
	log.Info("[scanswapout] start scan tx pool loop")
	for {
		txs, err := b.GetPendingTransactions()
		if err != nil {
			log.Error("[scanswapout] get pool txs error", "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		for _, tx := range txs {
			txid := tx.Hash.String()
			b.processSwapout(txid, isServer)
		}
		time.Sleep(restIntervalInScanJob)
	}
	return nil
}
