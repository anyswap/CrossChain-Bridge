package btc

import (
	"github.com/fsn-dev/crossChain-Bridge/tokens/btc/electrs"
)

func (b *BtcBridge) GetLatestBlockNumber() (uint64, error) {
	return electrs.GetLatestBlockNumber(b)
}

func (b *BtcBridge) GetTransaction(txHash string) (*electrs.ElectTx, error) {
	return electrs.GetTransaction(b, txHash)
}

func (b *BtcBridge) GetElectTransactionStatus(txHash string) (*electrs.ElectTxStatus, error) {
	return electrs.GetElectTransactionStatus(b, txHash)
}

func (b *BtcBridge) FindUtxos(addr string) ([]*electrs.ElectUtxo, error) {
	return electrs.FindUtxos(b, addr)
}

func (b *BtcBridge) GetPoolTxidList() ([]string, error) {
	return electrs.GetPoolTxidList(b)
}

func (b *BtcBridge) GetPoolTransactions(addr string) ([]*electrs.ElectTx, error) {
	return electrs.GetPoolTransactions(b, addr)
}

func (b *BtcBridge) GetTransactionHistory(addr string, lastSeenTxid string) ([]*electrs.ElectTx, error) {
	return electrs.GetTransactionHistory(b, addr, lastSeenTxid)
}

func (b *BtcBridge) PostTransaction(txHex string) (txHash string, err error) {
	return electrs.PostTransaction(b, txHex)
}
