package eth

import (
	. "github.com/fsn-dev/crossChain-Bridge/tokens"
)

type EthBridge struct {
	CrossChainBridgeBase
	IsSrc bool
}

func NewCrossChainBridge(isSrc bool) CrossChainBridge {
	if isSrc {
		panic(ErrTodo)
	}
	return &EthBridge{
		IsSrc: isSrc,
	}
}
