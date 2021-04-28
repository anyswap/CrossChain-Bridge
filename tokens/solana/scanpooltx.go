package solana

import (
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

var (
	scannedTxs = tools.NewCachedScannedTxs(3000)
)

// StartPoolTransactionScanJob scan job
func (b *Bridge) StartPoolTransactionScanJob() {
	return
}
