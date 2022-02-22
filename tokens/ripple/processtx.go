package ripple

import (
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

func (b *Bridge) processTransaction(pairID, txid string) {
	if b.IsSrc {
		swapInfo, err := b.VerifyTransaction(pairID, txid, false)
		swapInfos := []*tokens.TxSwapInfo{swapInfo}
		errs := []error{err}
		tools.RegisterSwapin(txid, swapInfos, errs)
	}
	return
}
