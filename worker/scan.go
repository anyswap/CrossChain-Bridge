package worker

import (
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// StartScanJob scan job
func StartScanJob(isServer bool) {
	if tokens.SrcBridge.GetChainConfig().EnableScan {
		go tokens.SrcBridge.StartChainTransactionScanJob()
	}

	if tokens.DstBridge.GetChainConfig().EnableScan {
		go tokens.DstBridge.StartChainTransactionScanJob()
	}
}
