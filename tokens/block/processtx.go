package block

import (
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc/electrs"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

func (b *Bridge) processTransaction(txid string) {
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
	p2shBindAddrs, err := b.CheckSwapinTxType(tx)
	if err != nil {
		return
	}
	txid := *tx.Txid
	if len(p2shBindAddrs) > 0 {
		for _, p2shBindAddr := range p2shBindAddrs {
			b.processP2shSwapin(txid, p2shBindAddr)
		}
	} else {
		b.processSwapin(txid)
	}
}

func (b *Bridge) processSwapin(txid string) {
	if tools.IsSwapExist(txid, PairID, "", true) {
		return
	}
	swapInfo, err := b.verifySwapinTx(PairID, txid, true)
	tools.RegisterSwapin(txid, []*tokens.TxSwapInfo{swapInfo}, []error{err})
}

func (b *Bridge) processP2shSwapin(txid, bindAddress string) {
	if tools.IsSwapExist(txid, PairID, bindAddress, true) {
		return
	}
	swapInfo, err := b.verifyP2shSwapinTx(PairID, txid, bindAddress, true)
	tools.RegisterP2shSwapin(txid, swapInfo, err)
}

func isP2pkhSwapinPrior(tx *electrs.ElectTx, depositAddress string) bool {
	txFrom := getTxFrom(tx.Vin, depositAddress)
	if txFrom == depositAddress {
		return false
	}
	var memoScript string
	for i := len(tx.Vout) - 1; i >= 0; i-- { // reverse iterate
		output := tx.Vout[i]
		if *output.ScriptpubkeyType == opReturnType {
			memoScript = *output.ScriptpubkeyAsm
			break
		}
	}
	bindAddress, bindOk := GetBindAddressFromMemoScipt(memoScript)
	return bindOk && tokens.DstBridge.IsValidAddress(bindAddress)
}

// CheckSwapinTxType check swapin type
func (b *Bridge) CheckSwapinTxType(tx *electrs.ElectTx) (p2shBindAddrs []string, err error) {
	tokenCfg := b.GetTokenConfig(PairID)
	if tokenCfg == nil {
		return nil, fmt.Errorf("swap pair '%v' is not configed", PairID)
	}
	depositAddress := tokenCfg.DepositAddress
	p2pkhSwapinPrior := isP2pkhSwapinPrior(tx, depositAddress)
	p2shAddressMap := make(map[string]struct{})
	for _, output := range tx.Vout {
		if output.ScriptpubkeyAddress == nil {
			continue
		}
		switch *output.ScriptpubkeyType {
		case p2shType:
			// use the first registered p2sh address
			p2shAddress := *output.ScriptpubkeyAddress
			if _, exist := p2shAddressMap[p2shAddress]; exist {
				continue
			}
			p2shAddressMap[p2shAddress] = struct{}{}
			p2shBindAddr := tools.GetP2shBindAddress(p2shAddress)
			if p2shBindAddr != "" {
				p2shBindAddrs = append(p2shBindAddrs, p2shBindAddr)
			}
		case p2pkhType:
			if p2pkhSwapinPrior && *output.ScriptpubkeyAddress == depositAddress {
				return nil, nil // use p2pkh if exist
			}
		}
	}
	if len(p2shBindAddrs) > 0 {
		return p2shBindAddrs, nil
	}
	return nil, tokens.ErrTxWithWrongReceiver
}
