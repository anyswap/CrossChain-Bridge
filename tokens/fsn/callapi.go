package fsn

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/fsn-dev/crossChain-Bridge/rpc/client"
	"github.com/fsn-dev/crossChain-Bridge/types"
)

func (b *FsnBridge) GetTransactionAndReceipt(txHash string) (*types.RPCTxAndReceipt, error) {
	gateway := b.GatewayConfig
	url := gateway.ApiAddress
	var result *types.RPCTxAndReceipt
	err := client.RpcPost(&result, url, "fsn_getTransactionAndReceipt", txHash)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, errors.New("tx and receipt not found")
	}
	return result, nil
}

func (b *FsnBridge) ChainID() (*big.Int, error) {
	gateway := b.GatewayConfig
	url := gateway.ApiAddress
	var result string
	err := client.RpcPost(&result, url, "net_version")
	if err != nil {
		return nil, err
	}
	version := new(big.Int)
	if _, ok := version.SetString(result, 10); !ok {
		return nil, fmt.Errorf("invalid net_version result %q", result)
	}
	return version, nil
}
