package eth

import (
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	"github.com/fsn-dev/crossChain-Bridge/tokens/tools"
)

func (b *Bridge) processTransaction(txid string) {
	_ = b.processSwapout(txid)
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
