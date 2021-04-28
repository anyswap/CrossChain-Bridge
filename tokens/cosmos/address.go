package cosmos

import (
	"encoding/hex"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

// IsValidAddress check address
func (b *Bridge) IsValidAddress(address string) bool {
	if _, err := sdk.AccAddressFromBech32(address); err != nil {
		return false
	}
	return true
}

func (b *Bridge) EqualAddress(address1, address2 string) bool {
	acc1, err1 := sdk.AccAddressFromBech32(address1)
	acc2, err2 := sdk.AccAddressFromBech32(address2)
	if err1 == nil && err2 == nil {
		return acc1.Equals(acc2)
	}
	return false
}

// PublicKeyToAddress returns cosmos public key address
func (b *Bridge) PublicKeyToAddress(pubKeyHex string) (address string, err error) {
	pubKeyHex = strings.TrimPrefix(pubKeyHex, "0x")
	bb, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return
	}
	pk, err := btcec.ParsePubKey(bb, btcec.S256())
	if err != nil {
		return
	}
	cpk := pk.SerializeCompressed()
	var pub [33]byte
	copy(pub[:], cpk[:33])
	pubkey := secp256k1.PubKeySecp256k1(pub)
	addr := pubkey.Address()
	accAddress, err := sdk.AccAddressFromHex(addr.String())
	if err != nil {
		return
	}
	address = accAddress.String()
	return
}
