package btc

import (
	"fmt"

	"github.com/btcsuite/btcutil"
)

// DecodeAddress decode address
func (b *Bridge) DecodeAddress(addr string) (address btcutil.Address, err error) {
	chainConfig := b.Inherit.GetChainParams()
	address, err = btcutil.DecodeAddress(addr, chainConfig)
	if err != nil {
		return
	}
	if !address.IsForNet(chainConfig) {
		err = fmt.Errorf("invalid address for net")
		return
	}
	return
}

// NewAddressPubKeyHash encap
func (b *Bridge) NewAddressPubKeyHash(pkData []byte) (*btcutil.AddressPubKeyHash, error) {
	return btcutil.NewAddressPubKeyHash(btcutil.Hash160(pkData), b.Inherit.GetChainParams())
}

// NewAddressScriptHash encap
func (b *Bridge) NewAddressScriptHash(redeemScript []byte) (*btcutil.AddressScriptHash, error) {
	return btcutil.NewAddressScriptHash(redeemScript, b.Inherit.GetChainParams())
}

// IsValidAddress check address
func (b *Bridge) IsValidAddress(addr string) bool {
	_, err := b.DecodeAddress(addr)
	return err == nil
}

// IsP2pkhAddress check p2pkh addrss
func (b *Bridge) IsP2pkhAddress(addr string) bool {
	address, err := b.DecodeAddress(addr)
	if err != nil {
		return false
	}
	_, ok := address.(*btcutil.AddressPubKeyHash)
	return ok
}

// IsP2shAddress check p2sh addrss
func (b *Bridge) IsP2shAddress(addr string) bool {
	address, err := b.DecodeAddress(addr)
	if err != nil {
		return false
	}
	_, ok := address.(*btcutil.AddressScriptHash)
	return ok
}

// DecodeWIF decode wif
func DecodeWIF(wif string) (*btcutil.WIF, error) {
	return btcutil.DecodeWIF(wif)
}
