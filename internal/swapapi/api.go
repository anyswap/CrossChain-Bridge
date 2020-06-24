package swapapi

import (
	"encoding/hex"
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
	errSwapExist = newRPCError(-32097, "swap already exist")
	errNotBridge = newRPCError(-32096, "bridge is not btc")
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
		SrcToken:   config.SrcToken,
		DestToken:  config.DestToken,
		Version:    params.VersionWithMeta,
	}, nil
}

// GetSwapStatistics api
func GetSwapStatistics() (*SwapStatistics, error) {
	log.Debug("[api] receive GetSwapStatistics")
	return mongodb.GetSwapStatistics()
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
	if limit == 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}
	return limit
}

// GetSwapinHistory api
func GetSwapinHistory(address string, offset, limit int) ([]*SwapInfo, error) {
	log.Debug("[api] receive GetSwapinHistory", "address", address, "offset", offset, "limit", limit)
	limit = processHistoryLimit(limit)
	result, err := mongodb.FindSwapinResults(address, offset, limit)
	if err != nil {
		return nil, err
	}
	return ConvertMgoSwapResultsToSwapInfos(result), nil
}

// GetSwapoutHistory api
func GetSwapoutHistory(address string, offset, limit int) ([]*SwapInfo, error) {
	log.Debug("[api] receive GetSwapoutHistory", "address", address, "offset", offset, "limit", limit)
	limit = processHistoryLimit(limit)
	result, err := mongodb.FindSwapoutResults(address, offset, limit)
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
		TxID:      txid,
		TxType:    uint32(txType),
		Bind:      swapInfo.Bind,
		Status:    mongodb.TxNotStable,
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

// RecallSwapin api
func RecallSwapin(txid *string) (*PostResult, error) {
	log.Debug("[api] receive RecallSwapin", "txid", *txid)
	txidstr := *txid
	err := mongodb.RecallSwapin(txidstr)
	if err != nil {
		return nil, err
	}
	log.Info("[api] add recall swap", "txid", txidstr)
	return &SuccessPostResult, nil
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
		return nil, errNotBridge
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
		return nil, errNotBridge
	}
	txidstr := *txid
	if swap, _ := mongodb.FindSwapin(txidstr); swap != nil {
		return nil, errSwapExist
	}
	_, err := btc.BridgeInstance.VerifyP2shTransaction(txidstr, *bindAddr, true)
	if !tokens.ShouldRegisterSwapForError(err) {
		return nil, newRPCError(-32099, "verify p2sh swapin failed! "+err.Error())
	}
	var memo string
	if err != nil {
		memo = err.Error()
	}
	swap := &mongodb.MgoSwap{
		Key:       txidstr,
		TxID:      txidstr,
		TxType:    uint32(tokens.P2shSwapinTx),
		Bind:      *bindAddr,
		Status:    mongodb.TxNotStable,
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
