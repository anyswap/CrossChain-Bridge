package btc

import (
	. "github.com/fsn-dev/crossChain-Bridge/tokens"
)

func (b *BtcBridge) IsTransactionStable(txHash string) bool {
	return false
}

func (b *BtcBridge) VerifyTransaction(txHash string) (*TxSwapInfo, error) {
	return nil, ErrToDo
}
