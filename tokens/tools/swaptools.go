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

// IsSwapExist is swapin exist
func IsSwapExist(txid, pairID, bind string, isSwapin bool) bool {
	if mongodb.HasSession() {
		swap, _ := mongodb.FindSwap(isSwapin, txid, pairID, bind)
		return swap != nil
	}
	var result interface{}
	var method string
	if isSwapin {
		method = "swap.GetSwapin"
	} else {
		method = "swap.GetSwapout"
	}
	args := map[string]interface{}{
		"txid":   txid,
		"pairid": pairID,
		"bind":   bind,
	}
	for i := 0; i < retryRPCCount; i++ {
		err := client.RPCPost(&result, params.ServerAPIAddress, method, args)
		if err == nil {
			return result != nil
		}
		time.Sleep(retryRPCInterval)
	}
	return false
}

// RegisterSwapin register swapin
func RegisterSwapin(txid string, swapInfos []*tokens.TxSwapInfo, verifyErrors []error) {
	registerSwap(true, txid, swapInfos, verifyErrors)
}

// RegisterSwapout register swapout
func RegisterSwapout(txid string, swapInfos []*tokens.TxSwapInfo, verifyErrors []error) {
	registerSwap(false, txid, swapInfos, verifyErrors)
}

func registerSwap(isSwapin bool, txid string, swapInfos []*tokens.TxSwapInfo, verifyErrors []error) {
	if len(swapInfos) != len(verifyErrors) {
		log.Error("registerSwap with not equal number of swap infos and verify errors")
		return
	}
	for i, swapInfo := range swapInfos {
		verifyError := verifyErrors[i]
		if !tokens.ShouldRegisterSwapForError(verifyError) {
			continue
		}
		pairID := swapInfo.PairID
		bind := swapInfo.Bind
		if IsSwapExist(txid, pairID, bind, isSwapin) {
			return
		}
		isServer := dcrm.IsSwapServer()
		log.Info("[scan] register swap", "pairID", pairID, "isSwapin", isSwapin, "isServer", isServer, "tx", txid, "bind", bind)
		if isServer {
			var memo string
			if verifyError != nil {
				memo = verifyError.Error()
			}
			swap := &mongodb.MgoSwap{
				TxID:      txid,
				PairID:    pairID,
				TxTo:      swapInfo.TxTo,
				Bind:      bind,
				Status:    mongodb.GetStatusByTokenVerifyError(verifyError),
				Timestamp: time.Now().Unix(),
				Memo:      memo,
			}
			if isSwapin {
				swap.TxType = uint32(tokens.SwapinTx)
				_ = mongodb.AddSwapin(swap)
			} else {
				swap.TxType = uint32(tokens.SwapoutTx)
				_ = mongodb.AddSwapout(swap)
			}
		} else {
			var method string
			if isSwapin {
				method = "swap.Swapin"
			} else {
				method = "swap.Swapout"
			}
			args := map[string]interface{}{
				"txid":   txid,
				"pairid": pairID,
			}
			var result interface{}
			for i := 0; i < retryRPCCount; i++ {
				err := client.RPCPost(&result, params.ServerAPIAddress, method, args)
				if tokens.ShouldRegisterSwapForError(err) ||
					IsSwapAlreadyExistRegisterError(err) {
					break
				}
				time.Sleep(retryRPCInterval)
			}
		}
	}
}

// IsSwapAlreadyExistRegisterError is err of swap already exist
func IsSwapAlreadyExistRegisterError(err error) bool {
	return err == mongodb.ErrItemIsDup
}

// RegisterP2shSwapin register p2sh swapin
func RegisterP2shSwapin(txid string, swapInfo *tokens.TxSwapInfo, verifyError error) {
	if !tokens.ShouldRegisterSwapForError(verifyError) {
		return
	}
	isServer := dcrm.IsSwapServer()
	log.Info("[scan] register p2sh swapin", "isServer", isServer, "tx", txid)
	bind := swapInfo.Bind
	if isServer {
		var memo string
		if verifyError != nil {
			memo = verifyError.Error()
		}
		swap := &mongodb.MgoSwap{
			TxID:      txid,
			PairID:    swapInfo.PairID,
			TxTo:      swapInfo.TxTo,
			TxType:    uint32(tokens.P2shSwapinTx),
			Bind:      bind,
			Status:    mongodb.GetStatusByTokenVerifyError(verifyError),
			Timestamp: time.Now().Unix(),
			Memo:      memo,
		}
		_ = mongodb.AddSwapin(swap)
	} else {
		args := map[string]interface{}{
			"txid": txid,
			"bind": bind,
		}
		var result interface{}
		for i := 0; i < retryRPCCount; i++ {
			err := client.RPCPost(&result, params.ServerAPIAddress, "swap.P2shSwapin", args)
			if tokens.ShouldRegisterSwapForError(err) ||
				IsSwapAlreadyExistRegisterError(err) {
				break
			}
			time.Sleep(retryRPCInterval)
		}
	}
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
