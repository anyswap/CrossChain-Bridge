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
	return tools.RegisterSwapin(txid, swapInfo.Bind, err)
}

func (b *Bridge) processP2shSwapin(txid, bindAddress string) error {
	swapInfo, err := b.VerifyP2shTransaction(txid, bindAddress, true)
	return tools.RegisterP2shSwapin(txid, swapInfo.Bind, err)
}

func (b *Bridge) checkSwapinTxType(txHash string) (p2shBindAddr string, err error) {
	tx, err := b.GetTransactionByHash(txHash)
	if err != nil {
		log.Debug(b.TokenConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txHash, "err", err)
		return "", tokens.ErrTxNotFound
	}
	depositAddress := b.TokenConfig.DepositAddress
	txFrom := getTxFrom(tx.Vin, depositAddress)
	for _, output := range tx.Vout {
		if output.ScriptpubkeyAddress == nil {
			continue
		}
		scriptPubkeyAsm := *output.ScriptpubkeyAsm
		if regexCLTVCSV.FindString(scriptPubkeyAsm) != "" {
			continue
		}
		switch *output.ScriptpubkeyType {
		case p2shType:
			if p2shBindAddr == "" { // use the first registered p2sh address
				p2shAddress := *output.ScriptpubkeyAddress
				p2shBindAddr = tools.GetP2shBindAddress(p2shAddress)
			}
		default:
			if txFrom == depositAddress {
				continue // ignore is sender is configed deposit address
			}
			if *output.ScriptpubkeyAddress == depositAddress {
				return "", nil // p2pkh first if exist
			}
		}
	}
	if p2shBindAddr != "" {
		return p2shBindAddr, nil
	}
	return "", tokens.ErrTxWithWrongReceiver
}
