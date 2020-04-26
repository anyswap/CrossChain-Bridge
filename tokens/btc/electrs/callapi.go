package electrs

import (
	"github.com/fsn-dev/crossChain-Bridge/rpc/client"
	. "github.com/fsn-dev/crossChain-Bridge/tokens"
)

func GetLatestBlockNumber(b CrossChainBridge) (uint64, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.ApiAddress + "/blocks/tip/height"
	var result uint64
	err := client.RpcGet(&result, url)
	return result, err
}

func GetTransaction(b CrossChainBridge, txHash string) (*Tx, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.ApiAddress + "/tx/" + txHash
	var result Tx
	err := client.RpcGet(&result, url)
	return &result, err
}

func GetTransactionStatus(b CrossChainBridge, txHash string) (*TxStatus, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.ApiAddress + "/tx/" + txHash + "/status"
	var result TxStatus
	err := client.RpcGet(&result, url)
	return &result, err
}

func FindUtxos(b CrossChainBridge, addr string) (*[]*Utxo, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.ApiAddress + "/address/" + addr + "/utxo"
	var result []*Utxo
	err := client.RpcGet(&result, url)
	return &result, err
}
