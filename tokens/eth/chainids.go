package eth

import (
	"math/big"
	"strings"
)

// known network and chainID map
var (
	// ETH chain
	EthNetworkAndChainIDMap = map[string]*big.Int{
		"mainnet": big.NewInt(1),
		"rinkeby": big.NewInt(4),
		"goerli":  big.NewInt(5),
	}

	// ETC chain
	EtcNetworkAndChainIDMap = map[string]*big.Int{
		"mainnet": big.NewInt(61),
		"kotti":   big.NewInt(6),
		"mordor":  big.NewInt(63),
	}

	// FSN chain
	FsnNetworkAndChainIDMap = map[string]*big.Int{
		"mainnet": big.NewInt(32659),
		"testnet": big.NewInt(46688),
		"devnet":  big.NewInt(55555),
	}

	// OKEX chain
	OkexNetworkAndChainIDMap = map[string]*big.Int{
		"mainnet": big.NewInt(66),
	}

	// Kusama ecosystem chain (eg. Moonriver)
	KusamaNetworkAndChainIDMap = map[string]*big.Int{
		"moonriver": big.NewInt(1285),
		"shiden":    big.NewInt(336),
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
