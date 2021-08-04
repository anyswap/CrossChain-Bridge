package types

import (
	"testing"

	"github.com/anyswap/CrossChain-Bridge/common"
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
		{ // eth type 2 (eip1559 dynamic fee tx)
			"0x02f891038221e2843b9aca00843b9aca10830109b194593cc1a399a65d3eaf8316da933745c4f5b9442980a40c4c4285ec7a95254a0d413d33ae00355127da335aa3388793f0eaa2bd39937f6b36dd0dc080a0694f25018b0a857e0a8fb5ca452d740bfd9870d56413cf5393d0a1cbac2fd10ea00c90354efed7947f617041b58741a2dc319ba6075f7f70ff8cb1eb15db8baa98",
			"0x78fe88e8dfbc3b62773121d6b73a21d0e7b798290cbea00eeee5f6a03f8292f1",
		},
	}
)

func TestTxHash(t *testing.T) {
	for _, test := range txHashTests {
		tx := new(Transaction)
		if err := tx.UnmarshalBinary(common.FromHex(test.rawtx)); err != nil {
			t.Errorf("rawtx: %s, tx unmarshal error: %v", test.rawtx, err)
		}
		hash := tx.Hash().Hex()
		if hash != test.want {
			t.Errorf("rawtx %s: hash mismatch, have %s, want %s", test.rawtx, hash, test.want)
		}
	}
}
