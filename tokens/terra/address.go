package terra

import (
	"encoding/hex"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// IsValidAddress check address
func (b *Bridge) IsValidAddress(address string) bool {
	_, err := sdk.AccAddressFromBech32(address)
	return err == nil
}

// EqualAddress equal address
func (b *Bridge) EqualAddress(address1, address2 string) bool {
	acc1, err1 := sdk.AccAddressFromBech32(address1)
	acc2, err2 := sdk.AccAddressFromBech32(address2)
	return err1 == nil && err2 == nil && acc1.Equals(acc2)
}

// PublicKeyToAddress returns cosmos public key address
func (b *Bridge) PublicKeyToAddress(pubKeyHex string) (string, error) {
	pubKeyHex = strings.TrimPrefix(pubKeyHex, "0x")
	bs, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return "", err
	}
	pk, err := btcec.ParsePubKey(bs, btcec.S256())
	if err != nil {
		return "", err
	}
	cpk := pk.SerializeCompressed()
	accAddress, err := sdk.AccAddressFromHex(hex.EncodeToString(cpk))
	if err != nil {
		return "", err
	}
	return accAddress.String(), nil
}
