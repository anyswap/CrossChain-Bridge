package eth

// NonceSetterBase base nonce setter
type NonceSetterBase struct {
	SwapinNonce  uint64
	SwapoutNonce uint64
}

// NewNonceSetterBase new base nonce setter
func NewNonceSetterBase() *NonceSetterBase {
	return &NonceSetterBase{}
}

// SetNonce set nonce directly
func (b *Bridge) SetNonce(value uint64) {
	if b.IsSrcEndpoint() {
		b.SwapoutNonce = value
	} else {
		b.SwapinNonce = value
	}
}

// AdjustNonce adjust account nonce (eth like chain)
func (b *Bridge) AdjustNonce(value uint64) (nonce uint64) {
	nonce = value
	if b.IsSrcEndpoint() {
		if b.SwapoutNonce > value {
			nonce = b.SwapoutNonce
		} else {
			b.SwapoutNonce = value
		}
	} else {
		if b.SwapinNonce > value {
			nonce = b.SwapinNonce
		} else {
			b.SwapinNonce = value
		}
	}
	return nonce
}

// IncreaseNonce decrease account nonce (eth like chain)
func (b *Bridge) IncreaseNonce(value uint64) {
	if b.IsSrcEndpoint() {
		b.SwapoutNonce += value
	} else {
		b.SwapinNonce += value
	}
}
