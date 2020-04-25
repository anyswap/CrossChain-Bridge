package btc

import (
	. "github.com/fsn-dev/crossChain-Bridge/tokens"
)

type BtcBridge struct {
	CrossChainBridgeBase
	IsSrc bool
}

func NewCrossChainBridge(isSrc bool) CrossChainBridge {
	if !isSrc {
		panic(ErrBridgeDestinationNotSupported)
	}
	return &BtcBridge{
		IsSrc: isSrc,
	}
}
