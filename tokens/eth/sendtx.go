package eth

import (
	"errors"
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/types"
)

// SendTransaction send signed tx
func (b *Bridge) SendTransaction(signedTx interface{}) (txHash string, err error) {
	tx, ok := signedTx.(*types.Transaction)
	if !ok {
		fmt.Printf("signed tx is %+v\n", signedTx)
		return "", errors.New("wrong signed transaction type")
	}
	err = b.SendSignedTransaction(tx)
	if err != nil {
		log.Info("SendTransaction failed", "hash", tx.Hash().String(), "err", err)
		return "", err
	}
	log.Info("SendTransaction success", "hash", tx.Hash().String())
	return tx.Hash().String(), nil
}
