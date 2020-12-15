package btc

import (
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// BridgeInstance btc bridge instance
var BridgeInstance BridgeInterface

// BridgeInterface btc bridge interface
type BridgeInterface interface {
	tokens.CrossChainBridge

	GetCompressedPublicKey(fromPublicKey string, needVerify bool) (cPkData []byte, err error)
}
