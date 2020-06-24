package fsn

import (
	"errors"
	"fmt"
	"math/big"

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

// ChainID get chain id use net_version (eth_chainId does not work)
func (b *Bridge) ChainID() (*big.Int, error) {
	gateway := b.GatewayConfig
	url := gateway.APIAddress
	var result string
	err := client.RPCPost(&result, url, "net_version")
	if err != nil {
		return nil, err
	}
	version := new(big.Int)
	if _, ok := version.SetString(result, 10); !ok {
		return nil, fmt.Errorf("invalid net_version result %q", result)
	}
	return version, nil
}
