package eth

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/common/hexutil"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/tools/rlp"
	"github.com/anyswap/CrossChain-Bridge/types"
)

// GetLatestBlockNumber call eth_blockNumber
func (b *Bridge) GetLatestBlockNumber() (uint64, error) {
	gateway := b.GatewayConfig
	url := gateway.APIAddress
	var result string
	err := client.RPCPost(&result, url, "eth_blockNumber")
	if err != nil {
		return 0, err
	}
	return common.GetUint64FromStr(result)
}

// GetBlockByHash call eth_getBlockByHash
func (b *Bridge) GetBlockByHash(blockHash string) (*types.RPCBlock, error) {
	gateway := b.GatewayConfig
	url := gateway.APIAddress
	var result *types.RPCBlock
	err := client.RPCPost(&result, url, "eth_getBlockByHash", blockHash, false)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, errors.New("block not found")
	}
	return result, nil
}

// GetBlockByNumber call eth_getBlockByNumber
func (b *Bridge) GetBlockByNumber(number *big.Int) (*types.RPCBlock, error) {
	gateway := b.GatewayConfig
	url := gateway.APIAddress
	var result *types.RPCBlock
	err := client.RPCPost(&result, url, "eth_getBlockByNumber", types.ToBlockNumArg(number), false)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, errors.New("block not found")
	}
	return result, nil
}

// GetTransactionByHash call eth_getTransactionByHash
func (b *Bridge) GetTransactionByHash(txHash string) (*types.RPCTransaction, error) {
	gateway := b.GatewayConfig
	url := gateway.APIAddress
	var result *types.RPCTransaction
	err := client.RPCPost(&result, url, "eth_getTransactionByHash", txHash)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, errors.New("tx not found")
	}
	return result, nil
}

// GetPendingTransactions call eth_pendingTransactions
func (b *Bridge) GetPendingTransactions() ([]*types.RPCTransaction, error) {
	gateway := b.GatewayConfig
	url := gateway.APIAddress
	var result []*types.RPCTransaction
	err := client.RPCPost(&result, url, "eth_pendingTransactions")
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetTransactionReceipt call eth_getTransactionReceipt
func (b *Bridge) GetTransactionReceipt(txHash string) (*types.RPCTxReceipt, error) {
	gateway := b.GatewayConfig
	url := gateway.APIAddress
	var result *types.RPCTxReceipt
	err := client.RPCPost(&result, url, "eth_getTransactionReceipt", txHash)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, errors.New("tx receipt not found")
	}
	return result, nil
}

// GetContractLogs get contract logs
func (b *Bridge) GetContractLogs(contractAddress, logTopic string, blockHeight uint64) ([]*types.RPCLog, error) {
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

// GetLogs call eth_getLogs
func (b *Bridge) GetLogs(filterQuery *types.FilterQuery) ([]*types.RPCLog, error) {
	gateway := b.GatewayConfig
	url := gateway.APIAddress
	args, err := types.ToFilterArg(filterQuery)
	if err != nil {
		return nil, err
	}
	var result []*types.RPCLog
	err = client.RPCPost(&result, url, "eth_getLogs", args)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetPoolNonce call eth_getTransactionCount
func (b *Bridge) GetPoolNonce(address string) (uint64, error) {
	gateway := b.GatewayConfig
	url := gateway.APIAddress
	account := common.HexToAddress(address)
	var result hexutil.Uint64
	err := client.RPCPost(&result, url, "eth_getTransactionCount", account, "pending")
	return uint64(result), err
}

// SuggestPrice call eth_gasPrice
func (b *Bridge) SuggestPrice() (*big.Int, error) {
	gateway := b.GatewayConfig
	url := gateway.APIAddress
	var result hexutil.Big
	err := client.RPCPost(&result, url, "eth_gasPrice")
	if err != nil {
		return nil, err
	}
	return result.ToInt(), nil
}

// SendSignedTransaction call eth_sendRawTransaction
func (b *Bridge) SendSignedTransaction(tx *types.Transaction) error {
	data, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return err
	}
	gateway := b.GatewayConfig
	url := gateway.APIAddress
	var result interface{}
	return client.RPCPost(&result, url, "eth_sendRawTransaction", common.ToHex(data))
}

// ChainID call eth_chainId
// Notice: eth_chainId return 0x0 for mainnet which is wrong (use net_version instead)
func (b *Bridge) ChainID() (*big.Int, error) {
	gateway := b.GatewayConfig
	url := gateway.APIAddress
	var result hexutil.Big
	err := client.RPCPost(&result, url, "eth_chainId")
	if err != nil {
		return nil, err
	}
	return result.ToInt(), nil
}

// NetworkID call net_version
func (b *Bridge) NetworkID() (*big.Int, error) {
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

// GetCode call eth_getCode
func (b *Bridge) GetCode(contract string) ([]byte, error) {
	gateway := b.GatewayConfig
	url := gateway.APIAddress
	var result hexutil.Bytes
	err := client.RPCPost(&result, url, "eth_getCode", contract, "latest")
	return []byte(result), err
}

// CallContract call eth_call
func (b *Bridge) CallContract(contract string, data hexutil.Bytes, blockNumber string) (string, error) {
	gateway := b.GatewayConfig
	url := gateway.APIAddress
	reqArgs := map[string]interface{}{
		"to":   contract,
		"data": data,
	}
	var result string
	err := client.RPCPost(&result, url, "eth_call", reqArgs, blockNumber)
	if err != nil {
		return "", err
	}
	return result, nil
}
