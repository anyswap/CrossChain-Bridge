package eth

import (
	"errors"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/types"
)

// SendTransaction send signed tx
func (b *Bridge) SendTransaction(signedTx interface{}) (txHash string, err error) {
	if b.IsZKSync() {
		return b.SendZKSyncTransaction(signedTx)
	}
	tx, ok := signedTx.(*types.Transaction)
	if !ok {
		log.Printf("signed tx is %+v", signedTx)
		return "", errors.New("wrong signed transaction type")
	}
	txHash, err = b.SendSignedTransaction(tx)
	if err != nil {
		log.Info("SendTransaction failed", "hash", txHash, "err", err)
	} else {
		log.Info("SendTransaction success", "hash", txHash)
	}
	//#log.Trace("SendTransaction success", "raw", tx.RawStr())
	return txHash, err
}

func (b *Bridge) SendZKSyncTransaction(signedTx interface{}) (txHash string, err error) {
	tx, ok := signedTx.(*SignedZKSyncTx)
	if !ok {
		log.Printf("signed tx is %+v", signedTx)
		return "", errors.New("wrong signed transaction type")
	}
	txHash, err = b.SendSignedZKSyncTransaction(tx.Raw)
	chainID := b.SignerChainID
	if err != nil {
		log.Info("SendTransaction failed", "chainID", chainID, "hash", txHash, "err", err)
	} else {
		log.Info("SendTransaction success", "chainID", chainID, "hash", txHash)
	}
	return txHash, err
}
