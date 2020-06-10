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
	"github.com/fsn-dev/crossChain-Bridge/params"
	"github.com/fsn-dev/crossChain-Bridge/rpc/client"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
)

var (
	p2shSwapinScanStarter    sync.Once
	p2shSwapServerAPIAddress string

	scannedBlocks      = newCachedScannedBlocks(10)
	scanStatusFileName = "btcscanstatus"
	scanStatusFile     *os.File

	restIntervalInP2shScanJob = 10 * time.Second
)

// StartP2shSwapinScanJob scan job
func (b *Bridge) StartP2shSwapinScanJob(isServer bool) {
	p2shSwapinScanStarter.Do(func() {
		if isServer {
			b.startP2shSwapinScanJobOnServer()
		} else {
			b.startP2shSwapinScanJobOnOracle()
		}
	})
}

func (b *Bridge) startP2shSwapinScanJobOnServer() {
	log.Info("[scanp2sh] server start scan p2sh swapin job")

	go b.scanP2shInTransactionPool(true)

	b.scanP2shTransactionHistory(true)
}

func (b *Bridge) startP2shSwapinScanJobOnOracle() {
	log.Info("[scanp2sh] oracle start scan p2sh swapin job")

	// init p2shSwapServerAPIAddress
	p2shSwapServerAPIAddress = getSwapServerAPIAddress()
	if p2shSwapServerAPIAddress == "" {
		log.Info("[scanp2sh] stop scan p2sh swapin job as no Oracle.ServerAPIAddress configed")
		return
	}

	go b.scanP2shInTransactionPool(false)

	b.scanP2shTransactionHistory(false)
}

func (b *Bridge) processP2shSwapin(txid string, isServer bool) error {
	if isServer {
		swap, _ := mongodb.FindSwapin(txid)
		if swap != nil {
			return nil
		}
	}
	swapInfo, err := b.CheckP2shTransaction(txid, isServer, true)
	if !tokens.ShouldRegisterSwapForError(err) {
		log.Trace("[scanp2sh] CheckP2shTransaction", "txid", txid, "isServer", isServer, "err", err)
		return err
	}
	if isServer {
		err = b.registerP2shSwapin(txid, swapInfo.Bind)
	} else if !b.isSwapinExistByQuery(txid) {
		err = b.postRegisterP2shSwapin(txid, swapInfo.Bind)
	}
	if err != nil {
		log.Trace("[scanp2sh] processP2shSwapin", "txid", txid, "isServer", isServer, "err", err)
	}
	return err
}

func (b *Bridge) registerP2shSwapin(txid string, bind string) error {
	log.Info("[scanp2sh] register p2sh swapin", "tx", txid, "bind", bind)
	swap := &mongodb.MgoSwap{
		Key:       txid,
		TxID:      txid,
		TxType:    uint32(tokens.P2shSwapinTx),
		Bind:      bind,
		Status:    mongodb.TxNotStable,
		Timestamp: time.Now().Unix(),
	}
	return mongodb.AddSwapin(swap)
}

func (b *Bridge) postRegisterP2shSwapin(txid string, bind string) error {
	log.Info("[scanp2sh] post register p2sh swapin", "tx", txid, "bind", bind)

	args := map[string]interface{}{
		"txid": txid,
		"bind": bind,
	}
	var result interface{}
	err := client.RPCPost(&result, p2shSwapServerAPIAddress, "swap.P2shSwapin", args)
	if err != nil {
		log.Debug("rpc call swap.P2shSwapin failed", "args", args, "err", err)
	}
	return err
}

func openBtcScanStatusFile() (err error) {
	if scanStatusFile != nil {
		return
	}
	scanStatusFilePath := common.AbsolutePath(params.DataDir, scanStatusFileName)
	scanStatusFile, err = os.OpenFile(scanStatusFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	log.Info("[scanp2sh] openBtcScanStatusFile succeed", "path", scanStatusFilePath)
	return nil
}

func getLatestScanHeight() uint64 {
	buf := make([]byte, 33)
	n, err := scanStatusFile.ReadAt(buf, 0)
	if err != nil {
		log.Error("read scanstatus file failed", "err", err)
		return 0
	}
	fileContent := strings.TrimSpace(string(buf[:n]))
	height, err := common.GetUint64FromStr(fileContent)
	if err != nil {
		log.Error("parse scanstatus file failed", "err", err)
		return 0
	}
	log.Info("[scanp2sh] getLatestScanHeight", "height", height)
	return height
}

func updateLatestScanHeight(height uint64) {
	fileContent := fmt.Sprintf("%d\n", height)
	retryCount := 3
	for i := 0; i < retryCount; i++ {
		_, err := scanStatusFile.Seek(0, 0)
		if err == nil {
			_, err = scanStatusFile.WriteString(fileContent)
			if err == nil {
				break
			}
		}
		time.Sleep(1 * time.Second)
	}
	_ = scanStatusFile.Sync()
	log.Info("[scanp2sh] updateLatestScanHeight", "height", height)
}

func (b *Bridge) getLatestBlock() uint64 {
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

func (b *Bridge) scanP2shTransactionHistory(isServer bool) {
	err := openBtcScanStatusFile()
	if err != nil {
		log.Error("openBtcScanStatusFile failed", "err", err)
		return
	}

	startHeight := getLatestScanHeight()
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
				_ = b.processP2shSwapin(txid, isServer)
			}
			scannedBlocks.cacheScannedBlock(blockHash, h)
			log.Info("[scanp2sh] scanned tx history", "blockHash", blockHash, "height", h, "txs", len(txids))
			h++
		}
		if latest > confirmations {
			latestStable := latest - confirmations
			if height < latestStable {
				height = latestStable
				updateLatestScanHeight(height)
			}
		}
		time.Sleep(restIntervalInP2shScanJob)
	}
}

func (b *Bridge) scanP2shInTransactionPool(isServer bool) {
	log.Info("[scanp2sh] start scan tx pool loop")
	for {
		txids, err := b.GetPoolTxidList()
		if err != nil {
			log.Error("[scanp2sh] get pool tx list error", "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		for _, txid := range txids {
			_ = b.processP2shSwapin(txid, isServer)
		}
		time.Sleep(restIntervalInScanJob)
	}
}

func getBindAddress(p2shAddress string, isServer bool) (bindAddress string) {
	if isServer {
		bindAddress, _ = mongodb.FindP2shBindAddress(p2shAddress)
	} else {
		var info tokens.P2shAddressInfo
		err := client.RPCPost(&info, p2shSwapServerAPIAddress, "swap.GetP2shAddressInfo", p2shAddress)
		if err == nil {
			bindAddress = info.BindAddress
		} else {
			log.Debug("rpc call swap.GetP2shAddressInfo failed", "p2shAddress", p2shAddress, "err", err)
		}
	}
	return bindAddress
}

// CheckP2shTransaction check p2sh tx
func (b *Bridge) CheckP2shTransaction(txHash string, isServer bool, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	tx, err := b.GetTransactionByHash(txHash)
	if err != nil {
		log.Debug("Bridge::GetTransaction fail", "tx", txHash, "err", err)
		return nil, tokens.ErrTxNotFound
	}
	var bindAddress, p2shAddress string
	for _, output := range tx.Vout {
		switch *output.ScriptpubkeyType {
		case "p2sh":
			p2shAddress = *output.ScriptpubkeyAddress
			bindAddress = getBindAddress(p2shAddress, isServer)
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
