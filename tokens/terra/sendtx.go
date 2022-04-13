package terra

import (
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
)

// SendTransaction send signed tx
func (b *Bridge) SendTransaction(signedTx interface{}) (txHash string, err error) {
	txBytes, ok := signedTx.([]byte)
	if !ok {
		log.Printf("wrong signed tx type '%T'", signedTx)
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
	}
	if params.IsDebugMode() {
		log.Infof("SendTransaction rawtx is %v", string(txBytes))
	}
	return txHash, err
}
