package crypto

import (
	"fmt"
	"math/big"
)

// First byte is the network
// Second byte is the version
// Remaining bytes are the payload
type hash []byte

func NewRippleHash(s string) (Hash, error) {
	// Special case which will deal short addresses
	switch s {
	case "0":
		return newHashFromString(ACCOUNT_ZERO)
	case "1":
		return newHashFromString(ACCOUNT_ONE)
	default:
		return newHashFromString(s)
	}
}

// Checks hash matches expected version
func NewRippleHashCheck(s string, version HashVersion) (Hash, error) {
	hash, err := NewRippleHash(s)
	if err != nil {
		return nil, err
	}
	if hash.Version() != version {
		want := hashTypes[version].Description
		got := hashTypes[hash.Version()].Description
		return nil, fmt.Errorf("Bad version for: %s expected: %s got: %s ", s, want, got)
	}
	return hash, nil
}

func NewAccountId(b []byte) (Hash, error) {
	return newHash(b, RIPPLE_ACCOUNT_ID)
}

func NewAccountPublicKey(b []byte) (Hash, error) {
	return newHash(b, RIPPLE_ACCOUNT_PUBLIC)
}

func NewAccountPrivateKey(b []byte) (Hash, error) {
	return newHash(b, RIPPLE_ACCOUNT_PRIVATE)
}

func NewNodePublicKey(b []byte) (Hash, error) {
	return newHash(b, RIPPLE_NODE_PUBLIC)
}

func NewNodePrivateKey(b []byte) (Hash, error) {
	return newHash(b, RIPPLE_NODE_PRIVATE)
}

func NewFamilySeed(b []byte) (Hash, error) {
	return newHash(b, RIPPLE_FAMILY_SEED)
}

func AccountId(key Key, sequence *uint32) (Hash, error) {
	return NewAccountId(key.Id(sequence))
}

func AccountPublicKey(key Key, sequence *uint32) (Hash, error) {
	return NewAccountPublicKey(key.Public(sequence))
}

func AccountPrivateKey(key Key, sequence *uint32) (Hash, error) {
	return NewAccountPrivateKey(key.Private(sequence))
}

func NodePublicKey(key Key) (Hash, error) {
	return NewNodePublicKey(key.Public(nil))
}

func NodePrivateKey(key Key) (Hash, error) {
	return NewNodePrivateKey(key.Private(nil))
}

func GenerateFamilySeed(password string) (Hash, error) {
	return NewFamilySeed(Sha512Quarter([]byte(password)))
}

func newHash(b []byte, version HashVersion) (Hash, error) {
	n := hashTypes[version].Payload
	if len(b) > n {
		return nil, fmt.Errorf("Hash is wrong size, expected: %d got: %d", n, len(b))
	}
	return append(hash{byte(version)}, b...), nil
}

func newHashFromString(s string) (Hash, error) {
	decoded, err := Base58Decode(s, ALPHABET)
	if err != nil {
		return nil, err
	}
	return hash(decoded[:len(decoded)-4]), nil
}

func (h hash) String() string {
	b := append(hash{byte(h.Version())}, h.Payload()...)
	return Base58Encode(b, ALPHABET)
}

func (h hash) Version() HashVersion {
	return HashVersion(h[0])
}

func (h hash) Payload() []byte {
	return h[1:]
}

// Return a slice of the payload with leading zeroes omitted
func (h hash) PayloadTrimmed() []byte {
	payload := h.Payload()
	for i := range payload {
		if payload[i] != 0 {
			return payload[i:]
		}
	}
	return payload[len(payload)-1:]
}

func (h hash) Value() *big.Int {
	return big.NewInt(0).SetBytes(h.Payload())
}

func (h hash) MarshalText() ([]byte, error) {
	return []byte(h.String()), nil
}

func (h hash) Clone() Hash {
	c := make(hash, len(h))
	copy(c, h)
	return c
}
