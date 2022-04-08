package terra

import (
	"encoding/json"
	"errors"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var (
	errEmptyURLs = errors.New("empty URLs")

	wrapRPCQueryError = tokens.WrapRPCQueryError

	rpcTimeout = 60
)

// GetLatestBlockNumberOf impl
func (b *Bridge) GetLatestBlockNumberOf(apiAddress string) (uint64, error) {
	var result GetBlockResult
	err := client.RPCGetWithTimeout(&result, apiAddress, rpcTimeout)
	if err != nil {
		return 0, err
	}
	return common.GetUint64FromStr(result.Block.Header.Height)
}

// GetLatestBlockNumber impl
func (b *Bridge) GetLatestBlockNumber() (uint64, error) {
	gateway := b.GatewayConfig
	maxHeight, err := getMaxLatestBlockNumber(gateway.APIAddress)
	if maxHeight > 0 {
		tokens.CmpAndSetLatestBlockHeight(maxHeight, b.IsSrcEndpoint())
		return maxHeight, nil
	}
	return 0, err
}

func getMaxLatestBlockNumber(urls []string) (maxHeight uint64, err error) {
	if len(urls) == 0 {
		return 0, errEmptyURLs
	}
	var result GetBlockResult
	path := "/cosmos/base/tendermint/v1beta1/blocks/latest"
	for _, url := range urls {
		err = client.RPCGetWithTimeout(&result, url+path, rpcTimeout)
		if err == nil {
			height, errt := common.GetUint64FromStr(result.Block.Header.Height)
			if errt == nil && height > maxHeight {
				maxHeight = height
			}
		}
	}
	if maxHeight > 0 {
		return maxHeight, nil
	}
	return 0, wrapRPCQueryError(err, "GetLatestBlock")
}

// BroadcastTx broadcast tx
func (b *Bridge) BroadcastTx(req *BroadcastTxRequest) (txHash string, err error) {
	data, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	var result string

	path := "/cosmos/tx/v1beta1/txs"
	urls := append(b.GatewayConfig.APIAddress, b.GatewayConfig.APIAddressExt...)
	for _, url := range urls {
		res, err := client.RPCRawPostWithTimeout(url+path, string(data), rpcTimeout)
		if result == "" && err == nil && res != "" {
			result = res
		}
	}

	if result == "" {
		return "", wrapRPCQueryError(err, "BroadcastTx")
	}

	var btResult BroadcastTxResult
	err = json.Unmarshal([]byte(result), &btResult)
	if err != nil {
		return "", err
	}

	txHash = btResult.TxResponse.TxHash
	if txHash != "" {
		return txHash, nil
	}
	return "", wrapRPCQueryError(err, "BroadcastTx")
}

func (b *Bridge) SimulateTx(req *SimulateRequest) (resp *SimulateResponse, err error) {
	return nil, tokens.ErrTodo
}
