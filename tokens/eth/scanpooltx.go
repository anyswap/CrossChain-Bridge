package eth

import (
	"fmt"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

var (
	scannedTxs = tools.NewCachedScannedTxs(300)
)

// StartPoolTransactionScanJob scan job
func (b *Bridge) StartPoolTransactionScanJob() {
	chainName := b.ChainConfig.BlockChain
	log.Infof("[scanpool] start scan %v tx pool job", chainName)
	errorSubject := fmt.Sprintf("[scanpool] get %v pool txs error", chainName)
	scanSubject := fmt.Sprintf("[scanpool] scanned %v tx", chainName)
	for {
		txs, err := b.GetPendingTransactions()
		if err != nil {
			log.Error(errorSubject, "err", err)
			if strings.Contains(strings.ToLower(err.Error()), "method not found") {
				log.Info("stop scanning pool tx as eth_pendingTransactions not found")
				return
			}
			time.Sleep(retryIntervalInScanJob)
			continue
		}
		for _, tx := range txs {
			txid := tx.Hash.String()
			if scannedTxs.IsTxScanned(txid) {
				continue
			}
			log.Trace(scanSubject, "txid", txid)
			b.processTransaction(txid)
			scannedTxs.CacheScannedTx(txid)
		}
		time.Sleep(restIntervalInScanJob)
	}
}
