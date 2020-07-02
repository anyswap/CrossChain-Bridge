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
	p2shBindAddr, err := b.checkSwapinTxType(txid)
	if err != nil {
		return
	}
	if p2shBindAddr != "" {
		_ = b.processP2shSwapin(txid, p2shBindAddr)
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

func (b *Bridge) processP2shSwapin(txid, bindAddress string) error {
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

func (b *Bridge) checkSwapinTxType(txHash string) (p2shBindAddr string, err error) {
	tx, err := b.GetTransactionByHash(txHash)
	if err != nil {
		log.Debug(b.TokenConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txHash, "err", err)
		return "", tokens.ErrTxNotFound
	}
	dcrmAddress := b.TokenConfig.DcrmAddress
	txFrom := getTxFrom(tx.Vin, dcrmAddress)
	for _, output := range tx.Vout {
		switch *output.ScriptpubkeyType {
		case p2pkhType:
			if txFrom == dcrmAddress {
				continue // p2pkh is ignore is sender is configed dcrm address
			}
			if *output.ScriptpubkeyAddress == dcrmAddress {
				return "", nil // p2pkh first if exist
			}
		case p2shType:
			if p2shBindAddr == "" { // use the first registered p2sh address
				p2shAddress := *output.ScriptpubkeyAddress
				p2shBindAddr = tools.GetP2shBindAddress(p2shAddress)
			}
		}
	}
	if p2shBindAddr != "" {
		return p2shBindAddr, nil
	}
	return "", tokens.ErrTxWithWrongReceiver
}
