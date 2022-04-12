package terra

import (
	"encoding/base64"
	"encoding/json"

	"github.com/anyswap/CrossChain-Bridge/tokens"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

// GetBalanceByDenom impl
func (b *Bridge) GetBalanceByDenom(address, denom string) (res sdk.Dec, err error) {
	urls := append(b.GatewayConfig.APIAddress, b.GatewayConfig.APIAddressExt...)
	for _, url := range urls {
		res, err = GetBalanceByDenom(url, address, denom)
		if err == nil {
			return res, nil
		}
	}
	return zeroDec, wrapRPCQueryError(err, "GetBalanceByDenom", denom)
}

// GetTaxRate impl
func (b *Bridge) GetTaxRate() (res sdk.Dec, err error) {
	urls := append(b.GatewayConfig.APIAddress, b.GatewayConfig.APIAddressExt...)
	for _, url := range urls {
		res, err = GetTaxRate(url)
		if err == nil {
			return res, nil
		}
	}
	return zeroDec, wrapRPCQueryError(err, "GetTaxRate")
}

// GetContractInfo impl
func (b *Bridge) GetContractInfo(contract string) (res *QueryContractInfoResponse, err error) {
	urls := append(b.GatewayConfig.APIAddress, b.GatewayConfig.APIAddressExt...)
	for _, url := range urls {
		res, err = GetContractInfo(url, contract)
		if err == nil {
			return res, nil
		}
	}
	return nil, wrapRPCQueryError(err, "GetContractInfo", contract)
}

func base64EncodedJSON(v interface{}) (string, error) {
	jsonData, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(jsonData), nil
}

// QueryContractStore impl
func (b *Bridge) QueryContractStore(contract string, query interface{}) (res interface{}, err error) {
	jsonData, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	queryMsg := base64.StdEncoding.EncodeToString(jsonData)
	urls := append(b.GatewayConfig.APIAddress, b.GatewayConfig.APIAddressExt...)
	for _, url := range urls {
		res, err = QueryContractStore(url, contract, queryMsg)
		if err == nil {
			return res, nil
		}
	}
	return nil, wrapRPCQueryError(err, "QueryContractStore", string(jsonData))
}
