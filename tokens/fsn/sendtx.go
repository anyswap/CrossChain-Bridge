package fsn

import (
	"errors"
)

func (b *FsnBridge) SendTransaction(signedTx interface{}) (txHash string, err error) {
	tx, ok := signedTx.(*Transaction)
	if !ok {
		return "", errors.New("wrong signed transaction type")
	}
	err = b.SendSignedTransaction(tx)
	if err != nil {
		return "", err
	}
	return tx.Hash().String(), nil
}
