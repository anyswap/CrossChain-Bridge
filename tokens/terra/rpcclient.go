package terra

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	rpcTimeout = 60

	zeroDec = sdk.Dec{}
	zeroInt = sdk.Int{}
)

func joinURLPath(url, path string) string {
	url = strings.TrimSuffix(url, "/")
	if !strings.HasPrefix(path, "/") {
		url += "/"
	}
	return url + path
}

// SetRPCTimeout set rpc timeout
func SetRPCTimeout(timeout int) {
	rpcTimeout = timeout
}

// GetLatestBlock get latest block
func GetLatestBlock(url string) (*Block, error) {
	path := "/blocks/latest"
	var result GetBlockResult
	err := client.RPCGetWithTimeout(&result, joinURLPath(url, path), rpcTimeout)
	if err != nil {
		return nil, err
	}
	return result.Block, err
}

// GetLatestBlockNumber get latest block height
func GetLatestBlockNumber(url string) (height uint64, err error) {
	block, err := GetLatestBlock(url)
	if err != nil {
		return 0, err
	}
	if block == nil {
		return 0, fmt.Errorf("wrong block result")
	}
	return common.GetUint64FromStr(block.Header.Height)
}

// BroadcastTx broadcast tx
func BroadcastTx(url, txData string) (txHash string, err error) {
	path := "/cosmos/tx/v1beta1/txs"
	// broadcast tx needs more rpc call time, here we double it
	result, err := client.RPCRawPostWithTimeout(joinURLPath(url, path), txData, 2*rpcTimeout)
	if err != nil {
		log.Trace("broadcast tx failed", "url", url, "path", path, "err", err)
		return "", err
	}

	var btResult BroadcastTxResult
	err = json.Unmarshal([]byte(result), &btResult)
	if err != nil {
		return "", err
	}

	return btResult.TxResponse.TxHash, nil
}

// SimulateTx simulate tx
func SimulateTx(url string, req *SimulateRequest) (result *SimulateResponse, err error) {
	path := "/cosmos/tx/v1beta1/simulate"
	err = client.RPCPostJSONRequestWithTimeout(joinURLPath(url, path), req, &result, rpcTimeout)
	return result, err
}

// GetBaseAccount get account details
func GetBaseAccount(url, address string) (*BaseAccount, error) {
	path := "/cosmos/auth/v1beta1/accounts/" + address
	var result GetBaseAccountResult
	err := client.RPCGetWithTimeout(&result, joinURLPath(url, path), rpcTimeout)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(result.Account.Address, address) {
		return nil, fmt.Errorf("get account address mismatch, have %v want %v", result.Account.Address, address)
	}
	return result.Account, nil
}

// GetTransactionByHash get tx by hash
func GetTransactionByHash(url, txHash string) (*GetTxResult, error) {
	path := "/cosmos/tx/v1beta1/txs/" + txHash
	var result GetTxResult
	err := client.RPCGetWithTimeout(&result, joinURLPath(url, path), rpcTimeout)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(result.TxResponse.TxHash, txHash) {
		return nil, fmt.Errorf("get tx hash mismatch, have %v want %v", result.TxResponse.TxHash, txHash)
	}
	return &result, nil
}

// GetBalance get balance by denom
func GetBalance(url, address, denom string) (sdk.Int, error) {
	path := fmt.Sprintf("/cosmos/bank/v1beta1/balances/%s/by_denom?denom=%s", address, denom)
	var result QueryBalanceResponse
	err := client.RPCGetWithTimeout(&result, joinURLPath(url, path), rpcTimeout)
	if err != nil {
		return zeroInt, err
	}
	if !strings.EqualFold(result.Balance.Denom, denom) {
		return zeroInt, fmt.Errorf("get balance denom mismatch, have %v want %v", result.Balance.Denom, denom)
	}
	amount, ok := sdk.NewIntFromString(result.Balance.Amount)
	if !ok {
		return zeroInt, fmt.Errorf("get balance amount parse to big.Int error,%v", result.Balance.Amount)
	}
	return amount, nil
}

// GetTaxCap get tax cap of a denom
func GetTaxCap(url, denom string) (sdk.Int, error) {
	path := "/terra/treasury/v1beta1/tax_caps/" + denom
	var result QueryTaxCapResuslt
	err := client.RPCGetWithTimeout(&result, joinURLPath(url, path), rpcTimeout)
	if err != nil {
		return zeroInt, err
	}
	taxCap, ok := sdk.NewIntFromString(result.TaxCap)
	if !ok {
		return zeroInt, fmt.Errorf("wrong tax cap %v of denom %v", result.TaxCap, denom)
	}
	return taxCap, nil
}

// GetTaxRate get current tax rate
func GetTaxRate(url string) (sdk.Dec, error) {
	path := "/terra/treasury/v1beta1/tax_rate"
	var result QueryTaxRateResuslt
	err := client.RPCGetWithTimeout(&result, joinURLPath(url, path), rpcTimeout)
	if err != nil {
		return zeroDec, err
	}
	return sdk.NewDecFromStr(result.TaxRate)
}

// GetGasPrice get gas price
func GetGasPrice(url, denom string) (sdk.Dec, error) {
	path := "/v1/txs/gas_prices"
	result := make(map[string]string)
	err := client.RPCGetWithTimeout(&result, joinURLPath(url, path), rpcTimeout)
	if err != nil {
		return zeroDec, err
	}
	val, exist := result[denom]
	if !exist {
		return zeroDec, fmt.Errorf("no gas price for denom '%v'", denom)
	}
	return sdk.NewDecFromStr(val)
}

// GetContractInfo get contract info
func GetContractInfo(url, contract string) (*QueryContractInfoResponse, error) {
	path := "/terra/wasm/v1beta1/contracts/" + contract
	var result QueryContractInfoResult
	err := client.RPCGetWithTimeout(&result, joinURLPath(url, path), rpcTimeout)
	if err != nil {
		return nil, err
	}
	return &result.ContractInfo, nil
}

// QueryContractStore query contract store
// `queryMsg` is json formed message with base64
func QueryContractStore(url, contract, queryMsg string) (interface{}, error) {
	path := fmt.Sprintf("/terra/wasm/v1beta1/contracts/%s/store?query_msg=%s", contract, queryMsg)
	var result QueryContractStoreResult
	err := client.RPCGetWithTimeout(&result, joinURLPath(url, path), rpcTimeout)
	if err != nil {
		return nil, err
	}
	return result.QueryResult, nil
}
