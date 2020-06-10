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

// IsP2pkhAddress check p2sh addrss
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

// GetChainConfig get chain config (net params)
func (b *Bridge) GetChainConfig() *chaincfg.Params {
	token := b.TokenConfig
	networkID := strings.ToLower(token.NetID)
	switch networkID {
	case "mainnet":
		return &chaincfg.MainNetParams
	case "testnet3":
		return &chaincfg.TestNet3Params
	}
	return &chaincfg.TestNet3Params
}
