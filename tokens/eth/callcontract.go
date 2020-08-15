package eth

import (
	"math/big"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/common/hexutil"
)

// GetErc20TotalSupply get erc20 total supply of address
func (b *Bridge) GetErc20TotalSupply(contract string) (*big.Int, error) {
	data := make(hexutil.Bytes, 4)
	copy(data[:4], erc20CodeParts["totalSupply"])
	result, err := b.CallContract(contract, data, "pending")
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
	result, err := b.CallContract(contract, data, "pending")
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
