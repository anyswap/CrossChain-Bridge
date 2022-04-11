package terra

import (
	"encoding/hex"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
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

// PubKeyFromStr get public key from hex string
func PubKeyFromStr(pubKeyHex string) (cryptotypes.PubKey, error) {
	pubKeyHex = strings.TrimPrefix(pubKeyHex, "0x")
	bs, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return nil, err
	}
	return PubKeyFromBytes(bs)
}

// PubKeyFromBytes get public key from bytes
func PubKeyFromBytes(pubKeyBytes []byte) (cryptotypes.PubKey, error) {
	cmp, err := btcec.ParsePubKey(pubKeyBytes, btcec.S256())
	if err != nil {
		return nil, err
	}

	compressedPublicKey := make([]byte, secp256k1.PubKeySize)
	copy(compressedPublicKey, cmp.SerializeCompressed())

	return &secp256k1.PubKey{Key: compressedPublicKey}, nil
}

// PublicKeyToAddress returns cosmos public key address
func PublicKeyToAddress(pubKeyHex string) (string, error) {
	pk, err := PubKeyFromStr(pubKeyHex)
	if err != nil {
		return "", err
	}
	accAddress, err := sdk.AccAddressFromHex(pk.Address().String())
	if err != nil {
		return "", err
	}
	return accAddress.String(), nil
}
