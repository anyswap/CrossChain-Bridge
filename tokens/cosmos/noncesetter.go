package cosmos

import (
	"strings"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/mongodb"
)

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

// SetNonce set nonce directly always increase
func (b *Bridge) SetNonce(pairID string, value uint64) {
	tokenCfg := b.GetTokenConfig(pairID)
	account := strings.ToLower(tokenCfg.DcrmAddress)
	if b.IsSrcEndpoint() {
		if b.SwapoutNonce[account] < value {
			b.SwapoutNonce[account] = value
			_ = mongodb.UpdateLatestSwapoutNonce(account, value)
		}
	} else {
		if b.SwapinNonce[account] < value {
			b.SwapinNonce[account] = value
			_ = mongodb.UpdateLatestSwapinNonce(account, value)
		}
	}
}

// InitNonces init nonces
func (b *Bridge) InitNonces(nonces map[string]uint64) {
	if b.IsSrcEndpoint() {
		b.SwapoutNonce = nonces
	} else {
		b.SwapinNonce = nonces
	}
	log.Info("init swap nonces finished", "isSwapin", !b.IsSrcEndpoint(), "nonces", nonces)
}
