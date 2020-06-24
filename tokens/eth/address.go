package eth

import (
	"github.com/anyswap/CrossChain-Bridge/common"
)

// IsValidAddress check address
func (b *Bridge) IsValidAddress(address string) bool {
	if !common.IsHexAddress(address) {
		return false
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
