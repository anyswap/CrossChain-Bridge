package eth

import (
	"bytes"
	"fmt"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc"
	"github.com/anyswap/CrossChain-Bridge/types"
)

var (
	// ExtCodeParts extended func hashes and log topics
	ExtCodeParts map[string][]byte

	// first 4 bytes of `Keccak256Hash([]byte("Swapin(bytes32,address,uint256)"))`
	swapinFuncHash = common.FromHex("0xec126c77")
	logSwapinTopic = common.FromHex("0x05d0634fe981be85c22e2942a880821b70095d84e152c3ea3c17a4e4250d9d61")

	// first 4 bytes of `Keccak256Hash([]byte("Swapout(uint256,string)"))`
	mBTCSwapoutFuncHash = common.FromHex("0xad54056d")
	mBTCLogSwapoutTopic = common.FromHex("0x9c92ad817e5474d30a4378deface765150479363a897b0590fbb12ae9d89396b")

	// first 4 bytes of `Keccak256Hash([]byte("Swapout(uint256,address)"))`
	mETHSwapoutFuncHash = common.FromHex("0x628d6cba")
	mETHLogSwapoutTopic = common.FromHex("0x6b616089d04950dc06c45c6dd787d657980543f89651aec47924752c7d16c888")
)

var mBTCExtCodeParts = map[string][]byte{
	// Extended interfaces
	"SwapinFuncHash":  swapinFuncHash,
	"LogSwapinTopic":  logSwapinTopic,
	"SwapoutFuncHash": mBTCSwapoutFuncHash,
	"LogSwapoutTopic": mBTCLogSwapoutTopic,
}

var mETHExtCodeParts = map[string][]byte{
	// Extended interfaces
	"SwapinFuncHash":  swapinFuncHash,
	"LogSwapinTopic":  logSwapinTopic,
	"SwapoutFuncHash": mETHSwapoutFuncHash,
	"LogSwapoutTopic": mETHLogSwapoutTopic,
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

func (b *Bridge) getContractCode(contract string, onlyContractAddress bool) (code []byte, err error) {
	isSpecialCase := types.IsOkexChain(b.SignerChainID)
	retryCount := 3
	for i := 0; ; i++ {
		code, err = b.GetCode(contract)
		if err != nil {
			log.Warn("get contract code failed", "contract", contract, "err", err)
		} else if len(code) > 0 {
			break
		} else if onlyContractAddress {
			log.Warn("get contract code failed", "contract", contract, "err", "empty byte code")
		} else if i >= retryCount || !isSpecialCase {
			break
		}
		time.Sleep(3 * time.Second)
	}
	return code, err
}

// VerifyContractCode verify contract code
func (b *Bridge) VerifyContractCode(contract string, codePartsSlice ...map[string][]byte) (err error) {
	code, err := b.getContractCode(contract, true)
	if err != nil {
		return err
	}
	return VerifyContractCodeParts(code, codePartsSlice...)
}

// VerifyContractCodeParts verify contract code parts
func VerifyContractCodeParts(code []byte, codePartsSlice ...map[string][]byte) (err error) {
	for _, codeParts := range codePartsSlice {
		for key, part := range codeParts {
			if !bytes.Contains(code, part) {
				return fmt.Errorf("contract byte code miss '%v' bytes '%x'", key, part)
			}
		}
	}
	return nil
}

// VerifyErc20ContractCode verify erc20 contract code
func VerifyErc20ContractCode(code []byte) (err error) {
	return VerifyContractCodeParts(code, erc20CodeParts)
}

// VerifySwapContractCode verify swap contract code
func VerifySwapContractCode(code []byte) (err error) {
	return VerifyContractCodeParts(code, ExtCodeParts, erc20CodeParts)
}

// VerifyErc20ContractAddress verify erc20 contract
// For proxy contract delegating erc20 contract, verify its contract code hash
func (b *Bridge) VerifyErc20ContractAddress(contract, codeHash string, isProxy bool) (err error) {
	code, err := b.getContractCode(contract, true)
	if err != nil {
		return err
	}
	if !isProxy {
		err = VerifyErc20ContractCode(code)
		if err != nil {
			return err
		}
	} else if codeHash == "" {
		return fmt.Errorf("proxy contract of erc20 must specify code hash")
	}
	if codeHash != "" {
		calcedCodeHash := common.Keccak256Hash(code).String()
		if codeHash != calcedCodeHash {
			return fmt.Errorf("code hash mismatch. contract=%v, have=%v, want=%v", contract, codeHash, calcedCodeHash)
		}
	}
	return nil
}

// VerifyAnyswapContractAddress verify anyswap contract
func (b *Bridge) VerifyAnyswapContractAddress(contract string) (err error) {
	return b.VerifyContractCode(contract, ExtCodeParts)
}

// InitExtCodeParts init extended code parts
func InitExtCodeParts() {
	InitExtCodePartsWithFlag(isSwapoutToStringAddress())
}

// InitExtCodePartsWithFlag init extended code parts with flag
func InitExtCodePartsWithFlag(isMbtc bool) {
	switch {
	case isMbtc:
		ExtCodeParts = mBTCExtCodeParts
	default:
		ExtCodeParts = mETHExtCodeParts
	}
	log.Info("init extented code parts", "isMBTC", isMbtc)
}

func isSwapoutToStringAddress() bool {
	return params.IsSwapoutToStringAddress() || btc.BridgeInstance != nil
}

func getSwapinFuncHash() []byte {
	return ExtCodeParts["SwapinFuncHash"]
}

func getLogSwapoutTopic() (topTopic []byte, topicsLen int) {
	topTopic = ExtCodeParts["LogSwapoutTopic"]
	if isSwapoutToStringAddress() {
		topicsLen = 2
	} else {
		topicsLen = 3
	}
	return topTopic, topicsLen
}
