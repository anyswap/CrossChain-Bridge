package btc

import (
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

var (
	scannedTxs = tools.NewCachedScannedTxs(100)
)

// StartPoolTransactionScanJob scan job
func (b *Bridge) StartPoolTransactionScanJob() {
	log.Info("[scanpool] start scan pool tx job", "isSrc", b.IsSrc)
	for {
		txids, err := b.GetPoolTxidList()
		if err != nil {
			log.Error("[scanpool] get pool tx list error", "isSrc", b.IsSrc, "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		log.Info("[scanpool] scan pool tx", "isSrc", b.IsSrc, "txs", len(txids))
		for _, txid := range txids {
			if scannedTxs.IsTxScanned(txid) {
				continue
			}
			b.processTransaction(txid)
			scannedTxs.CacheScannedTx(txid)
		}
		time.Sleep(restIntervalInScanJob)
	}
}
