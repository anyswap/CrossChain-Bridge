package cosmos

import (
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (b *Bridge) processTransaction(tx sdk.TxResponse) {
	if b.IsSrc {
		b.processSwapin(tx)
	} else {
		return
	}
}

func (b *Bridge) processSwapin(tx sdk.TxResponse) {
	swapInfos, errs := b.verifySwapinTx(PairID, tx, true)
	txid := tx.TxHash
	tools.RegisterSwapin(txid, swapInfos, errs)
}
