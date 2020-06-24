package electrs

import (
	"fmt"
	"sort"

	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// GetLatestBlockNumber call /blocks/tip/height
func GetLatestBlockNumber(b tokens.CrossChainBridge) (uint64, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.APIAddress + "/blocks/tip/height"
	var result uint64
	err := client.RPCGet(&result, url)
	return result, err
}

// GetTransactionByHash call /tx/{txHash}
func GetTransactionByHash(b tokens.CrossChainBridge, txHash string) (*ElectTx, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.APIAddress + "/tx/" + txHash
	var result ElectTx
	err := client.RPCGet(&result, url)
	return &result, err
}

// GetElectTransactionStatus call /tx/{txHash}/status
func GetElectTransactionStatus(b tokens.CrossChainBridge, txHash string) (*ElectTxStatus, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.APIAddress + "/tx/" + txHash + "/status"
	var result ElectTxStatus
	err := client.RPCGet(&result, url)
	return &result, err
}

// FindUtxos call /address/{add}/utxo (confirmed first, then big value first)
func FindUtxos(b tokens.CrossChainBridge, addr string) ([]*ElectUtxo, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.APIAddress + "/address/" + addr + "/utxo"
	var result []*ElectUtxo
	err := client.RPCGet(&result, url)
	sort.Sort(SortableElectUtxoSlice(result))
	return result, err
}

// GetPoolTxidList call /mempool/txids
func GetPoolTxidList(b tokens.CrossChainBridge) ([]string, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.APIAddress + "/mempool/txids"
	var result []string
	err := client.RPCGet(&result, url)
	return result, err
}

// GetPoolTransactions call /address/{addr}/txs/mempool
func GetPoolTransactions(b tokens.CrossChainBridge, addr string) ([]*ElectTx, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.APIAddress + "/address/" + addr + "/txs/mempool"
	var result []*ElectTx
	err := client.RPCGet(&result, url)
	return result, err
}

// GetTransactionHistory call /address/{addr}/txs/chain
func GetTransactionHistory(b tokens.CrossChainBridge, addr, lastSeenTxid string) ([]*ElectTx, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.APIAddress + "/address/" + addr + "/txs/chain"
	if lastSeenTxid != "" {
		url = url + "/" + lastSeenTxid
	}
	var result []*ElectTx
	err := client.RPCGet(&result, url)
	return result, err
}

// GetOutspend call /tx/{txHash}/outspend/{vout}
func GetOutspend(b tokens.CrossChainBridge, txHash string, vout uint32) (*ElectOutspend, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.APIAddress + "/tx/" + txHash + "/outspend/" + fmt.Sprintf("%d", vout)
	var result ElectOutspend
	err := client.RPCGet(&result, url)
	return &result, err
}

// PostTransaction call post to /tx
func PostTransaction(b tokens.CrossChainBridge, txHex string) (txHash string, err error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.APIAddress + "/tx"
	return client.RPCRawPost(url, txHex)
}

// GetBlockHash call /block-height/{height}
func GetBlockHash(b tokens.CrossChainBridge, height uint64) (string, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.APIAddress + "/block-height/" + fmt.Sprintf("%d", height)
	return client.RPCRawGet(url)
}

// GetBlockTxids call /block/{blockHash}/txids
func GetBlockTxids(b tokens.CrossChainBridge, blockHash string) ([]string, error) {
	_, gateway := b.GetTokenAndGateway()
	url := gateway.APIAddress + "/block/" + blockHash + "/txids"
	var result []string
	err := client.RPCGet(&result, url)
	return result, err
}
