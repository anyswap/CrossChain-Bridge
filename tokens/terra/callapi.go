package terra

import (
	"errors"

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
	var result GetBlockResponse
	err := client.RPCGetWithTimeout(&result, apiAddress, rpcTimeout)
	if err != nil {
		return 0, err
	}
	return uint64(result.Block.Header.Height), nil
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
	var result GetBlockResponse
	path := "/cosmos/base/tendermint/v1beta1/blocks/latest"
	for _, url := range urls {
		err := client.RPCGetWithTimeout(&result, url+path, rpcTimeout)
		if err == nil {
			height := uint64(result.Block.Header.Height)
			if height > maxHeight {
				maxHeight = height
			}
		}
	}
	if maxHeight > 0 {
		return maxHeight, nil
	}
	return 0, wrapRPCQueryError(err, path)
}
