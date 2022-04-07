package terra

import (
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// BuildRawTransaction build raw tx
func (b *Bridge) BuildRawTransaction(args *tokens.BuildTxArgs) (rawTx interface{}, err error) {
	return nil, tokens.ErrTodo
}

// GetPoolNonce impl NonceSetter interface
func (b *Bridge) GetPoolNonce(address, height string) (uint64, error) {
	return 0, tokens.ErrTodo
}
