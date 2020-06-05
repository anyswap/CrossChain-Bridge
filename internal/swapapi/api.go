package swapapi

import (
	"encoding/hex"
	"time"

	"github.com/btcsuite/btcd/txscript"
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/mongodb"
	"github.com/fsn-dev/crossChain-Bridge/params"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	"github.com/fsn-dev/crossChain-Bridge/tokens/btc"
	rpcjson "github.com/gorilla/rpc/v2/json2"
)

var (
	errSwapExist    = newRpcError(-32097, "swap already exist")
	errNotBtcBridge = newRpcError(-32096, "bridge is not btc")
)

func newRpcError(ec rpcjson.ErrorCode, message string) error {
	return &rpcjson.Error{
		Code:    ec,
		Message: message,
	}
}

func newRpcInternalError(err error) error {
	return newRpcError(-32000, "rpcError: "+err.Error())
}

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

func GetSwapStatistics() (*SwapStatistics, error) {
	log.Debug("[api] receive GetSwapStatistics")
	return mongodb.GetSwapStatistics()
}

func GetRawSwapin(txid *string) (*Swap, error) {
	return mongodb.FindSwapin(*txid)
}

func GetRawSwapinResult(txid *string) (*SwapResult, error) {
	return mongodb.FindSwapinResult(*txid)
}

func GetSwapin(txid *string) (*SwapInfo, error) {
	//log.Debug("[api] receive GetSwapin", "txid", *txid)
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

func GetRawSwapout(txid *string) (*Swap, error) {
	return mongodb.FindSwapout(*txid)
}

func GetRawSwapoutResult(txid *string) (*SwapResult, error) {
	return mongodb.FindSwapoutResult(*txid)
}

func GetSwapout(txid *string) (*SwapInfo, error) {
	//log.Debug("[api] receive GetSwapout", "txid", *txid)
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

func GetSwapinHistory(address string, offset, limit int) ([]*SwapInfo, error) {
	log.Debug("[api] receive GetSwapinHistory", "address", address, "offset", offset, "limit", limit)
	limit = processHistoryLimit(limit)
	result, err := mongodb.FindSwapinResults(address, offset, limit)
	if err != nil {
		return nil, err
	}
	return ConvertMgoSwapResultsToSwapInfos(result), nil
}

func GetSwapoutHistory(address string, offset, limit int) ([]*SwapInfo, error) {
	log.Debug("[api] receive GetSwapoutHistory", "address", address, "offset", offset, "limit", limit)
	limit = processHistoryLimit(limit)
	result, err := mongodb.FindSwapoutResults(address, offset, limit)
	if err != nil {
		return nil, err
	}
	return ConvertMgoSwapResultsToSwapInfos(result), nil
}

func Swapin(txid *string) (*PostResult, error) {
	log.Debug("[api] receive Swapin", "txid", *txid)
	txidstr := *txid
	if swap, _ := mongodb.FindSwapin(txidstr); swap != nil {
		return nil, errSwapExist
	}
	swapInfo, err := tokens.SrcBridge.VerifyTransaction(txidstr, true)
	if !tokens.ShouldRegisterSwapForError(err) {
		return nil, newRpcError(-32099, "verify swapin failed! "+err.Error())
	}
	var memo string
	if err != nil {
		memo = err.Error()
	}
	swap := &mongodb.MgoSwap{
		Key:       txidstr,
		TxId:      txidstr,
		TxType:    mongodb.SwapinTx,
		Bind:      swapInfo.Bind,
		Status:    mongodb.TxNotStable,
		Timestamp: time.Now().Unix(),
		Memo:      memo,
	}
	err = mongodb.AddSwapin(swap)
	if err != nil {
		return nil, err
	}
	log.Info("[api] add swapin", "swap", swap)
	return &SuccessPostResult, nil
}

func Swapout(txid *string) (*PostResult, error) {
	log.Debug("[api] receive Swapout", "txid", *txid)
	txidstr := *txid
	if swap, _ := mongodb.FindSwapout(txidstr); swap != nil {
		return nil, errSwapExist
	}
	swapInfo, err := tokens.DstBridge.VerifyTransaction(txidstr, true)
	if !tokens.ShouldRegisterSwapForError(err) {
		return nil, newRpcError(-32098, "verify swapout failed! "+err.Error())
	}
	var memo string
	if err != nil {
		memo = err.Error()
	}
	swap := &mongodb.MgoSwap{
		Key:       txidstr,
		TxId:      txidstr,
		TxType:    mongodb.SwapoutTx,
		Bind:      swapInfo.Bind,
		Status:    mongodb.TxNotStable,
		Timestamp: time.Now().Unix(),
		Memo:      memo,
	}
	err = mongodb.AddSwapout(swap)
	if err != nil {
		return nil, err
	}
	log.Info("[api] add swapout", "swap", swap)
	return &SuccessPostResult, nil
}

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

func IsValidSwapinBindAddress(address *string) bool {
	return tokens.DstBridge.IsValidAddress(*address)
}

func IsValidSwapoutBindAddress(address *string) bool {
	return tokens.SrcBridge.IsValidAddress(*address)
}

func RegisterP2shAddress(bindAddress string) (*P2shAddressInfo, error) {
	return CalcP2shAddress(bindAddress, true)
}

func GetP2shAddressInfo(p2shAddress string) (*P2shAddressInfo, error) {
	bindAddress, err := mongodb.FindP2shBindAddress(p2shAddress)
	if err != nil {
		return nil, err
	}
	return CalcP2shAddress(bindAddress, false)
}

func CalcP2shAddress(bindAddress string, addToDatabase bool) (*P2shAddressInfo, error) {
	btcBridge, ok := tokens.SrcBridge.(*btc.BtcBridge)
	if !ok {
		return nil, errNotBtcBridge
	}
	p2shAddr, redeemScript, err := btcBridge.GetP2shAddress(bindAddress)
	if err != nil {
		return nil, newRpcInternalError(err)
	}
	disasm, err := txscript.DisasmString(redeemScript)
	if err != nil {
		return nil, newRpcInternalError(err)
	}
	if addToDatabase {
		result, _ := mongodb.FindP2shAddress(bindAddress)
		if result == nil {
			mongodb.AddP2shAddress(&mongodb.MgoP2shAddress{
				Key:         bindAddress,
				P2shAddress: p2shAddr,
			})
		}
	}
	return &P2shAddressInfo{
		BindAddress:        bindAddress,
		P2shAddress:        p2shAddr,
		RedeemScript:       hex.EncodeToString(redeemScript),
		RedeemScriptDisasm: disasm,
	}, nil
}

func P2shSwapin(txid *string, bindAddr *string) (*PostResult, error) {
	log.Debug("[api] receive P2shSwapin", "txid", *txid, "bindAddress", *bindAddr)
	btcBridge, ok := tokens.SrcBridge.(*btc.BtcBridge)
	if !ok {
		return nil, errNotBtcBridge
	}
	txidstr := *txid
	if swap, _ := mongodb.FindSwapin(txidstr); swap != nil {
		return nil, errSwapExist
	}
	_, err := btcBridge.VerifyP2shTransaction(txidstr, *bindAddr, true)
	if !tokens.ShouldRegisterSwapForError(err) {
		return nil, newRpcError(-32099, "verify p2sh swapin failed! "+err.Error())
	}
	var memo string
	if err != nil {
		memo = err.Error()
	}
	swap := &mongodb.MgoSwap{
		Key:       txidstr,
		TxId:      txidstr,
		TxType:    mongodb.P2shSwapinTx,
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
