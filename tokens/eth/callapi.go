package eth

import (
	"errors"
	"math/big"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/common/hexutil"
	"github.com/fsn-dev/crossChain-Bridge/rpc/client"
	"github.com/fsn-dev/crossChain-Bridge/tools/rlp"
	"github.com/fsn-dev/crossChain-Bridge/types"
)

func (b *EthBridge) GetLatestBlockNumber() (uint64, error) {
	gateway := b.GatewayConfig
	url := gateway.ApiAddress
	var result string
	err := client.RpcPost(&result, url, "eth_blockNumber")
	if err != nil {
		return 0, err
	}
	return common.GetUint64FromStr(result)
}

func (b *EthBridge) GetBlockByHash(blockHash string) (*types.RPCBlock, error) {
	gateway := b.GatewayConfig
	url := gateway.ApiAddress
	var result *types.RPCBlock
	err := client.RpcPost(&result, url, "eth_getBlockByHash", blockHash, false)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, errors.New("block not found")
	}
	return result, nil
}

func (b *EthBridge) GetTransaction(txHash string) (*types.RPCTransaction, error) {
	gateway := b.GatewayConfig
	url := gateway.ApiAddress
	var result *types.RPCTransaction
	err := client.RpcPost(&result, url, "eth_getTransactionByHash", txHash)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, errors.New("tx not found")
	}
	return result, nil
}

func (b *EthBridge) GetTransactionReceipt(txHash string) (*types.RPCTxReceipt, error) {
	gateway := b.GatewayConfig
	url := gateway.ApiAddress
	var result *types.RPCTxReceipt
	err := client.RpcPost(&result, url, "eth_getTransactionReceipt", txHash)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, errors.New("tx receipt not found")
	}
	return result, nil
}

func (b *EthBridge) GetContractLogs(contractAddress, logTopic string, blockHeight uint64) ([]*types.RPCLog, error) {
	addresses := []common.Address{common.HexToAddress(contractAddress)}
	topics := []common.Hash{common.HexToHash(logTopic)}
	height := new(big.Int).SetUint64(blockHeight)

	filter := &types.FilterQuery{
		FromBlock: height,
		ToBlock:   height,
		Addresses: addresses,
		Topics:    [][]common.Hash{topics},
	}
	return b.GetLogs(filter)
}

func (b *EthBridge) GetLogs(filterQuery *types.FilterQuery) ([]*types.RPCLog, error) {
	gateway := b.GatewayConfig
	url := gateway.ApiAddress
	args, err := types.ToFilterArg(filterQuery)
	if err != nil {
		return nil, err
	}
	var result []*types.RPCLog
	err = client.RpcPost(&result, url, "eth_getLogs", args)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (b *EthBridge) GetPoolNonce(address string) (uint64, error) {
	gateway := b.GatewayConfig
	url := gateway.ApiAddress
	account := common.HexToAddress(address)
	var result hexutil.Uint64
	err := client.RpcPost(&result, url, "eth_getTransactionCount", account, "pending")
	return uint64(result), err
}

func (b *EthBridge) SuggestPrice() (*big.Int, error) {
	gateway := b.GatewayConfig
	url := gateway.ApiAddress
	var result hexutil.Big
	err := client.RpcPost(&result, url, "eth_gasPrice")
	if err != nil {
		return nil, err
	}
	return result.ToInt(), nil
}

func (b *EthBridge) SendSignedTransaction(tx *types.Transaction) error {
	data, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return err
	}
	gateway := b.GatewayConfig
	url := gateway.ApiAddress
	var result interface{}
	return client.RpcPost(&result, url, "eth_sendRawTransaction", common.ToHex(data))
}

func (b *EthBridge) ChainID() (*big.Int, error) {
	gateway := b.GatewayConfig
	url := gateway.ApiAddress
	var result hexutil.Big
	err := client.RpcPost(&result, url, "eth_chainId")
	if err != nil {
		return nil, err
	}
	return result.ToInt(), nil
}
