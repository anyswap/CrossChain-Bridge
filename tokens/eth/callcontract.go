package eth

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/common/hexutil"
)

// token types (should be all upper case)
const (
	ERC20TokenType = "ERC20"
)

// GetErc20TotalSupply get erc20 total supply of address
func (b *Bridge) GetErc20TotalSupply(contract string) (*big.Int, error) {
	data := make(hexutil.Bytes, 4)
	copy(data[:4], erc20CodeParts["totalSupply"])
	result, err := b.CallContract(contract, data, "latest")
	if err != nil {
		return nil, err
	}
	return common.GetBigIntFromStr(result)
}

// GetErc20Balance get erc20 balacne of address
func (b *Bridge) GetErc20Balance(contract, address string) (*big.Int, error) {
	data := make(hexutil.Bytes, 36)
	copy(data[:4], erc20CodeParts["balanceOf"])
	copy(data[4:], common.HexToAddress(address).Hash().Bytes())
	result, err := b.CallContract(contract, data, "latest")
	if err != nil {
		return nil, err
	}
	return common.GetBigIntFromStr(result)
}

// GetErc20Decimals get erc20 decimals
func (b *Bridge) GetErc20Decimals(contract string) (uint8, error) {
	data := make(hexutil.Bytes, 4)
	copy(data[:4], erc20CodeParts["decimals"])
	result, err := b.CallContract(contract, data, "latest")
	if err != nil {
		return 0, err
	}
	decimals, err := common.GetUint64FromStr(result)
	return uint8(decimals), err
}

// GetTokenBalance api
func (b *Bridge) GetTokenBalance(tokenType, tokenAddress, accountAddress string) (*big.Int, error) {
	switch strings.ToUpper(tokenType) {
	case ERC20TokenType:
		return b.GetErc20Balance(tokenAddress, accountAddress)
	default:
		return nil, fmt.Errorf("[%v] can not get token balance of token with type '%v'", b.ChainConfig.BlockChain, tokenType)
	}
}

// GetTokenSupply impl
func (b *Bridge) GetTokenSupply(tokenType, tokenAddress string) (*big.Int, error) {
	switch strings.ToUpper(tokenType) {
	case ERC20TokenType:
		return b.GetErc20TotalSupply(tokenAddress)
	default:
		return nil, fmt.Errorf("[%v] can not get token supply of token with type '%v'", b.ChainConfig.BlockChain, tokenType)
	}
}
