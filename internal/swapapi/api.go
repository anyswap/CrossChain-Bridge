package swapapi

import (
	"encoding/hex"
	"strings"
	"sync"
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

	oraclesHeartbeats sync.Map // string -> int64 // key is enode
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
		Identifier:          config.Identifier,
		MustRegisterAccount: params.MustRegisterAccount(),
		SrcChain:            config.SrcChain,
		DestChain:           config.DestChain,
		PairIDs:             tokens.GetAllPairIDs(),
		Version:             params.VersionWithMeta,
	}, nil
}

// UpdateOracleHeartbeat api
func UpdateOracleHeartbeat(oracle string, timestamp int64) error {
	var exist bool
	for _, enode := range dcrm.GetAllEnodes() {
		if strings.EqualFold(oracle, enode) {
			if !strings.EqualFold(oracle, dcrm.GetSelfEnode()) {
				exist = true
			}
			break
		}
	}
	if !exist {
		return newRPCError(-32000, "wrong oracle info")
	}
	key := strings.ToLower(oracle)
	value, ok := oraclesHeartbeats.Load(key)
	if ok {
		oldTime := value.(int64)
		if timestamp > oldTime && timestamp < time.Now().Unix()+60 {
			oraclesHeartbeats.Store(key, timestamp)
		}
	} else {
		oraclesHeartbeats.Store(key, timestamp)
	}
	return nil
}

// GetOraclesHeartbeat api
func GetOraclesHeartbeat() map[string]string {
	result := make(map[string]string, 4)
	oraclesHeartbeats.Range(func(k, v interface{}) bool {
		enode := k.(string)
		startIndex := strings.Index(enode, "enode://")
		endIndex := strings.Index(enode, "@")
		if startIndex != -1 && endIndex != -1 {
			enodeID := enode[startIndex+8 : endIndex]
			timestamp := v.(int64)
			timeStr := time.Unix(timestamp, 0).Format(time.RFC3339)
			result[strings.ToLower(enodeID)] = timeStr
		}
		return true
	})
	return result
}

// GetTokenPairInfo api
func GetTokenPairInfo(pairID string) (*tokens.TokenPairConfig, error) {
	pairCfg := tokens.GetTokenPairConfig(pairID)
	if pairCfg == nil {
		return nil, errTokenPairNotExist
	}
	return pairCfg, nil
}

// GetTokenPairsInfo api
func GetTokenPairsInfo(pairIDs string) (map[string]*tokens.TokenPairConfig, error) {
	var pairIDSlice []string
	if strings.EqualFold(pairIDs, "all") {
		pairIDSlice = tokens.GetAllPairIDs()
	} else {
		pairIDSlice = strings.Split(pairIDs, ",")
	}
	result := make(map[string]*tokens.TokenPairConfig, len(pairIDSlice))
	for _, pairID := range pairIDSlice {
		result[pairID] = tokens.GetTokenPairConfig(pairID)
	}
	return result, nil
}

// GetNonceInfo api
func GetNonceInfo() (*SwapNonceInfo, error) {
	swapinNonces, swapoutNonces := mongodb.LoadAllSwapNonces()
	return &SwapNonceInfo{
		SwapinNonces:  swapinNonces,
		SwapoutNonces: swapoutNonces,
	}, nil
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
		limit = 20 // default
	case limit > 100:
		limit = 100
	case limit < -100:
		limit = -100
	}
	return limit
}

// GetSwapinHistory api
func GetSwapinHistory(address, pairID string, offset, limit int, status string) ([]*SwapInfo, error) {
	log.Debug("[api] receive GetSwapinHistory", "address", address, "pairID", pairID, "offset", offset, "limit", limit, "status", status)
	limit = processHistoryLimit(limit)
	result, err := mongodb.FindSwapinResults(address, pairID, offset, limit, status)
	if err != nil {
		return nil, err
	}
	return ConvertMgoSwapResultsToSwapInfos(result), nil
}

// GetSwapoutHistory api
func GetSwapoutHistory(address, pairID string, offset, limit int, status string) ([]*SwapInfo, error) {
	log.Debug("[api] receive GetSwapoutHistory", "address", address, "pairID", pairID, "offset", offset, "limit", limit)
	limit = processHistoryLimit(limit)
	result, err := mongodb.FindSwapoutResults(address, pairID, offset, limit, status)
	if err != nil {
		return nil, err
	}
	return ConvertMgoSwapResultsToSwapInfos(result), nil
}

// Swapin api
func Swapin(txid, pairID *string) (*PostResult, error) {
	log.Debug("[api] receive Swapin", "txid", *txid, "pairID", *pairID)
	return swap(txid, pairID, true)
}

// RetrySwapin api
func RetrySwapin(txid, pairID *string) (*PostResult, error) {
	log.Debug("[api] retry Swapin", "txid", *txid, "pairID", *pairID)
	if _, ok := tokens.SrcBridge.(tokens.NonceSetter); !ok {
		return nil, errSwapCannotRetry
	}
	txidstr := *txid
	pairIDStr := *pairID
	if err := basicCheckSwapRegister(tokens.SrcBridge, pairIDStr); err != nil {
		return nil, err
	}
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
	return swap(txid, pairID, false)
}

func basicCheckSwapRegister(bridge tokens.CrossChainBridge, pairIDStr string) error {
	tokenCfg := bridge.GetTokenConfig(pairIDStr)
	if tokenCfg == nil {
		return tokens.ErrUnknownPairID
	}
	if tokenCfg.DisableSwap {
		return tokens.ErrSwapIsClosed
	}
	return nil
}

func swap(txid, pairID *string, isSwapin bool) (*PostResult, error) {
	txidstr := *txid
	pairIDStr := *pairID
	bridge := tokens.GetCrossChainBridge(isSwapin)
	if err := basicCheckSwapRegister(bridge, pairIDStr); err != nil {
		return nil, err
	}
	swapInfo, err := bridge.VerifyTransaction(pairIDStr, txidstr, true)
	var txType tokens.SwapTxType
	if isSwapin {
		txType = tokens.SwapinTx
	} else {
		txType = tokens.SwapoutTx
	}
	err = addSwapToDatabase(txidstr, txType, swapInfo, err)
	if err != nil {
		return nil, err
	}
	if isSwapin {
		log.Info("[api] receive swapin register", "txid", txidstr, "pairID", pairIDStr)
	} else {
		log.Info("[api] receive swapout register", "txid", txidstr, "pairID", pairIDStr)
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
	if err := basicCheckSwapRegister(btc.BridgeInstance, pairID); err != nil {
		return nil, err
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

// RegisterAddress register address for ETH like chain
func RegisterAddress(address string) (*PostResult, error) {
	if !params.MustRegisterAccount() {
		return &SuccessPostResult, nil
	}
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
