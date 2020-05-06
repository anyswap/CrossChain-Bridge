package eth

import (
	"errors"
	"fmt"

	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/types"
)

func (b *EthBridge) SendTransaction(signedTx interface{}) (txHash string, err error) {
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
