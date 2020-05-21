package eth

import (
	"github.com/fsn-dev/crossChain-Bridge/common"
)

func (b *EthBridge) IsValidAddress(address string) bool {
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
