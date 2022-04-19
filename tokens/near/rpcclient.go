package near

import (
	"strings"

	"github.com/anyswap/CrossChain-Bridge/common"
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
func GetLatestBlock(url string) (*Block, error) {
	return nil, nil
}

// GetLatestBlockNumber get latest block height
func GetLatestBlockNumber(url string) (height uint64, err error) {
	block, err := GetLatestBlock(url)
	if err != nil {
		return 0, err
	}
	return common.GetUint64FromStr(block.Header.Height)
}

// BroadcastTx broadcast tx
func BroadcastTx(url, txData string) (txHash string, err error) {
	return "", nil
}

// SimulateTx simulate tx
func SimulateTx(url string, req *SimulateRequest) (result *SimulateResponse, err error) {
	return nil, nil
}

// GetBaseAccount get account details
func GetBaseAccount(url, address string) (*BaseAccount, error) {
	return nil, nil
}

// GetTransactionByHash get tx by hash
func GetTransactionByHash(url, txHash string) (*GetTxResult, error) {
	return nil, nil
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
func GetContractInfo(url, contract string) (*QueryContractInfoResponse, error) {
	return nil, nil
}

// QueryContractStore query contract store
// `queryMsg` is json formed message with base64
func QueryContractStore(url, contract, queryMsg string) (interface{}, error) {
	return nil, nil
}
