package tokens

import (
	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/common/hexutil"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
)

func callContract(contract string, urls []string, data hexutil.Bytes, blockNumber string) (string, error) {
	reqArgs := map[string]interface{}{
		"to":   contract,
		"data": data,
	}
	var result string
	var err error
	for _, url := range urls {
		err = client.RPCPost(&result, url, "eth_call", reqArgs, blockNumber)
		if err == nil {
			return result, nil
		}
	}
	return "", err
}

func (c *TokenConfig) loadTokenPrice(isSrc bool) (err error) {
	c.TokenPrice = 0
	if TokenPriceCfg == nil {
		return nil
	}
	chainID := GetCrossChainBridge(isSrc).GetChainConfig().GetChainID()
	if chainID == nil && isSrc {
		// if source token price is not set, then use dest token price
		chainID = GetCrossChainBridge(false).GetChainConfig().GetChainID()
	}
	if chainID == nil {
		return ErrMissTokenPrice
	}
	// call `getTokenPrice(uint256 chainID, address tokenAddr)`
	data := make(hexutil.Bytes, 68)
	copy(data[:4], "0x87e320e4")
	copy(data[4:36], common.LeftPadBytes(chainID.Bytes(), 32))
	copy(data[36:], common.HexToAddress(c.ContractAddress).Hash().Bytes())
	result, err := callContract(TokenPriceCfg.Contract, TokenPriceCfg.APIAddress, data, "latest")
	if err != nil {
		log.Error("load token price failed", "token", c.ContractAddress, "err", err)
		return err
	}
	biTokenPrice, err := common.GetBigIntFromStr(result)
	if err != nil {
		return err
	}
	c.TokenPrice = FromBits(biTokenPrice, 4)
	if c.TokenPrice == 0 {
		return ErrMissTokenPrice
	}

	// convert to token amount
	*c.MaximumSwap /= c.TokenPrice
	*c.MinimumSwap /= c.TokenPrice
	*c.BigValueThreshold /= c.TokenPrice
	*c.MaximumSwapFee /= c.TokenPrice
	*c.MinimumSwapFee /= c.TokenPrice

	log.Info("load token price and convert to token amount success",
		"token", c.ContractAddress,
		"price", c.TokenPrice,
		"maxSwap", *c.MaximumSwap,
		"minSwap", *c.MinimumSwap,
		"bigSwap", *c.BigValueThreshold,
		"maxSwapFee", *c.MaximumSwapFee,
		"minSwapFee", *c.MinimumSwapFee,
	)

	return nil
}
