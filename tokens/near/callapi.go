package near

import (
	"encoding/base64"
	"encoding/json"

	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var (
	wrapRPCQueryError = tokens.WrapRPCQueryError
)

// GetLatestBlockNumberOf impl
func (b *Bridge) GetLatestBlockNumberOf(url string) (uint64, error) {
	return GetLatestBlockNumber(url)
}

// GetLatestBlockNumber impl
func (b *Bridge) GetLatestBlockNumber() (height uint64, err error) {
	var tmp uint64
	urls := b.GatewayConfig.APIAddress
	for _, url := range urls {
		tmp, err = GetLatestBlockNumber(url)
		if err == nil && tmp > height {
			height = tmp
		}
	}
	if height > 0 {
		tokens.CmpAndSetLatestBlockHeight(height, b.IsSrcEndpoint())
		return height, nil
	}
	return 0, wrapRPCQueryError(err, "GetLatestBlockNumber")
}

// GetLatestBlockNumber impl
func (b *Bridge) GetBlockByHash(hash string) (height string, err error) {
	var tmp string
	urls := b.GatewayConfig.APIAddress
	for _, url := range urls {
		tmp, err = GetBlockByHash(url, hash)
		if err == nil {
			return tmp, err
		}
	}
	return "0", wrapRPCQueryError(err, "GetBlockByHash")
}

// // BroadcastTx impl
// func (b *Bridge) BroadcastTx(req *BroadcastTxRequest) (txHash string, err error) {
// 	return "", nil
// }

// // SimulateTx impl
// func (b *Bridge) SimulateTx(req *SimulateRequest) (res *SimulateResponse, err error) {

// 	return nil, nil
// }

// GetBalance impl
func (b *Bridge) GetBalance(address, denom string) (res uint64, err error) {
	return 0, nil
}

// GetBaseAccount impl
// func (b *Bridge) GetBaseAccount(address string) (res *BaseAccount, err error) {
// 	return nil, nil
// }

// GetTaxCap impl
func (b *Bridge) GetTaxCap(denom string) (res uint64, err error) {
	return 0, nil
}

// GetTaxRate impl
func (b *Bridge) GetTaxRate() (res uint64, err error) {
	return 0, nil
}

// GetContractInfo impl
// func (b *Bridge) GetContractInfo(contract string) (res *QueryContractInfoResponse, err error) {
// 	return nil, nil
// }

func base64EncodedJSON(v interface{}) (string, error) {
	jsonData, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(jsonData), nil
}

// QueryContractStore impl
func (b *Bridge) QueryContractStore(contract string, query interface{}) (res interface{}, err error) {
	return nil, nil
}
