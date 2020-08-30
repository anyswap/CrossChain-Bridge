package eth

import (
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

func (b *Bridge) processTransaction(txid string) {
	if b.IsSrc {
		b.processSwapin(txid)
	} else {
		b.processSwapout(txid)
	}
}

func (b *Bridge) processSwapin(txid string) {
	swapInfos, errs := b.verifySwapinTx(txid, true)
	tools.RegisterSwapin(txid, swapInfos, errs)
}

func (b *Bridge) processSwapout(txid string) {
	swapInfos, errs := b.verifySwapoutTx(txid, true)
	tools.RegisterSwapout(txid, swapInfos, errs)
}
