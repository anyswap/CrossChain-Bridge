package riskctrl

import (
	"fmt"
	"math/big"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/shopspring/decimal"
)

var (
	depositAddress  string
	withdrawAddress string

	tokenType       = "ERC20"
	srcTokenAddress string
	dstTokenAddress string

	srcDecimals uint8
	dstDecimals uint8

	initialDiffValue   decimal.Decimal
	maxAuditDiffValue  decimal.Decimal
	minWithdrawReserve decimal.Decimal

	retryInterval = time.Second
)

// Work start risk control work
func Work() {
	log.Info("start risk control work")
	client.InitHTTPClient()
	InitCrossChainBridge()
	InitEmailConfig()

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

	initialDiffValue = decimal.NewFromFloat(riskConfig.InitialDiffValue)
	maxAuditDiffValue = decimal.NewFromFloat(riskConfig.MaxAuditDiffValue)
	minWithdrawReserve = decimal.NewFromFloat(riskConfig.MinWithdrawReserve)

	log.Info(fmt.Sprintf(`------ start audit work ------
srcTokenAddress    = %v
dstTokenAddress    = %v
depositAddress     = %v
withdrawAddress    = %v
initialDiffValue   = %v
maxAuditDiffValue  = %v
minWithdrawReserve = %v
`, srcTokenAddress, dstTokenAddress, depositAddress, withdrawAddress, initialDiffValue, maxAuditDiffValue, minWithdrawReserve))

	for {
		auditOnce()
		time.Sleep(30 * time.Second)
	}
}

func auditOnce() {
	auditBalanceDeviation()
	auditReserveBalance()
	log.Info("audit finish one turn")
}

func auditBalanceDeviation() {
	srcLatest, _ := srcBridge.GetLatestBlockNumber()
	dstLatest, _ := dstBridge.GetLatestBlockNumber()
	log.Info("get latest block number success", "srcLatest", srcLatest, "dstLatest", dstLatest)

	depositBalance := getDepositBalance()
	withdrawBalance := getWithdrawBalance()
	totalSupply := getTotalSupply()

	fDepositBalance := decimal.NewFromFloat(tokens.FromBits(depositBalance, srcDecimals))
	fWithdrawBalance := decimal.NewFromFloat(tokens.FromBits(withdrawBalance, srcDecimals))
	fTotalBalance := fDepositBalance.Add(fWithdrawBalance)
	fTotalSupply := decimal.NewFromFloat(tokens.FromBits(totalSupply, dstDecimals))

	diffValue := fTotalBalance.Sub(fTotalSupply).Sub(initialDiffValue)
	absDiffValue := diffValue.Abs()

	isNormal := true
	subject := "[risk] normal balance and total supply."
	if absDiffValue.Cmp(maxAuditDiffValue) > 0 {
		isNormal = false
		if diffValue.Sign() > 0 {
			subject = "[risk] balance larger than total supply."
		} else {
			subject = "[risk] balance smaller than total supply."
		}
	} else {
		prevSendAuditTimestamp = 0 // reset frequency check
	}

	logFn := log.Info
	if !isNormal {
		logFn = log.Error
	}

	content := fmt.Sprintf(`%v

fDepositBalance   = %v
fWithdrawBalance  = %v
fTotalBalance     = %v
fTotalSupply      = %v
initialDiffValue  = %v
diffValue         = %v
maxAuditDiffValue = %v
`, subject, fDepositBalance, fWithdrawBalance, fTotalBalance, fTotalSupply, initialDiffValue, diffValue, maxAuditDiffValue)

	logFn(content)

	if isNormal {
		return
	}

	now := time.Now().Unix()
	datetime := time.Unix(now, 0).Format("2006-01-02 15:04:05")

	content += fmt.Sprintf(`
srcTokenAddress   = %v
dstTokenAddress   = %v
depositAddress    = %v
withdrawAddress   = %v
srcLatestBlock    = %v
dstLatestBlock    = %v
datetime          = %v
`, srcTokenAddress, dstTokenAddress, depositAddress, withdrawAddress, srcLatest, dstLatest, datetime)

	_ = sendAuditEmail(subject, content)
}

func auditReserveBalance() {
	srcLatest, _ := srcBridge.GetLatestBlockNumber()
	withdrawBalance := getWithdrawBalance()

	fWithdrawBalance := decimal.NewFromFloat(tokens.FromBits(withdrawBalance, srcDecimals))

	isNormal := true
	subject := "[risk] normal withdraw reserve."
	if fWithdrawBalance.Cmp(minWithdrawReserve) < 0 {
		isNormal = false
		subject = "[risk] too low withdraw reserve."
	} else {
		prevSendLowReserveTimestamp = 0 // reset frequency check
	}

	logFn := log.Info
	if !isNormal {
		logFn = log.Error
	}

	content := fmt.Sprintf(`%v

fWithdrawBalance   = %v
minWithdrawReserve = %v
`, subject, fWithdrawBalance, minWithdrawReserve)

	logFn(content)

	if isNormal {
		return
	}

	now := time.Now().Unix()
	datetime := time.Unix(now, 0).Format("2006-01-02 15:04:05")

	content += fmt.Sprintf(`
withdrawAddress    = %v
srcTokenAddress    = %v
srcLatestBlock     = %v
datetime           = %v
`, withdrawAddress, srcTokenAddress, srcLatest, datetime)

	_ = sendLowReserveEmail(subject, content)
}

func getDepositBalance() *big.Int {
	var (
		depositBalance *big.Int
		err            error
	)
	for {
		if srcTokenAddress != "" {
			depositBalance, err = srcBridge.GetTokenBalance(tokenType, srcTokenAddress, depositAddress)
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
	return depositBalance
}

func getWithdrawBalance() *big.Int {
	var (
		withdrawBalance *big.Int
		err             error
	)
	for {
		if srcTokenAddress != "" {
			withdrawBalance, err = srcBridge.GetTokenBalance(tokenType, srcTokenAddress, withdrawAddress)
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
	return withdrawBalance
}

func getTotalSupply() *big.Int {
	var (
		totalSupply *big.Int
		err         error
	)
	for {
		totalSupply, err = dstBridge.GetTokenSupply(tokenType, dstTokenAddress)
		if err == nil {
			log.Info("get total supply success", "token", dstTokenAddress, "totalSupply", totalSupply)
			break
		}
		log.Warn("get total supply failed", "token", dstTokenAddress, "err", err)
		time.Sleep(retryInterval)
	}
	return totalSupply
}
