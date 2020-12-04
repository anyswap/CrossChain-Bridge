package ltc

import (
	"testing"

	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var b *Bridge

// nolint:gochecknoinits // allow in testing
func init() {
	b = NewCrossChainBridge(true)
	b.ChainConfig = &tokens.ChainConfig{
		BlockChain: "Litecoin",
		NetID:      netMainnet,
	}
	b.GatewayConfig = &tokens.GatewayConfig{
		APIAddress: []string{"http://5.189.139.168:4110"},
	}
}

func TestConvertAddress(t *testing.T) {
	t.Logf("Test convert BTC address to LTC")

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

	addr1, err := b.ConvertLTCAddress(ltcaddr, "")
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("BTC address: %v\n", addr.String())

	if addr1.String() != btcaddr {
		t.Fatalf("wrong convert result, ltc address should be %v\n", btcaddr)
	}
}

func TestGetTransaction(t *testing.T) {
	t.Logf("Test get transaction")

	tx, err := b.GetTransactionByHash("26a21a82ba02303d9969309a0ce3517195d41b90e67822e3d4459ee1324f76d8")
	if err != nil {
		t.Fatalf(err.Error())
	}
	for _, vout := range tx.Vout {
		t.Logf("vout: %+v\n", *vout.ScriptpubkeyAddress)
	}
	t.Logf("tx: %+v\n", tx)
}

func TestGetPoolTransactions(t *testing.T) {
	t.Logf("Test get pool transactions")

	txs, err := b.GetPoolTransactions("LbNWG6KAWYu9zcusDibCmJL2EktB8Gcxp4")
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("txs: %+v\n", txs)
}
