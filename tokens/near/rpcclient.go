package near

import (
	"fmt"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
)

var (
	rpcTimeout = 60
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
func GetLatestBlock(url string) (string, error) {
	request := &client.Request{}
	request.Method = "status"
	request.Params = []string{}
	request.ID = int(time.Now().UnixNano())
	request.Timeout = rpcTimeout
	var result NetworkStatus
	err := client.RPCPostRequest(url, request, &result)
	if err != nil {
		return "0", err
	}
	return result.syncInfo.latestBlockHeight, nil
}

func GetBlockByHash(url, hash string) (string, error) {
	request := &client.Request{}
	request.Method = "block"
	request.Params = map[string]string{"block_id": hash}
	request.ID = int(time.Now().UnixNano())
	request.Timeout = rpcTimeout
	var result BlockDetail
	err := client.RPCPostRequest(url, request, &result)
	if err != nil {
		return "0", err
	}
	return result.header.height, nil
}

// GetLatestBlockNumber get latest block height
func GetLatestBlockNumber(url string) (height uint64, err error) {
	block, err := GetLatestBlock(url)
	if err != nil {
		return 0, err
	}
	return common.GetUint64FromStr(block)
}

// BroadcastTx broadcast tx
func BroadcastTx(url, txData string) (txHash string, err error) {
	return "", nil
}

// // SimulateTx simulate tx
// func SimulateTx(url string, req *SimulateRequest) (result *SimulateResponse, err error) {
// 	return nil, nil
// }

// // GetBaseAccount get account details
// func GetBaseAccount(url, address string) (*BaseAccount, error) {
// 	return nil, nil
// }

// GetTransactionByHash get tx by hash
func GetTransactionByHash(url, txHash string) (*TransactionResult, error) {
	request := &client.Request{}
	request.Method = "tx"
	request.Params = []string{txHash, "userdemo.testnet"}
	request.ID = int(time.Now().UnixNano())
	request.Timeout = rpcTimeout
	var result TransactionResult
	err := client.RPCPostRequest(url, request, &result)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(result.Transaction.Hash, txHash) {
		return nil, fmt.Errorf("get tx hash mismatch, have %v want %v", result.Transaction.Hash, txHash)
	}
	return &result, nil
}

// GetBalance get balance by denom
func GetBalance(url, address, denom string) (uint64, error) {
	return 0, nil
}

// GetTaxCap get tax cap of a denom
func GetTaxCap(url, denom string) (uint64, error) {
	return 0, nil
}

// GetTaxRate get current tax rate
func GetTaxRate(url string) (uint64, error) {
	return 0, nil
}

// GetContractInfo get contract info
// func GetContractInfo(url, contract string) (*QueryContractInfoResponse, error) {
// 	return nil, nil
// }

// QueryContractStore query contract store
// `queryMsg` is json formed message with base64
func QueryContractStore(url, contract, queryMsg string) (interface{}, error) {
	return nil, nil
}
