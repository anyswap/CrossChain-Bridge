package eth

import (
	"time"

	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/mongodb"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
)

func (b *Bridge) processTransaction(txid string) {
	_ = b.processSwapout(txid)
}

func (b *Bridge) processSwapout(txid string) error {
	swapInfo, err := b.VerifyTransaction(txid, true)
	if !tokens.ShouldRegisterSwapForError(err) {
		return err
	}
	return b.registerSwapout(txid, swapInfo.Bind)
}

func (b *Bridge) registerSwapout(txid string, bind string) error {
	log.Info("[scan] register swapout", "tx", txid, "bind", bind)
	swap := &mongodb.MgoSwap{
		Key:       txid,
		TxID:      txid,
		TxType:    uint32(tokens.SwapoutTx),
		Bind:      bind,
		Status:    mongodb.TxNotStable,
		Timestamp: time.Now().Unix(),
	}
	return mongodb.AddSwapout(swap)
}
