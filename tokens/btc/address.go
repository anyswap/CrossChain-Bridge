package btc

import (
	"strings"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
)

func (b *BtcBridge) IsValidAddress(addr string) bool {
	chainConfig := b.GetChainConfig()
	address, err := btcutil.DecodeAddress(addr, chainConfig)
	if err != nil {
		return false
	}
	return address.IsForNet(chainConfig)
}

func (b *BtcBridge) GetChainConfig() *chaincfg.Params {
	token, _ := b.GetTokenAndGateway()
	networkID := strings.ToLower(*token.NetID)
	switch networkID {
	case "mainnet":
		return &chaincfg.MainNetParams
	case "testnet3":
		return &chaincfg.TestNet3Params
	}
	return &chaincfg.TestNet3Params
}
