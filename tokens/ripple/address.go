package ripple

import (
	"encoding/hex"
	"regexp"

	"github.com/anyswap/CrossChain-Bridge/log"
)

var rAddressReg = "^r[1-9a-km-zA-HJ-NP-Z]{32,33}$"

// IsValidAddress check address
func (b *Bridge) IsValidAddress(addr string) bool {
	match, err := regexp.MatchString(rAddressReg, addr)
	if err != nil {
		log.Warn("Error occurs when verify address", "error", err)
	}
	return match
}

// PublicKeyHexToAddress convert public key hex to ripple address
func PublicKeyHexToAddress(pubKeyHex string) (string, error) {
	pub, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return "", err
	}
	return PublicKeyToAddress(pub), nil
}

// PublicKeyToAddress converts pubkey to ripple address
func PublicKeyToAddress(pubkey []byte) string {
	return GetAddress(ImportPublicKey(pubkey), nil)
}
