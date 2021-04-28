package solana

import (
	"errors"

	"github.com/dfuse-io/solana-go"

	"github.com/anyswap/CrossChain-Bridge/log"
)

// SendTransaction send signed tx
func (b *Bridge) SendTransaction(signedTx interface{}) (txHash string, err error) {
	tx, ok := signedTx.(*solana.Transaction)
	if !ok {
		return "", errors.New("wrong signed transaction type")
	}
	txHash, err = b.BroadcastTx(tx)
	if err != nil {
		log.Info("SendTransaction failed", "hash", txHash, "err", err)
		return txHash, err
	}
	return txHash, nil
}
