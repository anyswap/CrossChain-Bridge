package btc

import (
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc/electrs"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

// ProcessTransaction process tx
func (b *Bridge) ProcessTransaction(tx *electrs.ElectTx) {
	txid := *tx.Txid
	if tools.IsSwapinExist(txid) {
		return
	}
	b.processTransactionImpl(tx)
}

func (b *Bridge) processTransaction(txid string) {
	if tools.IsSwapinExist(txid) {
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
		log.Debug("[processTransaction] "+b.TokenConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txid, "err", err)
		return
	}
	b.processTransactionImpl(tx)
}

func (b *Bridge) processTransactionImpl(tx *electrs.ElectTx) {
	p2shBindAddr, err := b.checkSwapinTxType(tx)
	if err != nil {
		return
	}
	txid := *tx.Txid
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

func (b *Bridge) checkSwapinTxType(tx *electrs.ElectTx) (p2shBindAddr string, err error) {
	depositAddress := b.TokenConfig.DepositAddress
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
