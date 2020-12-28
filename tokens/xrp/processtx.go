package xrp

import (
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

func (b *Bridge) processTransaction(txid string) {
	if b.IsSrc {
		swapInfo, err := b.VerifyTransaction("XRP", txid, false)
		swapInfos := []*tokens.TxSwapInfo{swapInfo}
		errs := []error{err}
		tools.RegisterSwapin(txid, swapInfos, errs)
	}
	return
}
