package cosmos

import (
	"strings"

	"github.com/anyswap/CrossChain-Bridge/log"
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
	log.Info("cosmos processSwapin", "tx", tx)
	swapInfos, errs := b.verifySwapinTx(tx, true)
	txid := strings.ToLower(tx.TxHash)
	log.Debug("cosmos processSwapin", "txid", txid, "swapinfos", swapInfos, "errs", errs)
	tools.RegisterSwapin(txid, swapInfos, errs)
}
