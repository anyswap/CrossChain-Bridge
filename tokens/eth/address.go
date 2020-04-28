package eth

import (
	"github.com/fsn-dev/crossChain-Bridge/common"
)

func (b *EthBridge) IsValidAddress(address string) bool {
	return common.IsHexAddress(address)
}
