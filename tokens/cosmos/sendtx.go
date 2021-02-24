package cosmos

import (
	"bytes"
	"encoding/hex"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/btcsuite/btcwallet/wallet/txauthor"
)

// SendTransaction send signed tx
func (b *Bridge) SendTransaction(signedTx interface{}) (txHash string, err error) {
	// TODO
	return "", nil
}
