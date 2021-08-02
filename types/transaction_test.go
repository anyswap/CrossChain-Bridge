package types

import (
	"testing"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/tools/rlp"
)

type txHashTest struct {
	rawtx string
	want  string
}

var (
	txHashTests = []txHashTest{
		{ // eth
			"0xf86e8231a0843b9aca64825208949873d61e6bf850d0b0c2f3c6e075980683f2d9fe87038d7ea4c6800080820136a093a2e1ead7f960623fb5d46c8605135d258f87e202df8ec5509567633618b367a05d236da2259cc584b2906493957d04ec00afa7d6e97568eac61281bb798b5302",
			"0xa8fb350068349a2593661deb21729ca32012cf59520fe4f0b8fd82cd8737a548",
		},
		{ // okex
			"0xf86c098405f5e100837a120094ce15aa76a07e109deb359da8a731df0d640066c287038d7ea4c680008081a8a0426b092ea37481fa4134de052a9f3830acfdaa03eb0419f650cea1f72a70931ca0060d9569961dfe86ed6179af15110f496cd7c4f34f716143b0bad22a1861ffb4",
			"0xc974c9d9d94e004aaa7af7a5f5f5670702c04073b2625987aa3577f6a3fbd281",
		},
	}
)

func TestTxHash(t *testing.T) {
	for _, test := range txHashTests {
		var tx Transaction
		if err := rlp.DecodeBytes(common.FromHex(test.rawtx), &tx); err != nil {
			t.Errorf("rawtx: %s, rlp decode error: %v", test.rawtx, err)
		}
		hash := tx.Hash().Hex()
		if hash != test.want {
			t.Errorf("rawtx %s: hash mismatch, have %s, want %s", test.rawtx, hash, test.want)
		}
	}
}
