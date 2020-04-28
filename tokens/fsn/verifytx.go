package fsn

import (
	. "github.com/fsn-dev/crossChain-Bridge/tokens"
)

func (b *FsnBridge) GetTransactionStatus(txHash string) *TxStatus {
	return nil
}

func (b *FsnBridge) VerifyTransaction(txHash string) (*TxSwapInfo, error) {
	return nil, ErrTodo
}
