package tools

import (
	"time"

	"github.com/anyswap/CrossChain-Bridge/dcrm"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var (
	retryRPCCount    = 3
	retryRPCInterval = 1 * time.Second
)

// IsSwapinExist is swapin exist
func IsSwapinExist(txid string) bool {
	if mongodb.HasSession() {
		swap, _ := mongodb.FindSwapin(txid)
		return swap != nil
	}
	var result interface{}
	for i := 0; i < retryRPCCount; i++ {
		err := client.RPCPost(&result, params.ServerAPIAddress, "swap.GetSwapin", txid)
		if err == nil {
			return result != nil
		}
		time.Sleep(retryRPCInterval)
	}
	return false
}

// IsSwapoutExist is swapout exist
func IsSwapoutExist(txid string) bool {
	if mongodb.HasSession() {
		swap, _ := mongodb.FindSwapout(txid)
		return swap != nil
	}
	var result interface{}
	for i := 0; i < retryRPCCount; i++ {
		err := client.RPCPost(&result, params.ServerAPIAddress, "swap.GetSwapout", txid)
		if err == nil {
			return result != nil
		}
		time.Sleep(retryRPCInterval)
	}
	return false
}

// RegisterSwapin register swapin
func RegisterSwapin(txid, bind string, verifyError error) error {
	return registerSwap(true, txid, bind, verifyError)
}

// RegisterSwapout register swapout
func RegisterSwapout(txid, bind string, verifyError error) error {
	return registerSwap(false, txid, bind, verifyError)
}

func registerSwap(isSwapin bool, txid, bind string, verifyError error) error {
	if !tokens.ShouldRegisterSwapForError(verifyError) {
		return verifyError
	}
	isServer := dcrm.IsSwapServer()
	log.Info("[scan] register swap", "isSwapin", isSwapin, "isServer", isServer, "tx", txid, "bind", bind)
	if isServer {
		var memo string
		if verifyError != nil {
			memo = verifyError.Error()
		}
		swap := &mongodb.MgoSwap{
			Key:       txid,
			TxID:      txid,
			Bind:      bind,
			Status:    mongodb.GetStatusByTokenVerifyError(verifyError),
			Timestamp: time.Now().Unix(),
			Memo:      memo,
		}
		if isSwapin {
			swap.TxType = uint32(tokens.SwapinTx)
			return mongodb.AddSwapin(swap)
		}
		swap.TxType = uint32(tokens.SwapoutTx)
		return mongodb.AddSwapout(swap)
	}
	var result interface{}
	if isSwapin {
		return client.RPCPost(&result, params.ServerAPIAddress, "swap.Swapin", txid)
	}
	return client.RPCPost(&result, params.ServerAPIAddress, "swap.Swapout", txid)
}

// RegisterP2shSwapin register p2sh swapin
func RegisterP2shSwapin(txid, bind string, verifyError error) error {
	if !tokens.ShouldRegisterSwapForError(verifyError) {
		return verifyError
	}
	isServer := dcrm.IsSwapServer()
	log.Info("[scan] register p2sh swapin", "isServer", isServer, "tx", txid, "bind", bind)
	if isServer {
		var memo string
		if verifyError != nil {
			memo = verifyError.Error()
		}
		swap := &mongodb.MgoSwap{
			Key:       txid,
			TxID:      txid,
			TxType:    uint32(tokens.P2shSwapinTx),
			Bind:      bind,
			Status:    mongodb.GetStatusByTokenVerifyError(verifyError),
			Timestamp: time.Now().Unix(),
			Memo:      memo,
		}
		return mongodb.AddSwapin(swap)
	}
	args := map[string]interface{}{
		"txid": txid,
		"bind": bind,
	}
	var result interface{}
	return client.RPCPost(&result, params.ServerAPIAddress, "swap.P2shSwapin", args)
}

// GetP2shBindAddress get p2sh bind address
func GetP2shBindAddress(p2shAddress string) (bindAddress string) {
	if mongodb.HasSession() {
		bindAddress, _ = mongodb.FindP2shBindAddress(p2shAddress)
		return bindAddress
	}
	var result tokens.P2shAddressInfo
	for i := 0; i < retryRPCCount; i++ {
		err := client.RPCPost(&result, params.ServerAPIAddress, "swap.GetP2shAddressInfo", p2shAddress)
		if err == nil {
			return result.BindAddress
		}
		time.Sleep(retryRPCInterval)
	}
	return ""
}

// GetLatestScanHeight get latest scanned block height
func GetLatestScanHeight(isSrc bool) uint64 {
	if mongodb.HasSession() {
		for {
			latestInfo, err := mongodb.FindLatestScanInfo(isSrc)
			if err == nil {
				height := latestInfo.BlockHeight
				log.Info("GetLatestScanHeight", "isSrc", isSrc, "height", height)
				return height
			}
			time.Sleep(1 * time.Second)
		}
	}
	var result mongodb.MgoLatestScanInfo
	for {
		err := client.RPCPost(&result, params.ServerAPIAddress, "swap.GetLatestScanInfo", isSrc)
		if err == nil {
			height := result.BlockHeight
			log.Info("GetLatestScanHeight", "isSrc", isSrc, "height", height)
			return height
		}
		time.Sleep(1 * time.Second)
	}
}

// LoopGetLatestBlockNumber loop and get latest block number
func LoopGetLatestBlockNumber(b tokens.CrossChainBridge) uint64 {
	for {
		latest, err := b.GetLatestBlockNumber()
		if err != nil {
			log.Error("get latest block failed", "isSrc", b.IsSrcEndpoint(), "err", err)
			time.Sleep(3 * time.Second)
			continue
		}
		return latest
	}
}

// UpdateLatestScanInfo update latest scan info
func UpdateLatestScanInfo(isSrc bool, height uint64) error {
	if dcrm.IsSwapServer() {
		return mongodb.UpdateLatestScanInfo(isSrc, height)
	}
	return nil
}

// IsAddressRegistered is address registered
func IsAddressRegistered(address string) bool {
	if mongodb.HasSession() {
		result, _ := mongodb.FindRegisteredAddress(address)
		return result != nil
	}
	var result interface{}
	for i := 0; i < retryRPCCount; i++ {
		err := client.RPCPost(&result, params.ServerAPIAddress, "swap.GetRegisteredAddress", address)
		if err == nil {
			return result != nil
		}
		time.Sleep(retryRPCInterval)
	}
	return false
}
