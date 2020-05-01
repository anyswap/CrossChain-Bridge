package eth

import (
	"github.com/fsn-dev/crossChain-Bridge/tokens"
)

func (b *EthBridge) GetTransactionStatus(txHash string) *tokens.TxStatus {
	return nil
}

func (b *EthBridge) VerifyTransaction(txHash string) (*tokens.TxSwapInfo, error) {
	return nil, tokens.ErrTodo
}
