package worker

import (
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// StartScanJob scan job
func StartScanJob(isServer bool) {
	if tokens.SrcBridge.GetChainConfig().EnableScan {
		if scanChainSupport, ok := tokens.SrcBridge.(tokens.ScanChainSupport); ok {
			go scanChainSupport.StartChainTransactionScanJob()
			go scanChainSupport.StartPoolTransactionScanJob()
		}
		if scanHistorySupport, ok := tokens.SrcBridge.(tokens.ScanHistorySupport); ok {
			go scanHistorySupport.StartSwapHistoryScanJob()
		}
	}

	if tokens.DstBridge.GetChainConfig().EnableScan {
		if scanChainSupport, ok := tokens.DstBridge.(tokens.ScanChainSupport); ok {
			go scanChainSupport.StartChainTransactionScanJob()
			go scanChainSupport.StartPoolTransactionScanJob()
		}
	}
}
