package zol

import (
	"fmt"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"testing"
)

func TestDecodeAddress_SOL(t *testing.T) {
	b := NewCrossChainBridge(true)
	b.ChainConfig = &tokens.ChainConfig{
		BlockChain: "ZeroLimit",
		NetID:      "mainnet",
	}

	res, err := b.DecodeAddress("3DGNfnbTYUgJ8B3Vwq7U5cF8baXq9Tp9AC")
	fmt.Printf("%v, %v", res, err)
	if err != nil {
		panic("")
	}
}
