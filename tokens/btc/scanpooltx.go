package btc

import (
	"time"

	"github.com/fsn-dev/crossChain-Bridge/log"
)

func (b *Bridge) StartPoolTransactionScanJob() {
	log.Info("[scanpool] start scan pool tx job", "isSrc", b.IsSrc)
	for {
		txids, err := b.GetPoolTxidList()
		if err != nil {
			log.Error("[scanpool] get pool tx list error", "isSrc", b.IsSrc, "err", err)
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		for _, txid := range txids {
			b.processTransaction(txid)
		}
		time.Sleep(restIntervalInScanJob)
	}
}
