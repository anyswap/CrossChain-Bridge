package eth

import (
	"github.com/anyswap/CrossChain-Bridge/common"
	mapset "github.com/deckarep/golang-set"
)

var (
	cachedContractAddrs    = mapset.NewSet()
	maxCachedContractAddrs = 50

	cachedNoncontractAddrs    = mapset.NewSet()
	maxNoncachedContractAddrs = 500
)

// ShouldCheckAddressMixedCase check address mixed case
// eg. RSK chain do not check mixed case or not same as eth
func (b *Bridge) ShouldCheckAddressMixedCase() bool {
	return true
}

// IsValidAddress check address
func (b *Bridge) IsValidAddress(address string) bool {
	if !common.IsHexAddress(address) {
		return false
	}
	if !b.ShouldCheckAddressMixedCase() {
		return true
	}
	unprefixedHex, ok, hasUpperChar := common.GetUnprefixedHex(address)
	if hasUpperChar {
		// valid checksum
		if unprefixedHex != common.HexToAddress(address).String()[2:] {
			return false
		}
	}
	return ok
}

// IsContractAddress is contract address
func (b *Bridge) IsContractAddress(address string) (bool, error) {
	if cachedNoncontractAddrs.Contains(address) {
		return false, nil
	}
	if cachedContractAddrs.Contains(address) {
		return true, nil
	}

	code, err := b.getContractCode(address, false)
	if err != nil {
		return false, err
	}
	if len(code) > 1 { // unexpect RSK getCode return 0x00
		addCachedContractAddr(address)
		return true, nil
	}
	addNoncachedContractAddr(address)
	return false, nil
}

func addNoncachedContractAddr(address string) {
	if cachedNoncontractAddrs.Cardinality() >= maxNoncachedContractAddrs {
		cachedNoncontractAddrs.Pop()
	}
	cachedNoncontractAddrs.Add(address)
}

func addCachedContractAddr(address string) {
	if cachedContractAddrs.Cardinality() >= maxCachedContractAddrs {
		cachedContractAddrs.Pop()
	}
	cachedContractAddrs.Add(address)
}
