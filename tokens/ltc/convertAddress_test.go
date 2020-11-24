package ltc

import (
	"testing"

	"github.com/anyswap/CrossChain-Bridge/tokens"
)

func TestConvertAddress(t *testing.T) {
	t.Logf("Test convert BTC address to LTC")

	b := NewCrossChainBridge(true)
	b.ChainConfig = &tokens.ChainConfig{
		BlockChain: "Bitcoin",
		NetID:      netMainnet,
	}

	btcaddr := "3KkxrUKBzSnb99rNvbwp6Gn934MaLokhbU"
	ltcaddr := "MRy7AMj9wZe1wf8H2Uw9uv2YMkx2QoPHCt"

	t.Logf("BTC address: %v\n", btcaddr)

	addr, err := b.ConvertBTCAddress(btcaddr, "Main")
	if err != nil {
		t.Fatalf(err.Error())
	}

	t.Logf("LTC address: %v\n", addr.String())

	if addr.String() != ltcaddr {
		t.Fatalf("wrong convert result, ltc address should be %v\n", ltcaddr)
	}
}
