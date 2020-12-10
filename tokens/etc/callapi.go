package etc

import (
	"errors"

	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/types"
)

// GetTransactionAndReceipt get tx and receipt (fsn special)
func (b *Bridge) GetTransactionAndReceipt(txHash string) (*types.RPCTxAndReceipt, error) {
	gateway := b.GatewayConfig
	var result *types.RPCTxAndReceipt
	var err error
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress
		err = client.RPCPost(&result, url, "eth_getTransactionAndReceipt", txHash)
		if err == nil && result != nil {
			return result, nil
		}
	}
	if result == nil {
		return nil, errors.New("tx and receipt not found")
	}
	return nil, err
}
