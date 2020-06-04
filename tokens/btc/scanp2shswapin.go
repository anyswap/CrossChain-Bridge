package btc

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/mongodb"
	"github.com/fsn-dev/crossChain-Bridge/rpc/client"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
)

var (
	p2shSwapinScanStarter    sync.Once
	p2shSwapServerApiAddress string

	scannedBlocks  = newCachedScannedBlocks(10)
	scanStatusFile *os.File

	restIntervalInP2shScanJob = 10 * time.Second
)

func (b *BtcBridge) StartP2shSwapinScanJob(isServer bool) error {
	p2shSwapinScanStarter.Do(func() {
		if isServer {
			b.StartP2shSwapinScanJobOnServer()
		} else {
			b.StartP2shSwapinScanJobOnOracle()
		}
	})
	return nil
}

func (b *BtcBridge) StartP2shSwapinScanJobOnServer() error {
	log.Info("[scanp2sh] start scan p2sh swapin job")

	go b.scanP2shInTransactionPool(true)

	return b.scanP2shTransactionHistory(true)
}

func (b *BtcBridge) StartP2shSwapinScanJobOnOracle() error {
	log.Info("[scanp2sh] start scan p2sh swapin job")

	// init p2shSwapServerApiAddress
	p2shSwapServerApiAddress = getSwapServerApiAddress()
	if p2shSwapServerApiAddress == "" {
		log.Info("[scanp2sh] stop scan p2sh swapin job as no Oracle.ServerApiAddress configed")
		return nil
	}

	go b.scanP2shInTransactionPool(false)

	return b.scanP2shTransactionHistory(false)
}

func (b *BtcBridge) processP2shSwapin(txid string, bind string, isServer bool) error {
	if isServer {
		return b.registerP2shSwapin(txid)
	}
	if !b.IsSwapinExistByQuery(txid) {
		return b.postRegisterP2shSwapin(txid, bind)
	}
	return nil
}

func (b *BtcBridge) registerP2shSwapin(txid string) error {
	log.Info("[scanp2sh] register swapin", "tx", txid)
	swap := &mongodb.MgoSwap{
		Key:       txid,
		TxId:      txid,
		Status:    mongodb.TxNotStable,
		Timestamp: time.Now().Unix(),
	}
	return mongodb.AddSwapin(swap)
}

func (b *BtcBridge) postRegisterP2shSwapin(txid string, bind string) error {
	log.Info("[scanp2sh] post register p2sh swapin", "tx", txid, "bind", bind)

	args := map[string]interface{}{
		"txid": txid,
		"bind": bind,
	}
	var result interface{}
	return client.RpcPost(&result, p2shSwapServerApiAddress, "swap.P2shSwapin", args)
}

func OpenScanStatusFile() {
	if scanStatusFile != nil {
		return
	}
	var err error
	execDir, _ := common.ExecuteDir()
	scanStatusFilePath := common.AbsolutePath(execDir, "scanstatus")
	scanStatusFile, err = os.OpenFile(scanStatusFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	log.Info("[scanp2sh] OpenScanStatusFile", "path", scanStatusFilePath)
}

func GetLatestScanHeight() uint64 {
	buf := make([]byte, 33)
	n, err := scanStatusFile.ReadAt(buf, 0)
	fileContent := strings.TrimSpace(string(buf[:n]))
	height, err := common.GetUint64FromStr(fileContent)
	if err != nil {
		log.Error("parse scanstatus file failed", "err", err)
		return 0
	}
	log.Info("[scanp2sh] GetLatestScanHeight", "height", height)
	return height
}

func UpdateLatestScanHeight(height uint64) error {
	fileContent := fmt.Sprintf("%d", height)
	scanStatusFile.Seek(0, 0)
	scanStatusFile.WriteString(fileContent)
	scanStatusFile.Sync()
	log.Info("[scanp2sh] UpdateLatestScanHeight", "height", height)
	return nil
}

func (b *BtcBridge) getLatestBlock() uint64 {
	for {
		latest, err := b.GetLatestBlockNumber()
		if err != nil {
			log.Error("[scanp2sh] get latest block error", "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		return latest
	}
}

func (b *BtcBridge) scanP2shTransactionHistory(isServer bool) error {
	OpenScanStatusFile()
	startHeight := GetLatestScanHeight()
	confirmations := *b.TokenConfig.Confirmations

	var height uint64
	if startHeight == 0 {
		latest := b.getLatestBlock()
		if latest > confirmations {
			height = latest - confirmations
		}
	} else {
		height = startHeight
	}
	log.Info("[scanp2sh] start scan tx history loop", "start", height)

	for {
		latest := b.getLatestBlock()
		for h := height + 1; h <= latest; {
			blockHash, err := b.GetBlockHash(h)
			if err != nil {
				log.Error("[scanp2sh] get block hash error", "height", h, "err", err)
				time.Sleep(retryIntervalInScanJob)
				continue
			}
			if scannedBlocks.isBlockScanned(blockHash) {
				h++
				continue
			}
			txids, err := b.GetBlockTxids(blockHash)
			if err != nil {
				log.Error("[scanp2sh] get block transactions error", "height", h, "blockHash", blockHash, "err", err)
				time.Sleep(retryIntervalInScanJob)
				continue
			}
			for _, txid := range txids {
				swapInfo, err := b.CheckP2shTransaction(txid, true)
				if !tokens.ShouldRegisterSwapForError(err) {
					continue
				}
				b.processP2shSwapin(txid, swapInfo.Bind, isServer)
			}
			scannedBlocks.cacheScannedBlock(blockHash, h)
			log.Info("[scanp2sh] scanned tx history", "blockHash", blockHash, "height", h)
			h++
		}
		if latest > confirmations {
			latestStable := latest - confirmations
			if height < latestStable {
				height = latestStable
				UpdateLatestScanHeight(height)
			}
		}
		time.Sleep(restIntervalInP2shScanJob)
	}
	return nil
}

func (b *BtcBridge) scanP2shInTransactionPool(isServer bool) error {
	log.Info("[scanp2sh] start scan tx pool loop")
	for {
		txids, err := b.GetPoolTxidList()
		if err != nil {
			log.Error("[scanp2sh] get pool tx list error", "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		for _, txid := range txids {
			swapInfo, err := b.CheckP2shTransaction(txid, true)
			if !tokens.ShouldRegisterSwapForError(err) {
				continue
			}
			b.processP2shSwapin(txid, swapInfo.Bind, isServer)
		}
		time.Sleep(restIntervalInScanJob)
	}
	return nil
}

func (b *BtcBridge) CheckP2shTransaction(txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	tx, err := b.GetTransactionByHash(txHash)
	if err != nil {
		log.Debug("BtcBridge::GetTransaction fail", "tx", txHash, "err", err)
		return nil, tokens.ErrTxNotFound
	}
	var bindAddress, p2shAddress string
	for _, output := range tx.Vout {
		switch *output.Scriptpubkey_type {
		case "p2sh":
			p2shAddress = *output.Scriptpubkey_address
			bindAddress, _ = mongodb.FindP2shBindAddress(p2shAddress)
			if bindAddress != "" {
				break
			}
		}
	}
	if bindAddress == "" {
		return nil, tokens.ErrTxWithWrongReceiver
	}
	return b.VerifyP2shTransaction(txHash, bindAddress, allowUnstable)
}

type cachedScannedBlockRecord struct {
	hash   string
	height uint64
}

type cachedScannedBlocks struct {
	nextIndex int
	capacity  int
	blocks    []cachedScannedBlockRecord
}

func newCachedScannedBlocks(capacity int) *cachedScannedBlocks {
	return &cachedScannedBlocks{
		nextIndex: 0,
		capacity:  capacity,
		blocks:    make([]cachedScannedBlockRecord, capacity),
	}
}

func (c *cachedScannedBlocks) cacheScannedBlock(hash string, height uint64) {
	c.blocks[c.nextIndex] = cachedScannedBlockRecord{
		hash:   hash,
		height: height,
	}
	c.nextIndex = (c.nextIndex + 1) % c.capacity
}

func (c *cachedScannedBlocks) isBlockScanned(blockHash string) bool {
	for _, block := range c.blocks {
		if block.hash == blockHash {
			return true
		}
	}
	return false
}
