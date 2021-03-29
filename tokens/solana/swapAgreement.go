package solana

import (
	"strings"
)

/*
								| 								Solana2ETH							|				ETH2Solana									|
Swapin					|		Solana2ETHSwapinAgreement			|	ETH2SolanaSwapinAgreement	|
Swapout				|	Read bind address from ETH tx data		|	ETH2SolanaSwapoutAgreement	|
*/

const (
	Solana2ETHSwapinAgreementType    = "Solana2ETHSwapinAgreement"
	ETH2SolanaSwapinAgreementType    = "ETH2SolanaSwapinAgreement"
	ETH2SolanaSwapinoutAgreementType = "ETH2SolanaSwapinoutAgreement"
)

const (
	SolanaAddressPrefix = "solana-"
	ETHAddressPrefix    = "eth-"
)

type Solana2ETHSwapinAgreement struct {
	SolanaDepositAddress string
	ETHBindAddress       string
}

func (p *Solana2ETHSwapinAgreement) Key() string {
	depositAddress := strings.ToLower(p.SolanaDepositAddress)
	return SolanaAddressPrefix + depositAddress
}

func (p *Solana2ETHSwapinAgreement) Type() string {
	return Solana2ETHSwapinAgreementType
}

func (p *Solana2ETHSwapinAgreement) Value() interface{} {
	return strings.ToLower(p.ETHBindAddress)
}

type ETH2SolanaSwapinAgreement struct {
	ETHDepositAddress string
	SolanaBindAddress string
}

func (p *ETH2SolanaSwapinAgreement) Key() string {
	depositAddress := strings.ToLower(p.ETHDepositAddress)
	return ETHAddressPrefix + depositAddress
}

func (p *ETH2SolanaSwapinAgreement) Type() string {
	return ETH2SolanaSwapAgreementType
}

func (p *ETH2SolanaSwapinAgreement) Value() interface{} {
	return strings.ToLower(p.SolanaBindAddress)
}

type ETH2SolanaSwapoutAgreement struct {
	SolanaWithdrawAddress string
	ETHBindAddress        string
}

func (p *ETH2SolanaSwapoutAgreement) Key() string {
	withdrawAddress := strings.ToLower(p.SolanaWithdrawAddress)
	return SolanaAddressPrefix + withdrawAddress
}

func (p *ETH2SolanaSwapoutAgreement) Type() string {
	return ETH2SolanaSwapAgreementType
}

func (p *ETH2SolanaSwapoutAgreement) Value() interface{} {
	return strings.ToLower(p.ETHBindAddress)
}
