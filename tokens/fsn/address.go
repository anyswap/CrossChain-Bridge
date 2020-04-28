package fsn

import (
	"github.com/fsn-dev/crossChain-Bridge/common"
)

func (b *FsnBridge) IsValidAddress(address string) bool {
	return common.IsHexAddress(address)
}
