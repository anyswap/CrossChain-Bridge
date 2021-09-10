// Package tools provides tools for scanning and registering swaps.
package tools

import (
	"errors"
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
	swapRPCTimeout   = 60 // seconds
)

// IsSwapExist is swapin exist
func IsSwapExist(txid, pairID, bind string, isSwapin bool) bool {
	if mongodb.HasClient() {
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
		err := client.RPCPostWithTimeout(swapRPCTimeout, &result, params.ServerAPIAddress, method, args)
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
		if bind == "" { // must have non empty bind address
			return
		}
		if IsSwapExist(txid, pairID, bind, isSwapin) {
			return
		}
		isServer := dcrm.IsSwapServer()
		log.Info("[scan] register swap", "pairID", pairID, "isSwapin", isSwapin, "isServer", isServer, "tx", txid, "bind", bind)
		if isServer && mongodb.HasClient() {
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
				err := client.RPCPostWithTimeout(swapRPCTimeout, &result, params.ServerAPIAddress, method, args)
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
	return errors.Is(err, mongodb.ErrItemIsDup)
}

// RegisterP2shSwapin register p2sh swapin
func RegisterP2shSwapin(txid string, swapInfo *tokens.TxSwapInfo, verifyError error) {
	if !tokens.ShouldRegisterSwapForError(verifyError) {
		return
	}
	isServer := dcrm.IsSwapServer()
	bind := swapInfo.Bind
	log.Info("[scan] register p2sh swapin", "isServer", isServer, "tx", txid, "bind", bind)
	if isServer && mongodb.HasClient() {
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
			err := client.RPCPostWithTimeout(swapRPCTimeout, &result, params.ServerAPIAddress, "swap.P2shSwapin", args)
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
	if mongodb.HasClient() {
		bindAddress, _ = mongodb.FindP2shBindAddress(p2shAddress)
		return bindAddress
	}
	var result tokens.P2shAddressInfo
	for i := 0; i < retryRPCCount; i++ {
		err := client.RPCPostWithTimeout(swapRPCTimeout, &result, params.ServerAPIAddress, "swap.GetP2shAddressInfo", p2shAddress)
		if err == nil {
			return result.BindAddress
		}
		time.Sleep(retryRPCInterval)
	}
	return ""
}

// GetLatestScanHeight get latest scanned block height
func GetLatestScanHeight(isSrc bool) uint64 {
	if mongodb.HasClient() {
		for i := 0; i < 3; i++ {
			latestInfo, err := mongodb.FindLatestScanInfo(isSrc)
			if err == nil {
				height := latestInfo.BlockHeight
				log.Info("GetLatestScanHeight", "isSrc", isSrc, "height", height)
				return height
			}
			time.Sleep(1 * time.Second)
		}
		return 0
	}
	var result mongodb.MgoLatestScanInfo
	for i := 0; i < 3; i++ {
		err := client.RPCPostWithTimeout(swapRPCTimeout, &result, params.ServerAPIAddress, "swap.GetLatestScanInfo", isSrc)
		if err == nil {
			height := result.BlockHeight
			log.Info("GetLatestScanHeight", "isSrc", isSrc, "height", height)
			return height
		}
		time.Sleep(1 * time.Second)
	}
	return 0
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
	if dcrm.IsSwapServer() && mongodb.HasClient() {
		return mongodb.UpdateLatestScanInfo(isSrc, height)
	}
	return nil
}

// IsAddressRegistered is address registered
func IsAddressRegistered(address string) bool {
	if mongodb.HasClient() {
		result, _ := mongodb.FindRegisteredAddress(address)
		return result != nil
	}
	var result interface{}
	for i := 0; i < retryRPCCount; i++ {
		err := client.RPCPostWithTimeout(swapRPCTimeout, &result, params.ServerAPIAddress, "swap.GetRegisteredAddress", address)
		if err == nil {
			return result != nil
		}
		time.Sleep(retryRPCInterval)
	}
	return false
}

// AdjustGatewayOrder adjust gateway order by block height
func AdjustGatewayOrder(isSrc bool) {
	// use block number as weight
	var weightedAPIs WeightedStringSlice
	bridge := tokens.GetCrossChainBridge(isSrc)
	gateway := bridge.GetGatewayConfig()
	length := len(gateway.APIAddress)
	maxHeight := uint64(0)
	for i := length; i > 0; i-- { // query in reverse order
		apiAddress := gateway.APIAddress[i-1]
		height, _ := bridge.GetLatestBlockNumberOf(apiAddress)
		weightedAPIs = weightedAPIs.Add(apiAddress, height)
		if height > maxHeight {
			maxHeight = height
		}
	}
	tokens.CmpAndSetLatestBlockHeight(maxHeight, isSrc)
	weightedAPIs.Reverse() // reverse as iter in reverse order in the above
	weightedAPIs = weightedAPIs.Sort()
	gateway.APIAddress = weightedAPIs.GetStrings()
	if isSrc {
		log.Info("adjust source gateways", "result", weightedAPIs)
	} else {
		log.Info("adjust dest gateways", "result", weightedAPIs)
	}

	if !params.EnableCheckBlockFork() {
		return
	}

	if len(gateway.APIAddressExt) == 0 {
		return
	}

	forkChecker := tokens.GetForkChecker(isSrc)
	if forkChecker == nil {
		return
	}

	var checkPointHeight uint64
	stableHeight := tokens.GetStableConfirmations(isSrc)
	if maxHeight > stableHeight {
		checkPointHeight = maxHeight - stableHeight
	}

	retryCount := 3
	shouldPanic := false
	retrySleepInterval := 3 * time.Second
	time.Sleep(retrySleepInterval)
	for i := 1; i <= retryCount; i++ {
		hash1, err1 := forkChecker.GetBlockHashOf(gateway.APIAddress, checkPointHeight)
		hash2, err2 := forkChecker.GetBlockHashOf(gateway.APIAddressExt, checkPointHeight)
		if err1 != nil || err2 != nil {
			if i == retryCount {
				log.Warn("[detect] get block hash failed", "height", checkPointHeight, "isSrc", isSrc, "count", i, "err1", err1, "err2", err2)
			}
			time.Sleep(retryRPCInterval)
			continue
		}
		if hash1 == hash2 {
			log.Info("[detect] check block hash success", "height", checkPointHeight, "hash", hash1, "isSrc", isSrc, "count", i, "stable", stableHeight)
			return
		}
		failedContext := []interface{}{
			"height", checkPointHeight,
			"hash1", hash1, "hash2", hash2,
			"isSrc", isSrc, "count", i, "stable", stableHeight,
		}
		if i == retryCount {
			if shouldPanic {
				log.Fatal("[detect] check block hash failed", failedContext...)
			}
			// recheck of previous check point, and panic if still mismatch
			shouldPanic, i = true, -1
			if checkPointHeight > stableHeight {
				checkPointHeight -= stableHeight
			} else {
				checkPointHeight = 0
			}
		} else {
			log.Warn("[detect] check block hash failed", failedContext...)
		}
		time.Sleep(retrySleepInterval)
	}
}
