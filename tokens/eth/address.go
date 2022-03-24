package eth

import (
	"strings"

	"github.com/anyswap/CrossChain-Bridge/common"
	mapset "github.com/deckarep/golang-set"
)

var (
	cachedContractAddrs    = mapset.NewSet()
	maxCachedContractAddrs = 50

	cachedNoncontractAddrs    = mapset.NewSet()
	maxNoncachedContractAddrs = 500

	contractCodeHashes    = make(map[common.Address]common.Hash)
	maxContractCodeHashes = 100
)

// ShouldCheckAddressMixedCase check address mixed case
// eg. RSK chain do not check mixed case or not same as eth
func (b *Bridge) ShouldCheckAddressMixedCase() bool {
	return !b.ChainConfig.IgnoreCheckAddressMixedCase
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
		if strings.ToUpper(address) == address {
			return true
		}
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

// GetContractCodeHash get contract code hash
func (b *Bridge) GetContractCodeHash(contract common.Address) common.Hash {
	codeHash, exist := contractCodeHashes[contract]
	if exist {
		return codeHash
	}
	if cachedNoncontractAddrs.Contains(contract.String()) {
		return common.Hash{}
	}
	if len(contractCodeHashes) > maxContractCodeHashes {
		contractCodeHashes = make(map[common.Address]common.Hash) // clear
	}

	code, err := b.getContractCode(contract.String(), false)
	if err == nil && len(code) > 1 {
		codeHash = common.Keccak256Hash(code)
		contractCodeHashes[contract] = codeHash
	}
	return codeHash
}
