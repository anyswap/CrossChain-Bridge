package eth

import (
	. "github.com/fsn-dev/crossChain-Bridge/tokens"
)

func (b *EthBridge) IsTransactionStable(txHash string) bool {
	return false
}

func (b *EthBridge) VerifyTransaction(txHash string) (*TxSwapInfo, error) {
	return nil, ErrTodo
}
