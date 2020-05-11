package electrs

import (
	"sort"

	"github.com/fsn-dev/crossChain-Bridge/rpc/client"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
)

func GetLatestBlockNumber(b tokens.CrossChainBridge) (uint64, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.ApiAddress + "/blocks/tip/height"
	var result uint64
	err := client.RpcGet(&result, url)
	return result, err
}

func GetTransaction(b tokens.CrossChainBridge, txHash string) (*ElectTx, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.ApiAddress + "/tx/" + txHash
	var result ElectTx
	err := client.RpcGet(&result, url)
	return &result, err
}

func GetElectTransactionStatus(b tokens.CrossChainBridge, txHash string) (*ElectTxStatus, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.ApiAddress + "/tx/" + txHash + "/status"
	var result ElectTxStatus
	err := client.RpcGet(&result, url)
	return &result, err
}

func FindUtxos(b tokens.CrossChainBridge, addr string) ([]*ElectUtxo, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.ApiAddress + "/address/" + addr + "/utxo"
	var result []*ElectUtxo
	err := client.RpcGet(&result, url)
	sort.Sort(SortableElectUtxoSlice(result))
	return result, err
}

func GetTransactionHistory(b tokens.CrossChainBridge, addr string, lastSeenTxid string) ([]*ElectTx, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.ApiAddress + "/address/" + addr + "/txs/chain"
	if lastSeenTxid != "" {
		url = url + "/" + lastSeenTxid
	}
	var result []*ElectTx
	err := client.RpcGet(&result, url)
	return result, err
}

func PostTransaction(b tokens.CrossChainBridge, txHex string) (txHash string, err error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.ApiAddress + "/tx"
	return client.RpcRawPost(url, txHex)
}
