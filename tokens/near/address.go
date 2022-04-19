package near

// IsValidAddress check address
func (b *Bridge) IsValidAddress(address string) bool {
	return false
}

// EqualAddress equal address
func (b *Bridge) EqualAddress(address1, address2 string) bool {
	return true
}

// PubKeyFromStr get public key from hex string
func PubKeyFromStr(pubKeyHex string) (string, error) {
	return "", nil
}

// PubKeyFromBytes get public key from bytes
func PubKeyFromBytes(pubKeyBytes []byte) (string, error) {
	return "", nil
}

// PublicKeyToAddress returns cosmos public key address
func PublicKeyToAddress(pubKeyHex string) (string, error) {
	return "", nil
}
