package colx

import (
	"bytes"
	"encoding/hex"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/giangnamnabka/btcwallet/wallet/txauthor"
)

// SendTransaction send signed tx
func (b *Bridge) SendTransaction(signedTx interface{}) (txHash string, err error) {
	authoredTx, ok := signedTx.(*txauthor.AuthoredTx)
	if !ok {
		return "", tokens.ErrWrongRawTx
	}

	tx := authoredTx.Tx
	if tx == nil {
		return "", tokens.ErrWrongRawTx
	}

	buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
	err = tx.Serialize(buf)
	if err != nil {
		return "", err
	}
	txHex := hex.EncodeToString(buf.Bytes())
	log.Info("Bridge send tx", "hash", tx.TxHash())

	txHash, err = b.PostTransaction(txHex)
	if err == nil {
		for _, txin := range tx.TxIn {
			inputtxhash := txin.PreviousOutPoint.Hash.String()
			inputvout := int(txin.PreviousOutPoint.Index)
			cond := func() bool {
				status, getstatuserr := b.GetElectTransactionStatus(txHash)
				if getstatuserr == nil && *status.Confirmed {
					return true
				}
				return false
			}
			_ = b.SetUnlockUtxoCond(inputtxhash, inputvout, cond)
		}
	}
	return txHash, err
}
