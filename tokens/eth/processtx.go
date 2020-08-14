package eth

import (
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

func (b *Bridge) processTransaction(txid string) {
	if b.IsSrc {
		_ = b.processSwapin(txid)
	} else {
		_ = b.processSwapout(txid)
	}
}

func (b *Bridge) processSwapin(txid string) error {
	if tools.IsSwapinExist(txid) {
		return nil
	}
	swapInfo, err := b.VerifyTransaction(txid, true)
	return tools.RegisterSwapin(txid, swapInfo.Bind, err)
}

func (b *Bridge) processSwapout(txid string) error {
	if tools.IsSwapoutExist(txid) {
		return nil
	}
	swapInfo, err := b.VerifyTransaction(txid, true)
	return tools.RegisterSwapout(txid, swapInfo.Bind, err)
}
