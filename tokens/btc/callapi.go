package btc

import (
	. "github.com/fsn-dev/crossChain-Bridge/tokens/btc/electrs"
)

func (b *BtcBridge) GetLatestBlockNumber() (uint64, error) {
	return GetLatestBlockNumber(b)
}

func (b *BtcBridge) GetTransaction(txHash string) (*ElectTx, error) {
	return GetTransaction(b, txHash)
}

func (b *BtcBridge) GetElectTransactionStatus(txHash string) (*ElectTxStatus, error) {
	return GetElectTransactionStatus(b, txHash)
}

func (b *BtcBridge) FindUtxos(addr string) (*[]*ElectUtxo, error) {
	return FindUtxos(b, addr)
}
