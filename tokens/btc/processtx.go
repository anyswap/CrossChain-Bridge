package btc

import (
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

func (b *Bridge) processTransaction(txid string) {
	p2shBindAddr, err := b.checkSwapinTxType(txid)
	if err != nil {
		return
	}
	if p2shBindAddr != "" {
		b.processP2shSwapin(txid, p2shBindAddr)
	} else {
		b.processSwapin(txid)
	}
}

func (b *Bridge) processSwapin(txid string) {
	swapInfo, err := b.verifySwapinTx(PairID, txid, true)
	tools.RegisterSwapin(txid, []*tokens.TxSwapInfo{swapInfo}, []error{err})
}

func (b *Bridge) processP2shSwapin(txid, bindAddress string) {
	swapInfo, err := b.verifyP2shSwapinTx(PairID, txid, bindAddress, true)
	tools.RegisterP2shSwapin(txid, swapInfo, err)
}

func (b *Bridge) checkSwapinTxType(txHash string) (p2shBindAddr string, err error) {
	tx, err := b.GetTransactionByHash(txHash)
	if err != nil {
		log.Debug(b.ChainConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txHash, "err", err)
		return "", tokens.ErrTxNotFound
	}
	tokenCfg := b.GetTokenConfig(PairID)
	if tokenCfg == nil {
		return "", fmt.Errorf("swap pair '%v' is not configed", PairID)
	}
	depositAddress := tokenCfg.DepositAddress
	txFrom := getTxFrom(tx.Vin, depositAddress)
	for _, output := range tx.Vout {
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
