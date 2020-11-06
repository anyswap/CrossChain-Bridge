package btc

import (
	"fmt"
	"math/big"

	"github.com/anyswap/CrossChain-Bridge/tokens/btc/electrs"
)

// GetLatestBlockNumberOf impl
func (b *Bridge) GetLatestBlockNumberOf(apiAddress string) (uint64, error) {
	return electrs.GetLatestBlockNumberOf(apiAddress)
}

// GetLatestBlockNumber impl
func (b *Bridge) GetLatestBlockNumber() (uint64, error) {
	return electrs.GetLatestBlockNumber(b)
}

// GetTransactionByHash impl
func (b *Bridge) GetTransactionByHash(txHash string) (*electrs.ElectTx, error) {
	return electrs.GetTransactionByHash(b, txHash)
}

// GetElectTransactionStatus impl
func (b *Bridge) GetElectTransactionStatus(txHash string) (*electrs.ElectTxStatus, error) {
	return electrs.GetElectTransactionStatus(b, txHash)
}

// FindUtxos impl
func (b *Bridge) FindUtxos(addr string) ([]*electrs.ElectUtxo, error) {
	return electrs.FindUtxos(b, addr)
}

// GetPoolTxidList impl
func (b *Bridge) GetPoolTxidList() ([]string, error) {
	return electrs.GetPoolTxidList(b)
}

// GetPoolTransactions impl
func (b *Bridge) GetPoolTransactions(addr string) ([]*electrs.ElectTx, error) {
	return electrs.GetPoolTransactions(b, addr)
}

// GetTransactionHistory impl
func (b *Bridge) GetTransactionHistory(addr, lastSeenTxid string) ([]*electrs.ElectTx, error) {
	return electrs.GetTransactionHistory(b, addr, lastSeenTxid)
}

// GetOutspend impl
func (b *Bridge) GetOutspend(txHash string, vout uint32) (*electrs.ElectOutspend, error) {
	return electrs.GetOutspend(b, txHash, vout)
}

// PostTransaction impl
func (b *Bridge) PostTransaction(txHex string) (txHash string, err error) {
	return electrs.PostTransaction(b, txHex)
}

// GetBlockHash impl
func (b *Bridge) GetBlockHash(height uint64) (string, error) {
	return electrs.GetBlockHash(b, height)
}

// GetBlockTxids impl
func (b *Bridge) GetBlockTxids(blockHash string) ([]string, error) {
	return electrs.GetBlockTxids(b, blockHash)
}

// GetBlock impl
func (b *Bridge) GetBlock(blockHash string) (*electrs.ElectBlock, error) {
	return electrs.GetBlock(b, blockHash)
}

// GetBlockTransactions impl
func (b *Bridge) GetBlockTransactions(blockHash string, startIndex uint32) ([]*electrs.ElectTx, error) {
	return electrs.GetBlockTransactions(b, blockHash, startIndex)
}

// EstimateFeePerKb impl
func (b *Bridge) EstimateFeePerKb(blocks int) (int64, error) {
	return electrs.EstimateFeePerKb(b, blocks)
}

// GetBalance impl
func (b *Bridge) GetBalance(account string) (*big.Int, error) {
	utxos, err := b.FindUtxos(account)
	if err != nil {
		return nil, err
	}
	var balance uint64
	for _, utxo := range utxos {
		balance += *utxo.Value
	}
	return new(big.Int).SetUint64(balance), nil
}

// GetTokenBalance impl
func (b *Bridge) GetTokenBalance(tokenType, tokenAddress, accountAddress string) (*big.Int, error) {
	return nil, fmt.Errorf("[%v] can not get token balance of token with type '%v'", b.ChainConfig.BlockChain, tokenType)
}

// GetTokenSupply impl
func (b *Bridge) GetTokenSupply(tokenType, tokenAddress string) (*big.Int, error) {
	return nil, fmt.Errorf("[%v] can not get token supply of token with type '%v'", b.ChainConfig.BlockChain, tokenType)
}
