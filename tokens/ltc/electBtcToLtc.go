package ltc

import (
	belectrs "github.com/anyswap/CrossChain-Bridge/tokens/btc/electrs"
)

// ToLTCVout convert address in ElectTx to LTC format
func (b *Bridge) ToLTCVout(vout *belectrs.ElectTxOut) *belectrs.ElectTxOut {
	// ScriptpubkeyAddress
	if vout.ScriptpubkeyAddress == nil {
		return vout
	}
	addr, err := b.ConvertBTCAddress(*vout.ScriptpubkeyAddress, "Main")
	if err != nil {
		return vout
	}
	*vout.ScriptpubkeyAddress = addr.String()
	return vout
}

// ToLTCTx convert address in ElectTx to LTC format
func (b *Bridge) ToLTCTx(tx *belectrs.ElectTx) *belectrs.ElectTx {
	// Vin Prevout ToLTCVout
	for _, vin := range tx.Vin {
		if vin.Prevout != nil {
			*vin.Prevout = *b.ToLTCVout(vin.Prevout)
		}
	}
	// Vout ToLTCVout
	for _, vout := range tx.Vout {
		*vout = *b.ToLTCVout(vout)
	}
	return tx
}
