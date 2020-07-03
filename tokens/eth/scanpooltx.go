package eth

import (
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

var (
	scannedTxs = tools.NewCachedScannedTxs(300)
)

// StartPoolTransactionScanJob scan job
func (b *Bridge) StartPoolTransactionScanJob() {
	log.Info("[scanpool] start scan tx pool loop", "isSrc", b.IsSrc)
	for {
		txs, err := b.GetPendingTransactions()
		if err != nil {
			log.Error("[scanpool] get pool txs error", "isSrc", b.IsSrc, "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		for _, tx := range txs {
			txid := tx.Hash.String()
			if scannedTxs.IsTxScanned(txid) {
				continue
			}
			log.Info("[scanpool] scanned tx", "isSrc", b.IsSrc, "txid", txid)
			b.processTransaction(txid)
			scannedTxs.CacheScannedTx(txid)
		}
		time.Sleep(restIntervalInScanJob)
	}
}
