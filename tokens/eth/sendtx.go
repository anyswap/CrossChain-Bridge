package eth

import (
	"errors"
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/types"
)

// SendTransaction send signed tx
func (b *Bridge) SendTransaction(signedTx interface{}) (txHash string, err error) {
	tx, ok := signedTx.(*types.Transaction)
	if !ok {
		fmt.Printf("signed tx is %+v\n", signedTx)
		return "", errors.New("wrong signed transaction type")
	}
	txHash = tx.Hash().String()
	err = b.SendSignedTransaction(tx)
	if err != nil {
		log.Info("SendTransaction failed", "hash", txHash, "err", err)
		return txHash, err
	}
	log.Info("SendTransaction success", "hash", txHash)
	if params.IsDebugging() {
		log.Warn("SendTransaction success", "raw", tx.RawStr())
	}
	return txHash, nil
}
