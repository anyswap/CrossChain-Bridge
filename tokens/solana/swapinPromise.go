package solana

import (
	"fmt"
	"strings"
)

type SolanaSwapinPromise struct {
	SolanaDepositAddress string
	ETHBindAddress       string
}

func (p *SolanaSwapinPromise) Key() string {
	depositAddress := strings.ToLower(p.SolanaDepositAddress)
	return SolanaDepositAddressPrefix+depositAddress
}

const SolanaSwapinPromiseType = "solana-eth-bindaddress"

const SolanaDepositAddressPrefix = "solana-deposit-address-"

func (p *SolanaSwapinPromise) Type() string {
	return SolanaSwapinPromiseType
}

func (p *SolanaSwapinPromise) Value() interface{} {
	return strings.ToLower(p.ETHBindAddress)
}
