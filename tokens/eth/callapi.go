package eth

import (
	"math/big"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/common/hexutil"
	"github.com/fsn-dev/crossChain-Bridge/rlp"
	"github.com/fsn-dev/crossChain-Bridge/rpc/client"
)

func (b *EthBridge) GetLatestBlockNumber() (uint64, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.ApiAddress
	var result string
	err := client.RpcPost(&result, url, "eth_blockNumber")
	if err != nil {
		return 0, err
	}
	return common.GetUint64FromStr(result)
}

func (b *EthBridge) GetBlockByHash(blockHash string) (*RPCBlock, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.ApiAddress
	var result RPCBlock
	err := client.RpcPost(&result, url, "eth_getBlockByHash", blockHash, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (b *EthBridge) GetTransaction(txHash string) (*RPCTransaction, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.ApiAddress
	var result RPCTransaction
	err := client.RpcPost(&result, url, "eth_getTransactionByHash", txHash)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (b *EthBridge) GetTransactionReceipt(txHash string) (*RPCTxReceipt, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.ApiAddress
	var result RPCTxReceipt
	err := client.RpcPost(&result, url, "eth_getTransactionReceipt", txHash)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (b *EthBridge) GetPoolNonce(address string) (uint64, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.ApiAddress
	account := common.HexToAddress(address)
	var result hexutil.Uint64
	err := client.RpcPost(&result, url, "eth_getTransactionCount", account, "pending")
	return uint64(result), err
}

func (b *EthBridge) SuggestPrice() (*big.Int, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.ApiAddress
	var result hexutil.Big
	err := client.RpcPost(&result, url, "eth_gasPrice")
	if err != nil {
		return nil, err
	}
	return result.ToInt(), nil
}

func (b *EthBridge) SendSignedTransaction(tx *Transaction) error {
	data, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return err
	}
	_, gateway := b.GetTokenAndGateway()
	url := gateway.ApiAddress
	var result interface{}
	return client.RpcPost(&result, url, "eth_sendRawTransaction", common.ToHex(data))
}

func (b *EthBridge) ChainID() (*big.Int, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.ApiAddress
	var result hexutil.Big
	err := client.RpcPost(&result, url, "eth_chainId")
	if err != nil {
		return nil, err
	}
	return result.ToInt(), nil
}
