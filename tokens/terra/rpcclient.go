package terra

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var (
	rpcTimeout = 60

	errEmptyTxHash = errors.New("empty tx hash")
)

func joinURLPath(url, path string) string {
	url = strings.TrimSuffix(url, "/")
	if !strings.HasPrefix(path, "/") {
		url += "/"
	}
	return url + path
}

// GetLatestBlock get latest block
func GetLatestBlock(url string) (height uint64, err error) {
	path := "/cosmos/base/tendermint/v1beta1/blocks/latest"
	var result GetBlockResult
	err = client.RPCGetWithTimeout(&result, joinURLPath(url, path), rpcTimeout)
	if err != nil {
		return 0, err
	}
	return common.GetUint64FromStr(result.Block.Header.Height)
}

// BroadcastTx broadcast tx
func BroadcastTx(url, txData string) (txHash string, err error) {
	path := "/cosmos/tx/v1beta1/txs"
	result, err := client.RPCRawPostWithTimeout(joinURLPath(url, path), txData, rpcTimeout)
	if err != nil {
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
func SimulateTx(url string, req *SimulateRequest) (resp *SimulateResponse, err error) {
	//path := "/cosmos/tx/v1beta1/simulate"
	return nil, tokens.ErrTodo
}

// GetAccount get account details
func GetAccount(url, address string) (result *QueryAccountResult, err error) {
	path := "/cosmos/auth/v1beta1/accounts/" + address
	err = client.RPCGetWithTimeout(&result, joinURLPath(url, path), rpcTimeout)
	return result, err
}
