package nebulas

import (
	"math/big"
	"strings"
)

// known network and chainID map
var (
	// Nebulas chain
	NebulasNetworkAndChainIDMap = map[string]*big.Int{
		"mainnet": big.NewInt(1),
		"testnet": big.NewInt(1001),
	}
)

// IsCustomNetwork is custom network
func IsCustomNetwork(networkID string) bool {
	return strings.EqualFold(networkID, "custom")
}

// GetChainIDOfNetwork get chainID of networkID
func GetChainIDOfNetwork(networkAndChainIDMap map[string]*big.Int, networkID string) *big.Int {
	if chainID, exist := networkAndChainIDMap[networkID]; exist {
		return chainID
	}
	return nil
}
