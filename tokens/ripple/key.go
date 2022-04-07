package ripple

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/crypto"
)

const (
	// PubKeyBytesLenCompressed is compressed pubkey byte length
	PubKeyBytesLenCompressed = 33
	// PubKeyBytesLenUncompressed is uncompressed pubkey byte length
	PubKeyBytesLenUncompressed = 65
)

const (
	pubkeyCompressed byte = 0x2
)

// ImportKeyFromSeed converts seed to ripple key
func ImportKeyFromSeed(seed string, cryptoType string) (crypto.Key, error) {
	shash, err := crypto.NewRippleHashCheck(seed, crypto.RIPPLE_FAMILY_SEED)
	if err != nil {
		return nil, fmt.Errorf("invalid seed, %w", err)
	}
	switch cryptoType {
	case "ed25519":
		key, _ := crypto.NewEd25519Key(shash.Payload())
		return key, nil
	case "ecdsa":
		key, _ := crypto.NewECDSAKey(shash.Payload())
		return key, nil
	default:
		return nil, fmt.Errorf("invalid crypto type %v", cryptoType)
	}
}

// ImportPublicKey converts pubkey to ripple pubkey
func ImportPublicKey(pubkey []byte) crypto.Key {
	return &EcdsaPublic{pub: pubkey}
}

// EcdsaPublic struct ripple ecdsa pubkey key
type EcdsaPublic struct {
	pub []byte
}

// GetAddress gets address from ripple key
func GetAddress(k crypto.Key, sequence *uint32) string {
	prefix := []byte{0}
	address := crypto.Base58Encode(append(prefix, k.Id(sequence)...), crypto.ALPHABET)
	return address
}

// Id returns pubkey bytes from ripple key
func (k *EcdsaPublic) Id(sequence *uint32) []byte {
	return crypto.Sha256RipeMD160(k.Public(sequence))
}

// Private not used
func (k *EcdsaPublic) Private(sequence *uint32) []byte {
	return nil
}

// Public returns pubkey bytes
func (k *EcdsaPublic) Public(sequence *uint32) []byte {
	if len(k.pub) == PubKeyBytesLenCompressed {
		return k.pub
	}
	xs := hex.EncodeToString(k.pub[1:33])
	ys := hex.EncodeToString(k.pub[33:])
	x, _ := new(big.Int).SetString(xs, 16)
	y, _ := new(big.Int).SetString(ys, 16)
	b := make([]byte, 0, PubKeyBytesLenCompressed)
	format := pubkeyCompressed
	if isOdd(y) {
		format |= 0x1
	}
	b = append(b, format)
	return paddedAppend(32, b, x.Bytes())

}

func isOdd(a *big.Int) bool {
	return a.Bit(0) == 1
}

func paddedAppend(size uint, dst, src []byte) []byte {
	for i := 0; i < int(size)-len(src); i++ {
		dst = append(dst, 0)
	}
	return append(dst, src...)
}
