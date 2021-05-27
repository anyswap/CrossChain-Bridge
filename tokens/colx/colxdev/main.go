package main

import (
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/common"
	colxutil "github.com/giangnamnabka/btcutil"
	"github.com/giangnamnabka/btcd/chaincfg"
	"github.com/giangnamnabka/btcd/btcec"
)

func main() {
	pubkey := "04d38309dfdfd9adf129287b68cf2e1f1124e0cbc40cc98f94e5f2d23c26712fa3b33d63280dd1448319a6a4f4111722d6b3a730ebe07652ed2b3770947b3de2e2"
	pkData := common.FromHex(pubkey)
	cPkData, _ := ToCompressedPublicKey(pkData)
	addr, _ := colxutil.NewAddressPubKeyHash(colxutil.Hash160(cPkData), &chaincfg.MainNetParams)
	fmt.Println(addr)
}

func ToCompressedPublicKey(pkData []byte) ([]byte, error) {
	pubKey, err := btcec.ParsePubKey(pkData, btcec.S256())
	if err != nil {
		return nil, err
	}
	return pubKey.SerializeCompressed(), nil
}