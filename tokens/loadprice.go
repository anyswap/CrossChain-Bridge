package tokens

import (
	"math/big"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/common/hexutil"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/tokens/eth/abicoder"
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

func (c *TokenPairConfig) loadTokenPrice() (err error) {
	if TokenPriceCfg == nil {
		return nil
	}

	var srcTokenPrice, dstTokenPrice float64
	srcChainID := GetCrossChainBridge(true).GetChainConfig().GetChainID()
	srcTokenAddress := c.SrcToken.ContractAddress
	dstChainID := GetCrossChainBridge(false).GetChainConfig().GetChainID()
	dstTokenAddress := c.DestToken.ContractAddress
	if srcChainID != nil {
		srcTokenPrice, _ = loadTokenPrice(srcChainID, srcTokenAddress)
	}
	if dstChainID != nil {
		dstTokenPrice, _ = loadTokenPrice(dstChainID, dstTokenAddress)
	}
	if srcTokenPrice == 0 && dstTokenPrice == 0 {
		return ErrMissTokenPrice
	}
	if srcTokenPrice == 0 {
		log.Info("srcTokenPrice is not config, use dstTokenPrice", "pairID", c.PairID)
		srcTokenPrice = dstTokenPrice
	}
	if dstTokenPrice == 0 {
		log.Info("dstTokenPrice is not config, use srcTokenPrice", "pairID", c.PairID)
		dstTokenPrice = srcTokenPrice
	}

	c.SrcToken.TokenPrice = srcTokenPrice
	c.DestToken.TokenPrice = dstTokenPrice

	log.Info("load token pair price success", "pairID", c.PairID,
		"srcTokenAddress", srcTokenAddress, "dstTokenAddress", dstTokenAddress,
		"srcTokenName", c.SrcToken.Name, "dstTokenName", c.DestToken.Name,
		"srcTokenPrice", c.SrcToken.TokenPrice, "dstTokenPrice", c.DestToken.TokenPrice,
	)
	return nil
}

func loadTokenPrice(chainID *big.Int, tokenAddress string) (float64, error) {
	// call `getTokenPrice(uint256 chainID, address tokenAddr)`
	data := make(hexutil.Bytes, 68)
	copy(data[:4], common.FromHex("0x87e320e4"))
	copy(data[4:36], common.LeftPadBytes(chainID.Bytes(), 32))
	copy(data[36:], common.HexToAddress(tokenAddress).Hash().Bytes())
	result, err := callContract(TokenPriceCfg.Contract, TokenPriceCfg.APIAddress, data, "latest")
	if err != nil {
		log.Error("load token price failed", "chainID", chainID, "token", tokenAddress, "err", err)
		return 0, err
	}
	str, err := abicoder.ParseStringInData(common.FromHex(result), 0)
	if err != nil {
		log.Error("parse token price failed", "chainID", chainID, "token", tokenAddress, "err", err)
		return 0, err
	}
	if str == "" {
		return 0, ErrMissTokenPrice
	}
	price, err := strconv.ParseFloat(str, 64)
	if err != nil {
		log.Error("parse token price as float failed", "price", str, "chainID", chainID, "token", tokenAddress, "err", err)
		return 0, err
	}
	log.Info("load token price success", "chainID", chainID, "token", tokenAddress, "price", price)
	return price, nil
}

func initAllTokenPrices() {
	if TokenPriceCfg == nil {
		return
	}
	for _, pairCfg := range GetTokenPairsConfig() {
		err := pairCfg.loadTokenPrice()
		if err != nil {
			log.Fatal("init token price failed", "pairID", pairCfg.PairID, "err", err)
		}
		log.Info("init token price success", "pairID", pairCfg.PairID)
		pairCfg.SrcToken.CalcAndStoreValue()
		pairCfg.DestToken.CalcAndStoreValue()
	}
	log.Info("init all token price success")
}

// reload specified pairIDs' token prices. if no pairIDs, then reload all.
func reloadTokenPrices(pairIDs []string) {
	if TokenPriceCfg == nil {
		return
	}
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
		oldSrcPrice := pairCfg.SrcToken.TokenPrice
		oldDstPrice := pairCfg.DestToken.TokenPrice
		err := pairCfg.loadTokenPrice()
		if err != nil {
			reloadAllSuccess = false
			log.Error("reload token price failed", "pairID", pairCfg.PairID, "err", err)
			continue
		}
		if pairCfg.SrcToken.TokenPrice != oldSrcPrice {
			pairCfg.SrcToken.CalcAndStoreValue()
		}
		if pairCfg.DestToken.TokenPrice != oldDstPrice {
			pairCfg.DestToken.CalcAndStoreValue()
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
