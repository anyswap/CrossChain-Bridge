package tokens

import (
	"os"
	"os/signal"
	"syscall"

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

	log.Info("load token price success", "token", c.ContractAddress, "name", c.Name, "price", c.TokenPrice)
	return nil
}

// reload specified pairIDs' token prices. if no pairIDs, then reload all.
func reloadTokenPrices(pairIDs []string) {
	var pairCfgs []*TokenPairConfig
	if len(pairIDs) == 0 {
		allPairsCfg := GetTokenPairsConfig()
		pairCfgs = make([]*TokenPairConfig, 0, len(allPairsCfg))
		for _, pairCfg := range allPairsCfg {
			pairCfgs = append(pairCfgs, pairCfg)
		}
	} else {
		pairCfgs = make([]*TokenPairConfig, 0, len(pairIDs))
		for _, pairID := range pairIDs {
			if pairCfg := GetTokenPairConfig(pairID); pairCfg != nil {
				pairCfgs = append(pairCfgs, pairCfg)
			}
		}
	}
	reloadAllSuccess := true
	for _, pairCfg := range pairCfgs {
		oldPrice := pairCfg.SrcToken.TokenPrice
		err := pairCfg.SrcToken.loadTokenPrice(true)
		if err == nil {
			if pairCfg.SrcToken.TokenPrice != oldPrice {
				pairCfg.SrcToken.CalcAndStoreValue()
			}
		} else {
			reloadAllSuccess = false
			log.Error("reload token price failed", "name", pairCfg.SrcToken.Name, "token", pairCfg.SrcToken.ContractAddress, "err", err)
		}

		oldPrice = pairCfg.DestToken.TokenPrice
		err = pairCfg.DestToken.loadTokenPrice(false)
		if err == nil {
			if pairCfg.DestToken.TokenPrice != oldPrice {
				pairCfg.DestToken.CalcAndStoreValue()
			}
		} else {
			reloadAllSuccess = false
			log.Error("reload token price failed", "name", pairCfg.SrcToken.Name, "token", pairCfg.DestToken.ContractAddress, "err", err)
		}
	}
	if reloadAllSuccess {
		if len(pairIDs) == 0 {
			log.Info("reload token price success", "pairIDs", "all")
		} else {
			log.Info("reload token price success", "pairIDs", pairIDs)
		}
	}
}

func watchAndReloadTokenPrices() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGUSR1)
	for {
		sig := <-signalChan
		log.Info("receive signal to reload token prices", "signal", sig)
		reloadTokenPrices(nil)
	}
}
