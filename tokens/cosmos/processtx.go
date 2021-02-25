package cosmos

import (
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

func (b *Bridge) processTransaction(txdata string) {
	// txstring is protobyte bytes of Tx
	if b.IsSrc {
		b.processSwapin(txdata)
	} else {
		return
	}
}

func (b *Bridge) processSwapin(txid string) {
	swapInfos, errs := b.verifySwapinTx(PairID, txid, true)
	tools.RegisterSwapin(txid, swapInfos, errs)
}
