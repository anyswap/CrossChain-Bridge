package cosmos

import (
	"errors"
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/log"
)

// SendTransaction send signed tx
func (b *Bridge) SendTransaction(signedTx interface{}) (txHash string, err error) {
	tx, ok := signedTx.(HashableStdTx)
	if !ok {
		fmt.Printf("signed tx is %+v\n", signedTx)
		return "", errors.New("wrong signed transaction type")
	}
	txHash = tx.Hash()
	err = b.BroadcastTx(tx.ToStdTx())
	if err != nil {
		log.Info("SendTransaction failed", "hash", txHash, "err", err)
		return txHash, err
	}
	return "", nil
}
