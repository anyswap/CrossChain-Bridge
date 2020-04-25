package fsn

import (
	. "github.com/fsn-dev/crossChain-Bridge/tokens"
)

func (b *FsnBridge) IsTransactionStable(txHash string) bool {
	return false
}

func (b *FsnBridge) VerifyTransaction(txHash string) (*TxSwapInfo, error) {
	return nil, ErrTodo
}
