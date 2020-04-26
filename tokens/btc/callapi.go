package btc

import (
	. "github.com/fsn-dev/crossChain-Bridge/tokens/btc/electrs"
)

func (b *BtcBridge) GetLatestBlockNumber() (uint64, error) {
	return GetLatestBlockNumber(b)
}

func (b *BtcBridge) GetTransaction(txHash string) (*Tx, error) {
	return GetTransaction(b, txHash)
}

func (b *BtcBridge) GetTransactionStatus(txHash string) (*TxStatus, error) {
	return GetTransactionStatus(b, txHash)
}

func (b *BtcBridge) FindUtxos(addr string) (*[]*Utxo, error) {
	return FindUtxos(b, addr)
}
