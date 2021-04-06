package tron

import (
	"errors"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/types"
)

// SendTransaction send signed tx
func (b *Bridge) SendTransaction(signedTx interface{}) (txHash string, err error) {
	tx, ok := signedTx.(*core.Transaction)
	if !ok {
		fmt.Printf("signed tx is %+v\n", signedTx)
		return "", errors.New("wrong signed transaction type")
	}
	txHash = CalcTxHash(tx)
	err = b.BroadcastTx(tx)
	if err != nil {
		log.Info("SendTransaction failed", "hash", txHash, "err", err)
		return txHash, err
	}
	log.Info("SendTransaction success", "hash", txHash)
	//#log.Trace("SendTransaction success", "raw", tx.RawStr())
	return txHash, nil
}
