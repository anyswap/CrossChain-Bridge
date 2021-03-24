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
	return fmt.Sprintf("solana-deposit-address-%s", depositAddress)
}

const SolanaSwapinPromiseType = "solana-eth-bindaddress"

func (p *SolanaSwapinPromise) Type() string {
	return SolanaSwapinPromiseType
}

func (p *SolanaSwapinPromise) Value() interface{} {
	return strings.ToLower(p.ETHBindAddress)
}
