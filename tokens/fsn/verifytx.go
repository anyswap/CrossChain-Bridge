package fsn

import (
	"github.com/fsn-dev/crossChain-Bridge/tokens"
)

func (b *FsnBridge) GetTransactionStatus(txHash string) *tokens.TxStatus {
	return nil
}

func (b *FsnBridge) VerifyTransaction(txHash string) (*tokens.TxSwapInfo, error) {
	return nil, tokens.ErrTodo
}
