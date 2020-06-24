package eth

import (
	"github.com/anyswap/CrossChain-Bridge/tokens"
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
	if !tokens.ShouldRegisterSwapForError(err) {
		return err
	}
	return tools.RegisterSwapin(txid, swapInfo.Bind)
}

func (b *Bridge) processSwapout(txid string) error {
	if tools.IsSwapoutExist(txid) {
		return nil
	}
	swapInfo, err := b.VerifyTransaction(txid, true)
	if !tokens.ShouldRegisterSwapForError(err) {
		return err
	}
	return tools.RegisterSwapout(txid, swapInfo.Bind)
}
