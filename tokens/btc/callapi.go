package btc

import (
	"github.com/anyswap/CrossChain-Bridge/tokens/btc/electrs"
)

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
