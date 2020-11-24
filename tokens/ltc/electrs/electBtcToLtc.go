package electrs

import (
	belectrs "github.com/anyswap/CrossChain-Bridge/tokens/btc/electrs"
	"github.com/btcsuite/btcutil"
	"github.com/ltcsuite/ltcutil"
)

var convertB2L func(string, string) (ltcutil.Address, error)
var convertL2B func(string, string) (btcutil.Address, error)

// SetL2BConvertor set addressConvertor to f
func SetL2BConvertor(f func(string, string) (btcutil.Address, error)) {
	convertL2B = f
}

// SetB2LConvertor set addressConvertor to f
func SetB2LConvertor(f func(string, string) (ltcutil.Address, error)) {
	convertB2L = f
}

// ToLTCVout convert address in ElectTx to LTC format
func ToLTCVout(vout belectrs.ElectTxOut) belectrs.ElectTxOut {
	// ScriptpubkeyAddress
	addr, err := convertB2L(*vout.ScriptpubkeyAddress, "Main")
	if err != nil {
		return vout
	}
	*vout.ScriptpubkeyAddress = addr.String()
	return vout
}

// ToLTCTx convert address in ElectTx to LTC format
func ToLTCTx(tx belectrs.ElectTx) belectrs.ElectTx {
	// Vin Prevout ToLTCVout
	for _, vin := range tx.Vin {
		if vin.Prevout != nil {
			*vin.Prevout = ToLTCVout(*vin.Prevout)
		}
	}
	// Vout ToLTCVout
	for _, vout := range tx.Vout {
		*vout = ToLTCVout(*vout)
	}
	return tx
}
