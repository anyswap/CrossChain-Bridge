package swapapi

import (
	"encoding/hex"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc"
	"github.com/btcsuite/btcd/txscript"
	rpcjson "github.com/gorilla/rpc/v2/json2"
)

var (
	errSwapExist       = newRPCError(-32097, "swap already exist")
	errNotBtcBridge    = newRPCError(-32096, "bridge is not btc")
	errSwapNotExist    = newRPCError(-32095, "swap not exist")
	errSwapCannotRetry = newRPCError(-32094, "swap can not retry")
	errTokenPairExist  = newRPCError(-32093, "token pair not exist")
)

func newRPCError(ec rpcjson.ErrorCode, message string) error {
	return &rpcjson.Error{
		Code:    ec,
		Message: message,
	}
}

func newRPCInternalError(err error) error {
	return newRPCError(-32000, "rpcError: "+err.Error())
}

// GetServerInfo api
func GetServerInfo() (*ServerInfo, error) {
	log.Debug("[api] receive GetServerInfo")
	config := params.GetConfig()
	if config == nil {
		return nil, nil
	}
	return &ServerInfo{
		Identifier: config.Identifier,
		SrcChain:   config.SrcChain,
		DestChain:  config.DestChain,
		Version:    params.VersionWithMeta,
	}, nil
}

// GetTokenPairInfo api
func GetTokenPairInfo(pairID string) (*tokens.TokenPairConfig, error) {
	pairCfg := tokens.GetTokenPairConfig(pairID)
	if pairCfg == nil {
		return nil, errTokenPairExist
	}
	return pairCfg, nil
}

// GetSwapStatistics api
func GetSwapStatistics(pairID string) (*SwapStatistics, error) {
	log.Debug("[api] receive GetSwapStatistics", "pairID", pairID)
	return mongodb.GetSwapStatistics(pairID)
}

// GetRawSwapin api
func GetRawSwapin(txid *string) (*Swap, error) {
	return mongodb.FindSwapin(*txid)
}

// GetRawSwapinResult api
func GetRawSwapinResult(txid *string) (*SwapResult, error) {
	return mongodb.FindSwapinResult(*txid)
}

// GetSwapin api
func GetSwapin(txid *string) (*SwapInfo, error) {
	txidstr := *txid
	result, err := mongodb.FindSwapinResult(txidstr)
	if err == nil {
		return ConvertMgoSwapResultToSwapInfo(result), nil
	}
	register, err := mongodb.FindSwapin(txidstr)
	if err == nil {
		return ConvertMgoSwapToSwapInfo(register), nil
	}
	return nil, mongodb.ErrSwapNotFound
}

// GetRawSwapout api
func GetRawSwapout(txid *string) (*Swap, error) {
	return mongodb.FindSwapout(*txid)
}

// GetRawSwapoutResult api
func GetRawSwapoutResult(txid *string) (*SwapResult, error) {
	return mongodb.FindSwapoutResult(*txid)
}

// GetSwapout api
func GetSwapout(txid *string) (*SwapInfo, error) {
	txidstr := *txid
	result, err := mongodb.FindSwapoutResult(txidstr)
	if err == nil {
		return ConvertMgoSwapResultToSwapInfo(result), nil
	}
	register, err := mongodb.FindSwapout(txidstr)
	if err == nil {
		return ConvertMgoSwapToSwapInfo(register), nil
	}
	return nil, mongodb.ErrSwapNotFound
}

func processHistoryLimit(limit int) int {
	switch {
	case limit == 0:
		limit = 20
	case limit > 100:
		limit = 100
	case limit < 0:
		limit = 1
	}
	return limit
}

// GetSwapinHistory api
func GetSwapinHistory(address, pairID string, offset, limit int) ([]*SwapInfo, error) {
	log.Debug("[api] receive GetSwapinHistory", "address", address, "pairID", pairID, "offset", offset, "limit", limit)
	limit = processHistoryLimit(limit)
	result, err := mongodb.FindSwapinResults(address, pairID, offset, limit)
	if err != nil {
		return nil, err
	}
	return ConvertMgoSwapResultsToSwapInfos(result), nil
}

// GetSwapoutHistory api
func GetSwapoutHistory(address, pairID string, offset, limit int) ([]*SwapInfo, error) {
	log.Debug("[api] receive GetSwapoutHistory", "address", address, "pairID", pairID, "offset", offset, "limit", limit)
	limit = processHistoryLimit(limit)
	result, err := mongodb.FindSwapoutResults(address, pairID, offset, limit)
	if err != nil {
		return nil, err
	}
	return ConvertMgoSwapResultsToSwapInfos(result), nil
}

// Swapin api
func Swapin(txid *string) (*PostResult, error) {
	log.Debug("[api] receive Swapin", "txid", *txid)
	txidstr := *txid
	if swap, _ := mongodb.FindSwapin(txidstr); swap != nil {
		return nil, errSwapExist
	}
	swapInfo, err := tokens.SrcBridge.VerifyTransaction(txidstr, true)
	if !tokens.ShouldRegisterSwapForError(err) {
		return nil, newRPCError(-32099, "verify swapin failed! "+err.Error())
	}
	err = addSwapToDatabase(txidstr, tokens.SwapinTx, swapInfo, err)
	if err != nil {
		return nil, err
	}
	return &SuccessPostResult, nil
}

// RetrySwapin api
func RetrySwapin(txid *string) (*PostResult, error) {
	log.Debug("[api] retry Swapin", "txid", *txid)
	txidstr := *txid
	swap, _ := mongodb.FindSwapin(txidstr)
	if swap == nil {
		return nil, errSwapNotExist
	}
	if !swap.Status.CanRetry() {
		return nil, errSwapCannotRetry
	}
	_, err := tokens.SrcBridge.VerifyTransaction(txidstr, true)
	if err != nil {
		return nil, newRPCError(-32099, "retry swapin failed! "+err.Error())
	}
	err = mongodb.UpdateSwapinStatus(txidstr, mongodb.TxNotStable, time.Now().Unix(), "")
	if err != nil {
		return nil, err
	}
	return &SuccessPostResult, nil
}

// Swapout api
func Swapout(txid *string) (*PostResult, error) {
	log.Debug("[api] receive Swapout", "txid", *txid)
	txidstr := *txid
	if swap, _ := mongodb.FindSwapout(txidstr); swap != nil {
		return nil, errSwapExist
	}
	swapInfo, err := tokens.DstBridge.VerifyTransaction(txidstr, true)
	if !tokens.ShouldRegisterSwapForError(err) {
		return nil, newRPCError(-32098, "verify swapout failed! "+err.Error())
	}
	err = addSwapToDatabase(txidstr, tokens.SwapoutTx, swapInfo, err)
	if err != nil {
		return nil, err
	}
	return &SuccessPostResult, nil
}

func addSwapToDatabase(txid string, txType tokens.SwapTxType, swapInfo *tokens.TxSwapInfo, verifyError error) error {
	var memo string
	if verifyError != nil {
		memo = verifyError.Error()
	}
	swap := &mongodb.MgoSwap{
		Key:       txid,
		PairID:    swapInfo.PairID,
		TxID:      txid,
		TxTo:      swapInfo.TxTo,
		TxType:    uint32(txType),
		Bind:      swapInfo.Bind,
		Status:    mongodb.GetStatusByTokenVerifyError(verifyError),
		Timestamp: time.Now().Unix(),
		Memo:      memo,
	}
	isSwapin := txType == tokens.SwapinTx
	log.Info("[api] add swap", "isSwapin", isSwapin, "swap", swap)
	if isSwapin {
		return mongodb.AddSwapin(swap)
	}
	return mongodb.AddSwapout(swap)
}

// IsValidSwapinBindAddress api
func IsValidSwapinBindAddress(address *string) bool {
	return tokens.DstBridge.IsValidAddress(*address)
}

// IsValidSwapoutBindAddress api
func IsValidSwapoutBindAddress(address *string) bool {
	return tokens.SrcBridge.IsValidAddress(*address)
}

// RegisterP2shAddress api
func RegisterP2shAddress(bindAddress string) (*tokens.P2shAddressInfo, error) {
	return calcP2shAddress(bindAddress, true)
}

// GetP2shAddressInfo api
func GetP2shAddressInfo(p2shAddress string) (*tokens.P2shAddressInfo, error) {
	bindAddress, err := mongodb.FindP2shBindAddress(p2shAddress)
	if err != nil {
		return nil, err
	}
	return calcP2shAddress(bindAddress, false)
}

func calcP2shAddress(bindAddress string, addToDatabase bool) (*tokens.P2shAddressInfo, error) {
	if btc.BridgeInstance == nil {
		return nil, errNotBtcBridge
	}
	p2shAddr, redeemScript, err := btc.BridgeInstance.GetP2shAddress(bindAddress)
	if err != nil {
		return nil, newRPCInternalError(err)
	}
	disasm, err := txscript.DisasmString(redeemScript)
	if err != nil {
		return nil, newRPCInternalError(err)
	}
	if addToDatabase {
		result, _ := mongodb.FindP2shAddress(bindAddress)
		if result == nil {
			_ = mongodb.AddP2shAddress(&mongodb.MgoP2shAddress{
				Key:         bindAddress,
				P2shAddress: p2shAddr,
			})
		}
	}
	return &tokens.P2shAddressInfo{
		BindAddress:        bindAddress,
		P2shAddress:        p2shAddr,
		RedeemScript:       hex.EncodeToString(redeemScript),
		RedeemScriptDisasm: disasm,
	}, nil
}

// P2shSwapin api
func P2shSwapin(txid, bindAddr *string) (*PostResult, error) {
	log.Debug("[api] receive P2shSwapin", "txid", *txid, "bindAddress", *bindAddr)
	if btc.BridgeInstance == nil {
		return nil, errNotBtcBridge
	}
	txidstr := *txid
	if swap, _ := mongodb.FindSwapin(txidstr); swap != nil {
		return nil, errSwapExist
	}
	swapInfo, err := btc.BridgeInstance.VerifyP2shTransaction(txidstr, *bindAddr, true)
	if !tokens.ShouldRegisterSwapForError(err) {
		return nil, newRPCError(-32099, "verify p2sh swapin failed! "+err.Error())
	}
	var memo string
	if err != nil {
		memo = err.Error()
	}
	swap := &mongodb.MgoSwap{
		Key:       txidstr,
		PairID:    swapInfo.PairID,
		TxID:      txidstr,
		TxTo:      swapInfo.TxTo,
		TxType:    uint32(tokens.P2shSwapinTx),
		Bind:      *bindAddr,
		Status:    mongodb.GetStatusByTokenVerifyError(err),
		Timestamp: time.Now().Unix(),
		Memo:      memo,
	}
	err = mongodb.AddSwapin(swap)
	if err != nil {
		return nil, err
	}
	log.Info("[api] add p2sh swapin", "swap", swap)
	return &SuccessPostResult, nil
}

// GetLatestScanInfo api
func GetLatestScanInfo(isSrc bool) (*LatestScanInfo, error) {
	return mongodb.FindLatestScanInfo(isSrc)
}

// RegisterAddress register address
func RegisterAddress(address string) (*PostResult, error) {
	address = strings.ToLower(address)
	err := mongodb.AddRegisteredAddress(address)
	if err != nil {
		return nil, err
	}
	log.Info("[api] register address", "address", address)
	return &SuccessPostResult, nil
}

// GetRegisteredAddress get registered address
func GetRegisteredAddress(address string) (*RegisteredAddress, error) {
	address = strings.ToLower(address)
	return mongodb.FindRegisteredAddress(address)
}
