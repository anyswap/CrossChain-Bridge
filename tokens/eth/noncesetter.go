package eth

import (
	"strings"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/mongodb"
)

// NonceSetterBase base nonce setter
type NonceSetterBase struct {
	SwapinNonce  map[string]uint64
	SwapoutNonce map[string]uint64
}

// NewNonceSetterBase new base nonce setter
func NewNonceSetterBase() *NonceSetterBase {
	return &NonceSetterBase{
		SwapinNonce:  make(map[string]uint64),
		SwapoutNonce: make(map[string]uint64),
	}
}

// SetNonce set nonce directly
func (b *Bridge) SetNonce(pairID string, value uint64) {
	tokenCfg := b.GetTokenConfig(pairID)
	account := strings.ToLower(tokenCfg.DcrmAddress)
	if b.IsSrcEndpoint() {
		b.SwapoutNonce[account] = value
	} else {
		b.SwapinNonce[account] = value
	}
}

// AdjustNonce adjust account nonce (eth like chain)
func (b *Bridge) AdjustNonce(pairID string, value uint64) (nonce uint64) {
	tokenCfg := b.GetTokenConfig(pairID)
	account := strings.ToLower(tokenCfg.DcrmAddress)
	nonce = value
	if b.IsSrcEndpoint() {
		if b.SwapoutNonce[account] > value {
			nonce = b.SwapoutNonce[account]
		} else {
			b.SwapoutNonce[account] = value
		}
	} else {
		if b.SwapinNonce[account] > value {
			nonce = b.SwapinNonce[account]
		} else {
			b.SwapinNonce[account] = value
		}
	}
	return nonce
}

// IncreaseNonce decrease account nonce (eth like chain)
func (b *Bridge) IncreaseNonce(pairID string, value uint64) {
	tokenCfg := b.GetTokenConfig(pairID)
	account := strings.ToLower(tokenCfg.DcrmAddress)
	if b.IsSrcEndpoint() {
		b.SwapoutNonce[account] += value
		_ = mongodb.UpdateLatestSwapNonce(account, false, b.SwapoutNonce[account])
	} else {
		b.SwapinNonce[account] += value
		_ = mongodb.UpdateLatestSwapNonce(account, true, b.SwapoutNonce[account])
	}
}

// InitNonces init nonces
func (b *Bridge) InitNonces(nonces map[string]uint64) {
	if b.IsSrcEndpoint() {
		b.SwapoutNonce = nonces
	} else {
		b.SwapinNonce = nonces
	}
	log.Info("init swap nonces finished", "isSrc", b.IsSrcEndpoint(), "nonces", nonces)
}
