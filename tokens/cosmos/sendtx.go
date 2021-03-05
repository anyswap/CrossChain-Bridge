package cosmos

import (
	"errors"

	"github.com/anyswap/CrossChain-Bridge/log"
)

// SendTransaction send signed tx
func (b *Bridge) SendTransaction(signedTx interface{}) (txHash string, err error) {
	tx, ok := signedTx.(HashableStdTx)
	if !ok {
		return "", errors.New("wrong signed transaction type")
	}
	txHash = tx.Hash()
	err = b.BroadcastTx(tx)
	if err != nil {
		log.Info("SendTransaction failed", "hash", txHash, "err", err)
		return txHash, err
	}
	return "", nil
}
