package btc

import (
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc/electrs"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

func (b *Bridge) processTransaction(txid string) {
	if tools.IsSwapExist(txid, PairID, true) {
		return
	}
	var tx *electrs.ElectTx
	var err error
	for i := 0; i < 2; i++ {
		tx, err = b.GetTransactionByHash(txid)
		if err == nil {
			break
		}
	}
	if err != nil {
		log.Debug("[processTransaction] "+b.ChainConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txid, "err", err)
		return
	}
	b.processTransactionImpl(tx)
}

func (b *Bridge) processTransactionImpl(tx *electrs.ElectTx) {
	p2shBindAddr, err := b.CheckSwapinTxType(tx)
	if err != nil {
		return
	}
	txid := *tx.Txid
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

// CheckSwapinTxType check swapin type
func (b *Bridge) CheckSwapinTxType(tx *electrs.ElectTx) (p2shBindAddr string, err error) {
	tokenCfg := b.GetTokenConfig(PairID)
	if tokenCfg == nil {
		return "", fmt.Errorf("swap pair '%v' is not configed", PairID)
	}
	depositAddress := tokenCfg.DepositAddress
	var txFrom string
	for _, output := range tx.Vout {
		if output.ScriptpubkeyAddress == nil {
			continue
		}
		switch *output.ScriptpubkeyType {
		case p2shType:
			// use the first registered p2sh address
			p2shAddress := *output.ScriptpubkeyAddress
			p2shBindAddr = tools.GetP2shBindAddress(p2shAddress)
			if p2shBindAddr != "" {
				return p2shBindAddr, nil
			}
		case p2pkhType:
			if *output.ScriptpubkeyAddress == depositAddress {
				if txFrom == "" {
					txFrom = getTxFrom(tx.Vin, depositAddress)
				}
				if txFrom == depositAddress {
					continue // ignore if sender is deposit address
				}
				return "", nil // use p2pkh if exist
			}
		}
	}
	return "", tokens.ErrTxWithWrongReceiver
}
