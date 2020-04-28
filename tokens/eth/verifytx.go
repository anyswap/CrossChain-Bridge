package eth

import (
	. "github.com/fsn-dev/crossChain-Bridge/tokens"
)

func (b *EthBridge) GetTransactionStatus(txHash string) *TxStatus {
	return nil
}

func (b *EthBridge) VerifyTransaction(txHash string) (*TxSwapInfo, error) {
	return nil, ErrTodo
}
