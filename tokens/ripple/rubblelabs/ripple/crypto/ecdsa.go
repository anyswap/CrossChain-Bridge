package crypto

import (
	"crypto/rand"
	"encoding/binary"
	"math/big"

	"github.com/btcsuite/btcd/btcec"
)

var (
	order = btcec.S256().N
	zero  = big.NewInt(0)
	one   = big.NewInt(1)
)

type ecdsaKey struct {
	*btcec.PrivateKey
}

func newKey(seed []byte) *btcec.PrivateKey {
	inc := big.NewInt(0).SetBytes(seed)
	inc.Lsh(inc, 32)
	for key := big.NewInt(0); ; inc.Add(inc, one) {
		key.SetBytes(Sha512Half(inc.Bytes()))
		if key.Cmp(zero) > 0 && key.Cmp(order) < 0 {
			privKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), key.Bytes())
			return privKey
		}
	}
}

// If seed is nil, generate a random one
func NewECDSAKey(seed []byte) (*ecdsaKey, error) {
	if seed == nil {
		seed = make([]byte, 16)
		if _, err := rand.Read(seed); err != nil {
			return nil, err
		}
	}
	return &ecdsaKey{newKey(seed)}, nil
}

func (k *ecdsaKey) generateKey(sequence uint32) *btcec.PrivateKey {
	seed := make([]byte, btcec.PubKeyBytesLenCompressed+4)
	copy(seed, k.PubKey().SerializeCompressed())
	binary.BigEndian.PutUint32(seed[btcec.PubKeyBytesLenCompressed:], sequence)
	key := newKey(seed)
	key.D.Add(key.D, k.D).Mod(key.D, order)
	key.X, key.Y = key.ScalarBaseMult(key.D.Bytes())
	return key
}

func (k *ecdsaKey) Id(sequence *uint32) []byte {
	if sequence == nil {
		return Sha256RipeMD160(k.PubKey().SerializeCompressed())
	}
	return Sha256RipeMD160(k.Public(sequence))
}

func (k *ecdsaKey) Private(sequence *uint32) []byte {
	if sequence == nil {
		return k.D.Bytes()
	}
	return k.generateKey(*sequence).D.Bytes()
}

func (k *ecdsaKey) Public(sequence *uint32) []byte {
	if sequence == nil {
		return k.PubKey().SerializeCompressed()
	}
	return k.generateKey(*sequence).PubKey().SerializeCompressed()
}
