// Package electrs get or post RPC queries to electrs server.
package electrs

import (
	"fmt"
	"sort"

	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// GetLatestBlockNumberOf call /blocks/tip/height
func GetLatestBlockNumberOf(apiAddress string) (uint64, error) {
	var result uint64
	url := apiAddress + "/blocks/tip/height"
	err := client.RPCGet(&result, url)
	if err == nil {
		return result, nil
	}
	return 0, err
}

// GetLatestBlockNumber call /blocks/tip/height
func GetLatestBlockNumber(b tokens.CrossChainBridge) (result uint64, err error) {
	gateway := b.GetGatewayConfig()
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress + "/blocks/tip/height"
		err = client.RPCGet(&result, url)
		if err == nil {
			return result, nil
		}
	}
	return 0, err
}

// GetTransactionByHash call /tx/{txHash}
func GetTransactionByHash(b tokens.CrossChainBridge, txHash string) (*ElectTx, error) {
	gateway := b.GetGatewayConfig()
	var result ElectTx
	var err error
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress + "/tx/" + txHash
		err = client.RPCGet(&result, url)
		if err == nil {
			return &result, nil
		}
	}
	return nil, err
}

// GetElectTransactionStatus call /tx/{txHash}/status
func GetElectTransactionStatus(b tokens.CrossChainBridge, txHash string) (*ElectTxStatus, error) {
	gateway := b.GetGatewayConfig()
	var result ElectTxStatus
	var err error
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress + "/tx/" + txHash + "/status"
		err = client.RPCGet(&result, url)
		if err == nil {
			return &result, nil
		}
	}
	return nil, err
}

// FindUtxos call /address/{add}/utxo (confirmed first, then big value first)
func FindUtxos(b tokens.CrossChainBridge, addr string) (result []*ElectUtxo, err error) {
	gateway := b.GetGatewayConfig()
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress + "/address/" + addr + "/utxo"
		err = client.RPCGet(&result, url)
		if err == nil {
			sort.Sort(SortableElectUtxoSlice(result))
			return result, nil
		}
	}
	return nil, err
}

// GetPoolTxidList call /mempool/txids
func GetPoolTxidList(b tokens.CrossChainBridge) (result []string, err error) {
	gateway := b.GetGatewayConfig()
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress + "/mempool/txids"
		err = client.RPCGet(&result, url)
		if err == nil {
			return result, nil
		}
	}
	return nil, err
}

// GetPoolTransactions call /address/{addr}/txs/mempool
func GetPoolTransactions(b tokens.CrossChainBridge, addr string) (result []*ElectTx, err error) {
	gateway := b.GetGatewayConfig()
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress + "/address/" + addr + "/txs/mempool"
		err = client.RPCGet(&result, url)
		if err == nil {
			return result, nil
		}
	}
	return nil, err
}

// GetTransactionHistory call /address/{addr}/txs/chain
func GetTransactionHistory(b tokens.CrossChainBridge, addr, lastSeenTxid string) (result []*ElectTx, err error) {
	gateway := b.GetGatewayConfig()
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress + "/address/" + addr + "/txs/chain"
		if lastSeenTxid != "" {
			url += "/" + lastSeenTxid
		}
		err = client.RPCGet(&result, url)
		if err == nil {
			return result, nil
		}
	}
	return nil, err
}

// GetOutspend call /tx/{txHash}/outspend/{vout}
func GetOutspend(b tokens.CrossChainBridge, txHash string, vout uint32) (*ElectOutspend, error) {
	gateway := b.GetGatewayConfig()
	var result ElectOutspend
	var err error
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress + "/tx/" + txHash + "/outspend/" + fmt.Sprintf("%d", vout)
		err = client.RPCGet(&result, url)
		if err == nil {
			return &result, nil
		}
	}
	return nil, err
}

// PostTransaction call post to /tx
func PostTransaction(b tokens.CrossChainBridge, txHex string) (txHash string, err error) {
	gateway := b.GetGatewayConfig()
	var success bool
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress + "/tx"
		hash0, err0 := client.RPCRawPost(url, txHex)
		if err0 == nil && !success {
			success = true
			txHash = hash0
		} else if err0 != nil {
			err = err0
		}
	}
	return txHash, err
}

// GetBlockHash call /block-height/{height}
func GetBlockHash(b tokens.CrossChainBridge, height uint64) (blockHash string, err error) {
	gateway := b.GetGatewayConfig()
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress + "/block-height/" + fmt.Sprintf("%d", height)
		blockHash, err = client.RPCRawGet(url)
		if err == nil {
			return blockHash, nil
		}
	}
	return "", err
}

// GetBlockTxids call /block/{blockHash}/txids
func GetBlockTxids(b tokens.CrossChainBridge, blockHash string) (result []string, err error) {
	gateway := b.GetGatewayConfig()
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress + "/block/" + blockHash + "/txids"
		err = client.RPCGet(&result, url)
		if err == nil {
			return result, nil
		}
	}
	return nil, err
}

// GetBlock call /block/{blockHash}
func GetBlock(b tokens.CrossChainBridge, blockHash string) (*ElectBlock, error) {
	gateway := b.GetGatewayConfig()
	var result ElectBlock
	var err error
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress + "/block/" + blockHash
		err = client.RPCGet(&result, url)
		if err == nil {
			return &result, nil
		}
	}
	return nil, err
}

// GetBlockTransactions call /block/{blockHash}/txs[/:start_index] (should start_index%25 == 0)
func GetBlockTransactions(b tokens.CrossChainBridge, blockHash string, startIndex uint32) (result []*ElectTx, err error) {
	gateway := b.GetGatewayConfig()
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress + "/block/" + blockHash + "/txs/" + fmt.Sprintf("%d", startIndex)
		err = client.RPCGet(&result, url)
		if err == nil {
			return result, nil
		}
	}
	return nil, err
}

// EstimateFeePerKb call /fee-estimates and multiply 1000
func EstimateFeePerKb(b tokens.CrossChainBridge, blocks int) (fee int64, err error) {
	var result map[int]float64
	gateway := b.GetGatewayConfig()
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress + "/fee-estimates"
		err = client.RPCGet(&result, url)
		if err == nil {
			break
		}
	}
	if err != nil {
		return 0, err
	}
	return int64(result[blocks] * 1000), nil
}
