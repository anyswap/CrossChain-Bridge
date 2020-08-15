package riskctrl

import (
	"math/big"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var (
	depositAddress  string
	withdrawAddress string

	srcTokenAddress string
	dstTokenAddress string

	srcDecimals uint8
	dstDecimals uint8

	initialDiffValue  float64
	maxAuditDiffValue float64
)

// Work start risk control work
func Work() {
	log.Info("start risk control work")
	client.InitHTTPClient()
	InitCrossChainBridge()

	exitCh := make(chan struct{})

	go audit()

	<-exitCh
}

func audit() {
	config := GetConfig()

	depositAddress = config.SrcToken.DepositAddress
	withdrawAddress = config.SrcToken.DcrmAddress

	srcTokenAddress = config.SrcToken.ContractAddress
	dstTokenAddress = config.DestToken.ContractAddress

	srcDecimals = *config.SrcToken.Decimals
	dstDecimals = *config.DestToken.Decimals

	initialDiffValue = riskConfig.InitialDiffValue
	maxAuditDiffValue = riskConfig.MaxAuditDiffValue

	log.Info("start audit work",
		"depositAddress", depositAddress,
		"withdrawAddress", withdrawAddress,
		"srcTokenAddress", srcTokenAddress,
		"dstTokenAddress", dstTokenAddress,
		"initialDiffValue", initialDiffValue,
		"maxAuditDiffValue", maxAuditDiffValue,
	)

	for {
		auditOnce()
		time.Sleep(30 * time.Second)
	}
}

func auditOnce() {
	var (
		depositBalance  *big.Int
		withdrawBalance *big.Int
		totalBalance    *big.Int
		totalSupply     *big.Int
		err             error

		retryInterval = time.Second
	)

	for {
		if srcTokenAddress != "" {
			depositBalance, err = srcBridge.GetErc20Balance(srcTokenAddress, depositAddress)
		} else {
			depositBalance, err = srcBridge.GetBalance(depositAddress)
		}
		if err == nil {
			log.Info("get deposit address balance success", "token", srcTokenAddress, "depositAddress", depositAddress, "depositBalance", depositBalance)
			break
		}
		log.Warn("get deposit address balance failed", "token", srcTokenAddress, "depositAddress", depositAddress, "err", err)
		time.Sleep(retryInterval)
	}

	for {
		if srcTokenAddress != "" {
			withdrawBalance, err = srcBridge.GetErc20Balance(srcTokenAddress, withdrawAddress)
		} else {
			withdrawBalance, err = srcBridge.GetBalance(withdrawAddress)
		}
		if err == nil {
			log.Info("get withdraw address balance success", "token", srcTokenAddress, "withdrawAddress", withdrawAddress, "withdrawBalance", withdrawBalance)
			break
		}
		log.Warn("get withdraw address balance failed", "token", srcTokenAddress, "withdrawAddress", withdrawAddress, "err", err)
		time.Sleep(retryInterval)
	}

	totalBalance = new(big.Int).Add(depositBalance, withdrawBalance)

	for {
		totalSupply, err = dstBridge.GetErc20TotalSupply(dstTokenAddress)
		if err == nil {
			log.Info("get total supply success", "token", dstTokenAddress, "totalSupply", totalSupply)
			break
		}
		log.Warn("get total supply failed", "token", dstTokenAddress, "err", err)
		time.Sleep(retryInterval)
	}

	fTotalBalance := tokens.FromBits(totalBalance, srcDecimals)
	fTotalSupply := tokens.FromBits(totalSupply, dstDecimals)

	diffValue := fTotalBalance - fTotalSupply
	diffValue -= initialDiffValue

	switch {
	case diffValue > maxAuditDiffValue:
		log.Error("[risk] balance larger than total supply", "totalBalance", fTotalBalance, "totalSupply", fTotalSupply, "diffValue", diffValue, "initialDiffValue", initialDiffValue)
	case diffValue < -maxAuditDiffValue:
		log.Error("[risk] balance smaller than total supply", "totalBalance", fTotalBalance, "totalSupply", fTotalSupply, "diffValue", -diffValue, "initialDiffValue", initialDiffValue)
	default:
		log.Info("[risk] normal balance and total supply", "totalBalance", fTotalBalance, "totalSupply", fTotalSupply, "diffValue", diffValue, "initialDiffValue", initialDiffValue)
	}
}
