package cosmos

import (
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc/electrs"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

func (b *Bridge) processTransaction(txdata string) {
	// GetTransaction
	// txstring is protobyte bytes of Tx
	if b.IsSrc {
		b.processSwapin(txdata)
	} else {
		return
	}
}

func (b *Bridge) processSwapin(txdata string) {
	swapInfos, errs := verifySwapinTx(PairID, txdata)
	tools.RegisterSwapin(swapInfo.Hash, swapInfos, errs)
}
