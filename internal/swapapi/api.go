package swapapi

import (
	"encoding/hex"
	"time"

	"github.com/anyswap/CrossChain-Bridge/dcrm"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc"
	"github.com/btcsuite/btcd/txscript"
	rpcjson "github.com/gorilla/rpc/v2/json2"
)

var (
	errNotBtcBridge      = newRPCError(-32096, "bridge is not btc")
	errTokenPairNotExist = newRPCError(-32095, "token pair not exist")
	errSwapCannotRetry   = newRPCError(-32094, "swap can not retry")
	errForbidOperation   = newRPCError(-32093, "forbid operation")
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
		PairIDs:    tokens.GetAllPairIDs(),
		Version:    params.VersionWithMeta,
	}, nil
}

// GetTokenPairInfo api
func GetTokenPairInfo(pairID string) (*tokens.TokenPairConfig, error) {
	pairCfg := tokens.GetTokenPairConfig(pairID)
	if pairCfg == nil {
		return nil, errTokenPairNotExist
	}
	return pairCfg, nil
}

// GetSwapStatistics api
func GetSwapStatistics(pairID string) (*SwapStatistics, error) {
	log.Debug("[api] receive GetSwapStatistics", "pairID", pairID)
	return mongodb.GetSwapStatistics(pairID)
}

// GetRawSwapin api
func GetRawSwapin(txid, pairID, bindAddr *string) (*Swap, error) {
	return mongodb.FindSwapin(*txid, *pairID, *bindAddr)
}

// GetRawSwapinResult api
func GetRawSwapinResult(txid, pairID, bindAddr *string) (*SwapResult, error) {
	return mongodb.FindSwapinResult(*txid, *pairID, *bindAddr)
}

// GetSwapin api
func GetSwapin(txid, pairID, bindAddr *string) (*SwapInfo, error) {
	txidstr := *txid
	pairIDStr := *pairID
	bindStr := *bindAddr
	result, err := mongodb.FindSwapinResult(txidstr, pairIDStr, bindStr)
	if err == nil {
		return ConvertMgoSwapResultToSwapInfo(result), nil
	}
	register, err := mongodb.FindSwapin(txidstr, pairIDStr, bindStr)
	if err == nil {
		return ConvertMgoSwapToSwapInfo(register), nil
	}
	return nil, mongodb.ErrSwapNotFound
}

// GetRawSwapout api
func GetRawSwapout(txid, pairID, bindAddr *string) (*Swap, error) {
	return mongodb.FindSwapout(*txid, *pairID, *bindAddr)
}

// GetRawSwapoutResult api
func GetRawSwapoutResult(txid, pairID, bindAddr *string) (*SwapResult, error) {
	return mongodb.FindSwapoutResult(*txid, *pairID, *bindAddr)
}

// GetSwapout api
func GetSwapout(txid, pairID, bindAddr *string) (*SwapInfo, error) {
	txidstr := *txid
	pairIDStr := *pairID
	bindStr := *bindAddr
	result, err := mongodb.FindSwapoutResult(txidstr, pairIDStr, bindStr)
	if err == nil {
		return ConvertMgoSwapResultToSwapInfo(result), nil
	}
	register, err := mongodb.FindSwapout(txidstr, pairIDStr, bindStr)
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
func Swapin(txid, pairID *string) (*PostResult, error) {
	log.Debug("[api] receive Swapin", "txid", *txid, "pairID", *pairID)
	txidstr := *txid
	pairIDStr := *pairID
	swapInfo, err := tokens.SrcBridge.VerifyTransaction(pairIDStr, txidstr, true)
	err = addSwapToDatabase(txidstr, tokens.SwapinTx, swapInfo, err)
	if err != nil {
		return nil, err
	}
	return &SuccessPostResult, nil
}

// RetrySwapin api
func RetrySwapin(txid, pairID *string) (*PostResult, error) {
	log.Debug("[api] retry Swapin", "txid", *txid, "pairID", *pairID)
	if _, ok := tokens.SrcBridge.(tokens.NonceSetter); !ok {
		return nil, errSwapCannotRetry
	}
	txidstr := *txid
	pairIDStr := *pairID
	swapInfo, err := tokens.SrcBridge.VerifyTransaction(pairIDStr, txidstr, true)
	if err != nil {
		return nil, newRPCError(-32099, "retry swapin failed! "+err.Error())
	}
	bindStr := swapInfo.Bind
	swap, _ := mongodb.FindSwapin(txidstr, pairIDStr, bindStr)
	if swap == nil {
		return nil, mongodb.ErrItemNotFound
	}
	if !swap.Status.CanRetry() {
		return nil, errSwapCannotRetry
	}
	err = mongodb.UpdateSwapinStatus(txidstr, pairIDStr, bindStr, mongodb.TxNotStable, time.Now().Unix(), "")
	if err != nil {
		return nil, err
	}
	return &SuccessPostResult, nil
}

// Swapout api
func Swapout(txid, pairID *string) (*PostResult, error) {
	log.Debug("[api] receive Swapout", "txid", *txid, "pairID", *pairID)
	txidstr := *txid
	pairIDStr := *pairID
	swapInfo, err := tokens.DstBridge.VerifyTransaction(pairIDStr, txidstr, true)
	err = addSwapToDatabase(txidstr, tokens.SwapoutTx, swapInfo, err)
	if err != nil {
		return nil, err
	}
	return &SuccessPostResult, nil
}

func addSwapToDatabase(txid string, txType tokens.SwapTxType, swapInfo *tokens.TxSwapInfo, verifyError error) (err error) {
	if !tokens.ShouldRegisterSwapForError(verifyError) {
		return newRPCError(-32099, "verify swap failed! "+verifyError.Error())
	}
	var memo string
	if verifyError != nil {
		memo = verifyError.Error()
	}
	swap := &mongodb.MgoSwap{
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
		err = mongodb.AddSwapin(swap)
	} else {
		err = mongodb.AddSwapout(swap)
	}
	return err
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
	pairID := btc.PairID
	if swap, _ := mongodb.FindSwapin(txidstr, pairID, *bindAddr); swap != nil {
		return nil, mongodb.ErrItemIsDup
	}
	swapInfo, err := btc.BridgeInstance.VerifyP2shTransaction(pairID, txidstr, *bindAddr, true)
	if !tokens.ShouldRegisterSwapForError(err) {
		return nil, newRPCError(-32099, "verify p2sh swapin failed! "+err.Error())
	}
	var memo string
	if err != nil {
		memo = err.Error()
	}
	swap := &mongodb.MgoSwap{
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
func RegisterAddress(address, pairID string) (result *RegisteredAddress, err error) {
	pairCfg := tokens.GetTokenPairConfig(pairID)
	if pairCfg == nil {
		return nil, errTokenPairNotExist
	}
	if btc.BridgeInstance != nil {
		// use RegisterP2shAddress instead
		return nil, errForbidOperation
	}

	var rootPubkey string
	if pairCfg.UseBip32 {
		rootPubkey = pairCfg.SrcToken.DcrmPubkey
	}

	result, err = mongodb.FindRegisteredAddress(rootPubkey, address)
	if err == nil && result != nil {
		return result, nil
	}

	if pairCfg.UseBip32 {
		err = registerBip32Address(rootPubkey, address)
	} else {
		err = mongodb.AddRegisteredAddress("", address, "")
	}
	if err != nil {
		return nil, err
	}

	return mongodb.FindRegisteredAddress(rootPubkey, address)
}

func registerBip32Address(rootPubkey, address string) error {
	inputCode, err := tokens.SrcBridge.GetBip32InputCode(address)
	if err != nil {
		return err
	}
	childPubkey, err := dcrm.GetBip32ChildKey(rootPubkey, inputCode)
	if err != nil {
		return err
	}
	childAddress, err := tokens.SrcBridge.PublicKeyToAddress(childPubkey)
	if err != nil {
		return err
	}
	return mongodb.AddRegisteredAddress(rootPubkey, address, childAddress)
}

// GetRegisteredAddress get registered address info
func GetRegisteredAddress(address, pairID string) (*RegisteredAddress, error) {
	pairCfg := tokens.GetTokenPairConfig(pairID)
	if pairCfg == nil {
		return nil, errTokenPairNotExist
	}
	var rootPubkey string
	if pairCfg.UseBip32 {
		rootPubkey = pairCfg.SrcToken.DcrmPubkey
	}
	return mongodb.FindRegisteredAddress(rootPubkey, address)
}

// GetBip32AddressInfo get bip32 address info
func GetBip32AddressInfo(bip32Addr, pairID string) (*RegisteredAddress, error) {
	pairCfg := tokens.GetTokenPairConfig(pairID)
	if pairCfg == nil {
		return nil, errTokenPairNotExist
	}
	rootPubkey := pairCfg.SrcToken.DcrmPubkey
	return mongodb.FindBip32AddressInfo(rootPubkey, bip32Addr)
}
