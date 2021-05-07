package eth

import (
	"errors"
	"fmt"
	"math/big"

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

	sender, err := types.Sender(b.Signer, tx)
	if err != nil {
		return "", err
	}

	// recheck balance before sending tx
	needValue := tx.Value()
	gasPrice := tx.GasPrice()
	gasLimit := tx.Gas()
	gasFee := new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(gasLimit))
	needValue = new(big.Int).Add(needValue, gasFee)
	err = b.checkBalance("", sender.Hex(), needValue)
	if err != nil {
		return "", err
	}

	txHash = tx.Hash().String()
	err = b.SendSignedTransaction(tx)
	if err != nil {
		log.Info("SendTransaction failed", "hash", txHash, "err", err)
		return txHash, err
	}
	log.Info("SendTransaction success", "hash", txHash)
	//#log.Trace("SendTransaction success", "raw", tx.RawStr())
	return txHash, nil
}
