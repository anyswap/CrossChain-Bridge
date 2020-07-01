package btc

import (
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

func (b *Bridge) processTransaction(txid string) {
	if tools.IsSwapinExist(txid) {
		return
	}
	p2shAddress, err := b.checkSwapinTxType(txid)
	if err != nil {
		return
	}
	if p2shAddress != "" {
		_ = b.processP2shSwapin(txid, p2shAddress)
	} else {
		_ = b.processSwapin(txid)
	}
}

func (b *Bridge) processSwapin(txid string) error {
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

func (b *Bridge) processP2shSwapin(txid, p2shAddress string) error {
	bindAddress := tools.GetP2shBindAddress(p2shAddress)
	if bindAddress == "" {
		return tokens.ErrTxWithWrongReceiver
	}
	swapInfo, err := b.VerifyP2shTransaction(txid, bindAddress, true)
	if !tokens.ShouldRegisterSwapForError(err) {
		return err
	}
	err = tools.RegisterP2shSwapin(txid, swapInfo.Bind)
	if err != nil {
		log.Trace("[scan] processP2shSwapin", "txid", txid, "err", err)
	}
	return err
}

func (b *Bridge) checkSwapinTxType(txHash string) (p2shAddress string, err error) {
	tx, err := b.GetTransactionByHash(txHash)
	if err != nil {
		log.Debug(b.TokenConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txHash, "err", err)
		return "", tokens.ErrTxNotFound
	}
	// p2pkh or p2sh swapin is decided by who appears first
	for _, output := range tx.Vout {
		switch *output.ScriptpubkeyType {
		case p2pkhType:
			if *output.ScriptpubkeyAddress == b.TokenConfig.DcrmAddress {
				return "", nil
			}
		case p2shType:
			return *output.ScriptpubkeyAddress, nil
		}
	}
	return "", tokens.ErrTxWithWrongReceiver
}
