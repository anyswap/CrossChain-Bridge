package worker

import (
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// StartScanJob scan job
func StartScanJob(isServer bool) {
	srcTokenCfg, _ := tokens.SrcBridge.GetTokenAndGateway()
	if srcTokenCfg.EnableScan {
		go tokens.SrcBridge.StartPoolTransactionScanJob()
		go tokens.SrcBridge.StartChainTransactionScanJob()
		go tokens.SrcBridge.StartSwapHistoryScanJob()
	}

	dstTokenCfg, _ := tokens.DstBridge.GetTokenAndGateway()
	if dstTokenCfg.EnableScan {
		go tokens.DstBridge.StartPoolTransactionScanJob()
		go tokens.DstBridge.StartChainTransactionScanJob()
		go tokens.DstBridge.StartSwapHistoryScanJob()
	}
}
