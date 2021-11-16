package colx

import (
	belectrs "github.com/anyswap/CrossChain-Bridge/tokens/btc/electrs"
)

// ToCOLXVout convert address in ElectTx to COLX format
func (b *Bridge) ToCOLXVout(vout *belectrs.ElectTxOut) *belectrs.ElectTxOut {
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

// ToCOLXTx convert address in ElectTx to COLX format
func (b *Bridge) ToCOLXTx(tx *belectrs.ElectTx) *belectrs.ElectTx {
	// Vin Prevout ToCOLXVout
	for _, vin := range tx.Vin {
		if vin.Prevout != nil {
			*vin.Prevout = *b.ToCOLXVout(vin.Prevout)
		}
	}
	// Vout ToCOLXVout
	for _, vout := range tx.Vout {
		*vout = *b.ToCOLXVout(vout)
	}
	return tx
}
