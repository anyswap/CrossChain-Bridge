package worker

import (
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc"
)

// StartScanJob scan job
func StartScanJob(isServer bool) {
	if tokens.SrcBridge.GetChainConfig().EnableScan {
		go tokens.SrcBridge.StartChainTransactionScanJob()
		go tokens.SrcBridge.StartPoolTransactionScanJob()
		if btc.BridgeInstance != nil {
			go btc.BridgeInstance.StartSwapHistoryScanJob()
		}
	}

	if tokens.DstBridge.GetChainConfig().EnableScan {
		go tokens.DstBridge.StartChainTransactionScanJob()
		go tokens.DstBridge.StartPoolTransactionScanJob()
	}
}
