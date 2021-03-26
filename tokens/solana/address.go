package solana

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/dfuse-io/solana-go"
)

// IsValidAddress check address
func (b *Bridge) IsValidAddress(address string) bool {
	_, err := solana.PublicKeyFromBase58(address)
	return (err == nil)
}

// PublicKeyToAddress returns cosmos public key address
func (b *Bridge) PublicKeyToAddress(pubKeyHex string) (address string, err error) {
	bz, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return "", errors.New("Decode pubkey hex error")
	}
	pub := PublicKeyFromBytes(bz)
	return fmt.Sprintf("%s", pub), nil
}

func PublicKeyFromBytes(in []byte) (out solana.PublicKey) {
	byteCount := len(in)
	if byteCount == 0 {
		return
	}

	max := 32
	if byteCount < max {
		max = byteCount
	}

	copy(out[:], in[0:max])
	return
}
