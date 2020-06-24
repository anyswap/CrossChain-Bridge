package btc

import (
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

func (b *Bridge) processTransaction(txid string) {
	_ = b.processSwapin(txid)
	_ = b.processP2shSwapin(txid)
}

func (b *Bridge) processSwapin(txid string) error {
	if tools.IsSwapinExist(txid) {
		return nil
	}
	swapInfo, err := b.VerifyTransaction(txid, true)
	if !tokens.ShouldRegisterSwapForError(err) {
		return err
	}
	err = tools.RegisterSwapin(txid, swapInfo.Bind)
	if err != nil {
		log.Trace("[scan] processSwapin", "txid", txid, "err", err)
	}
	return err
}

func (b *Bridge) processP2shSwapin(txid string) error {
	if tools.IsSwapinExist(txid) {
		return nil
	}
	swapInfo, err := b.checkP2shTransaction(txid, true)
	if !tokens.ShouldRegisterSwapForError(err) {
		return err
	}
	err = tools.RegisterP2shSwapin(txid, swapInfo.Bind)
	if err != nil {
		log.Trace("[scan] processP2shSwapin", "txid", txid, "err", err)
	}
	return err
}

func (b *Bridge) checkP2shTransaction(txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	tx, err := b.GetTransactionByHash(txHash)
	if err != nil {
		log.Debug(b.TokenConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txHash, "err", err)
		return nil, tokens.ErrTxNotFound
	}
	var bindAddress, p2shAddress string
	for _, output := range tx.Vout {
		if *output.ScriptpubkeyType == p2shType {
			p2shAddress = *output.ScriptpubkeyAddress
			bindAddress = tools.GetP2shBindAddress(p2shAddress)
			if bindAddress != "" {
				break
			}
		}
	}
	if bindAddress == "" {
		return nil, tokens.ErrTxWithWrongReceiver
	}
	return b.VerifyP2shTransaction(txHash, bindAddress, allowUnstable)
}
