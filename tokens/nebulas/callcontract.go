package nebulas

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/common"
)

// token types (should be all upper case)
const (
	ERC20TokenType = "NRC20"
)

// GetErc20TotalSupply get erc20 total supply of address
func (b *Bridge) GetErc20TotalSupply(contract string) (*big.Int, error) {
	result, err := b.CallContract(contract, "0", "totalSupply", "")
	if err != nil {
		return nil, err
	}
	var total string
	err = json.Unmarshal([]byte(result), &total)
	if err != nil {
		return nil, err
	}
	return common.GetBigIntFromStr(total)
}

// GetErc20Balance get erc20 balacne of address
func (b *Bridge) GetErc20Balance(contract, address string) (*big.Int, error) {
	result, err := b.CallContract(contract, "0", "balanceOf", address)
	if err != nil {
		return nil, err
	}
	var balance string
	err = json.Unmarshal([]byte(result), &balance)
	if err != nil {
		return nil, err
	}
	return common.GetBigIntFromStr(balance)
}

// GetErc20Decimals get erc20 decimals
func (b *Bridge) GetErc20Decimals(contract string) (uint8, error) {
	result, err := b.CallContract(contract, "0", "decimals", "")
	if err != nil {
		return 0, err
	}
	var decimals uint8
	err = json.Unmarshal([]byte(result), &decimals)
	if err != nil {
		return 0, err
	}
	return decimals, nil
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
