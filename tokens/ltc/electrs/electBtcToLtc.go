package electrs

import (
	belectrs "github.com/anyswap/CrossChain-Bridge/tokens/btc/electrs"
	"github.com/ltcsuite/ltcutil"
)

var addressConvertor func(addr, BTCNet string) (address ltcutil.Address, err error)

// SetAddressConvertor set addressConvertor to f
func SetAddressConvertor(f func(addr, BTCNet string) (address ltcutil.Address, err error)) {
	addressConvertor = f
}

// ToLTCVout convert address in ElectTx to LTC format
func ToLTCVout(vout belectrs.ElectTxOut) belectrs.ElectTxOut {
	// ScriptpubkeyAddress
	addr, err := addressConvertor(*vout.ScriptpubkeyAddress, "Main")
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
