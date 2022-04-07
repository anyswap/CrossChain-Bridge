package terra

import (
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// SendTransaction send signed tx
func (b *Bridge) SendTransaction(signedTx interface{}) (txHash string, err error) {
	return "", tokens.ErrTodo
}
