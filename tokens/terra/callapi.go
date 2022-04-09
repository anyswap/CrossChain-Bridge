package terra

import (
	"encoding/json"

	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var (
	wrapRPCQueryError = tokens.WrapRPCQueryError
)

// GetLatestBlockNumberOf impl
func (b *Bridge) GetLatestBlockNumberOf(url string) (uint64, error) {
	return GetLatestBlock(url)
}

// GetLatestBlockNumber impl
func (b *Bridge) GetLatestBlockNumber() (height uint64, err error) {
	var tmp uint64
	urls := b.GatewayConfig.APIAddress
	for _, url := range urls {
		tmp, err = GetLatestBlock(url)
		if err == nil && tmp > height {
			height = tmp
		}
	}
	if height > 0 {
		tokens.CmpAndSetLatestBlockHeight(height, b.IsSrcEndpoint())
		return height, nil
	}
	return 0, wrapRPCQueryError(err, "GetLatestBlock")
}

// BroadcastTx impl
func (b *Bridge) BroadcastTx(req *BroadcastTxRequest) (txHash string, err error) {
	data, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	urls := append(b.GatewayConfig.APIAddress, b.GatewayConfig.APIAddressExt...)
	var res string
	for _, url := range urls {
		res, err = BroadcastTx(url, string(data))
		if txHash == "" && err == nil && res != "" {
			txHash = res
		}
	}

	if txHash != "" {
		return txHash, nil
	}
	return "", wrapRPCQueryError(err, "BroadcastTx")
}

// SimulateTx impl
func (b *Bridge) SimulateTx(req *SimulateRequest) (res *SimulateResponse, err error) {
	urls := append(b.GatewayConfig.APIAddress, b.GatewayConfig.APIAddressExt...)
	for _, url := range urls {
		res, err = SimulateTx(url, req)
		if err == nil && res != nil {
			return res, nil
		}
	}
	return nil, wrapRPCQueryError(err, "SimulateTx")
}
