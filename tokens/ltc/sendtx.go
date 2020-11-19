package ltc

import (
	"bytes"
	"encoding/hex"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/ltcsuite/ltcwallet/wallet/txauthor"
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

	return b.PostTransaction(txHex)
}
