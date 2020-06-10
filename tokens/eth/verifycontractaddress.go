package eth

import (
	"bytes"
	"fmt"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
)

var extendedCodeParts = map[string][]byte{
	// Extended interfaces
	"SwapinFuncHash":  tokens.SwapinFuncHash[:],
	"LogSwapinTopic":  common.FromHex(tokens.LogSwapinTopic),
	"SwapoutFuncHash": tokens.SwapoutFuncHash[:],
	"LogSwapoutTopic": common.FromHex(tokens.LogSwapoutTopic),
}

var erc20CodeParts = map[string][]byte{
	// Erc20 interfaces
	"name":         common.FromHex("0x06fdde03"),
	"symbol":       common.FromHex("0x95d89b41"),
	"decimals":     common.FromHex("0x313ce567"),
	"totalSupply":  common.FromHex("0x18160ddd"),
	"balanceOf":    common.FromHex("0x70a08231"),
	"transfer":     common.FromHex("0xa9059cbb"),
	"transferFrom": common.FromHex("0x23b872dd"),
	"approve":      common.FromHex("0x095ea7b3"),
	"allowance":    common.FromHex("0xdd62ed3e"),
	"LogTransfer":  common.FromHex("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
	"LogApproval":  common.FromHex("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"),
}

// VerifyContractCode verify contract code
func (b *Bridge) VerifyContractCode(contract string, codePartsSlice ...map[string][]byte) (err error) {
	var code []byte
	retryCount := 3
	for i := 0; i < retryCount; i++ {
		code, err = b.GetCode(contract)
		if err == nil {
			break
		}
		log.Warn("get contract code failed", "contract", contract, "err", err)
		time.Sleep(1 * time.Second)
	}
	for _, codeParts := range codePartsSlice {
		for key, part := range codeParts {
			if !bytes.Contains(code, part) {
				return fmt.Errorf("contract byte code miss '%v' bytes '%x'", key, part)
			}
		}
	}
	return nil
}

// VerifyErc20ContractAddress verify erc20 contract
func (b *Bridge) VerifyErc20ContractAddress(contract string) (err error) {
	return b.VerifyContractCode(contract, erc20CodeParts)
}

// VerifyMbtcContractAddress verify mbtc contract
func (b *Bridge) VerifyMbtcContractAddress(contract string) (err error) {
	return b.VerifyContractCode(contract, extendedCodeParts, erc20CodeParts)
}
