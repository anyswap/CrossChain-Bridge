package terra

import (
	"fmt"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/tendermint/tendermint/crypto/tmhash"
)

// SendTransaction send signed tx
func (b *Bridge) SendTransaction(signedTx interface{}) (txHash string, err error) {
	txBytes, ok := signedTx.([]byte)
	if !ok {
		log.Printf("signed tx type %T is not []byte", signedTx)
		return "", fmt.Errorf("wrong signed transaction type")
	}
	req := &BroadcastTxRequest{
		TxBytes: string(txBytes),
		Mode:    "BROADCAST_MODE_SYNC",
	}
	txHash, err = b.BroadcastTx(req)
	if err != nil {
		log.Error("SendTransaction failed", "hash", txHash, "err", err)
	} else {
		log.Info("SendTransaction success", "hash", txHash)

		calcHash := fmt.Sprintf("%X", tmhash.Sum(txBytes))
		if !strings.EqualFold(txHash, calcHash) {
			logFunc := log.GetPrintFuncOr(params.IsDebugMode, log.Warn, log.Trace)
			logFunc("SendTransaction hash mismatch", "have", txHash, "calced", calcHash)
		}
	}
	if params.IsDebugMode() {
		log.Infof("SendTransaction rawtx is %x", txBytes)
	}
	return txHash, err
}
