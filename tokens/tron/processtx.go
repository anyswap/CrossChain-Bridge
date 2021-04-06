package tron

import (
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
)

func (b *Bridge) processTransaction(txext *api.TransactionExtention) {
	if b.IsSrc {
		b.processSwapin(txext)
	} else {
		b.processSwapout(txext)
	}
}

func (b *Bridge) processSwapin(txext *api.TransactionExtention) {
	tx := TransactionExtention{
		Transaction: txext.Transaction,
		Txid: txext.GetTxid(),
	}
	swapInfos, errs := b.verifySwapinTx(tx, true)
	tools.RegisterSwapin(fmt.Sprintf("%X", txext.GetTxid()), swapInfos, errs)
}

func (b *Bridge) processSwapout(txext *api.TransactionExtention) {
	tx := TransactionExtention{
		Transaction: txext.Transaction,
		Txid: txext.GetTxid(),
	}
	swapInfos, errs := b.verifySwapoutTx(tx, true)
	tools.RegisterSwapout(fmt.Sprintf("%X", txext.GetTxid()), swapInfos, errs)
}
