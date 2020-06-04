package electrs

import (
	"fmt"
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

func GetTransactionByHash(b tokens.CrossChainBridge, txHash string) (*ElectTx, error) {
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

func GetPoolTxidList(b tokens.CrossChainBridge) ([]string, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.ApiAddress + "/mempool/txids"
	var result []string
	err := client.RpcGet(&result, url)
	return result, err
}

func GetPoolTransactions(b tokens.CrossChainBridge, addr string) ([]*ElectTx, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.ApiAddress + "/address/" + addr + "/txs/mempool"
	var result []*ElectTx
	err := client.RpcGet(&result, url)
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

func GetOutspend(b tokens.CrossChainBridge, txHash string, vout uint32) (*ElectOutspend, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.ApiAddress + "/tx/" + txHash + "/outspend/" + fmt.Sprintf("%d", vout)
	var result ElectOutspend
	err := client.RpcGet(&result, url)
	return &result, err
}

func PostTransaction(b tokens.CrossChainBridge, txHex string) (txHash string, err error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.ApiAddress + "/tx"
	return client.RpcRawPost(url, txHex)
}

func GetBlockHash(b tokens.CrossChainBridge, height uint64) (string, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.ApiAddress + "/block-height/" + fmt.Sprintf("%d", height)
	return client.RpcRawGet(url)
}

func GetBlockTxids(b tokens.CrossChainBridge, blockHash string) ([]string, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.ApiAddress + "/block/" + blockHash + "/txids"
	var result []string
	err := client.RpcGet(&result, url)
	return result, err
}
