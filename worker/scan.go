package worker

import (
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// StartScanJob scan job
func StartScanJob(isServer bool) {
	go tokens.SrcBridge.StartPoolTransactionScanJob()
	go tokens.SrcBridge.StartChainTransactionScanJob()
	go tokens.SrcBridge.StartSwapHistoryScanJob()

	go tokens.DstBridge.StartPoolTransactionScanJob()
	go tokens.DstBridge.StartChainTransactionScanJob()
	go tokens.DstBridge.StartSwapHistoryScanJob()
}
