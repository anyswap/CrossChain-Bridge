package fsn

import (
	. "github.com/fsn-dev/crossChain-Bridge/tokens"
)

type FsnBridge struct {
	CrossChainBridgeBase
	IsSrc bool
}

func NewCrossChainBridge(isSrc bool) CrossChainBridge {
	panic(ErrTodo)
	return &FsnBridge{
		IsSrc: isSrc,
	}
}
