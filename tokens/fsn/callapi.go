package fsn

import (
	"errors"

	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/types"
)

// GetTransactionAndReceipt get tx and receipt (fsn special)
func (b *Bridge) GetTransactionAndReceipt(txHash string) (*types.RPCTxAndReceipt, error) {
	gateway := b.GatewayConfig
	url := gateway.APIAddress
	var result *types.RPCTxAndReceipt
	err := client.RPCPost(&result, url, "fsn_getTransactionAndReceipt", txHash)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, errors.New("tx and receipt not found")
	}
	return result, nil
}
