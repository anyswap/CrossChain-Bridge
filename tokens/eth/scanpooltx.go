package eth

import (
	"time"

	"github.com/fsn-dev/crossChain-Bridge/log"
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
			b.processTransaction(txid)
		}
		time.Sleep(restIntervalInScanJob)
	}
}
