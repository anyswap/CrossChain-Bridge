package btc

import (
	"strings"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
)

// IsValidAddress check address
func (b *Bridge) IsValidAddress(addr string) bool {
	chainConfig := b.GetChainConfig()
	address, err := btcutil.DecodeAddress(addr, chainConfig)
	if err != nil {
		return false
	}
	return address.IsForNet(chainConfig)
}

// IsP2pkhAddress check p2pkh addrss
func (b *Bridge) IsP2pkhAddress(addr string) bool {
	chainConfig := b.GetChainConfig()
	address, err := btcutil.DecodeAddress(addr, chainConfig)
	if err != nil {
		return false
	}
	if !address.IsForNet(chainConfig) {
		return false
	}
	_, ok := address.(*btcutil.AddressPubKeyHash)
	return ok
}

// IsP2shAddress check p2sh addrss
func (b *Bridge) IsP2shAddress(addr string) bool {
	chainConfig := b.GetChainConfig()
	address, err := btcutil.DecodeAddress(addr, chainConfig)
	if err != nil {
		return false
	}
	if !address.IsForNet(chainConfig) {
		return false
	}
	_, ok := address.(*btcutil.AddressScriptHash)
	return ok
}

// GetChainConfig get chain config (net params)
func (b *Bridge) GetChainConfig() *chaincfg.Params {
	networkID := strings.ToLower(b.ChainConfig.NetID)
	switch networkID {
	case netMainnet:
		return &chaincfg.MainNetParams
	case netTestnet3:
		return &chaincfg.TestNet3Params
	}
	return &chaincfg.TestNet3Params
}
