package block

import (
	"github.com/anyswap/CrossChain-Bridge/tokens/btc"
)

// Bridge block bridge inherit from btc bridge
type Bridge struct {
	*btc.Bridge
}

// NewCrossChainBridge new fsn bridge
func NewCrossChainBridge(isSrc bool) *Bridge {
	instance := &Bridge{Bridge: btc.NewCrossChainBridge(isSrc)}
	btc.BridgeInstance = instance
	return instance
}
