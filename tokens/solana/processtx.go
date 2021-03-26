package solana

import (
	"strings"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

func (b *Bridge) processTransaction(tx *GetConfirmedTransactonResult) {
	if b.IsSrc {
		b.processSwapin(tx)
	} else {
		return
	}
}

func (b *Bridge) processSwapin(tx *GetConfirmedTransactonResult) {
	log.Info("soalna processSwapin", "tx", tx)
	swapInfos, errs := b.verifySwapinTx(PairID, tx, true)
	txid := strings.ToLower(tx.Transaction.Signatures[0].String())
	log.Debug("solana processSwapin", "txid", txid, "swapinfos", swapInfos, "errs", errs)
	tools.RegisterSwapin(txid, swapInfos, errs)
}
