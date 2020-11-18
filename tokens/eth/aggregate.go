package eth

import (
	"fmt"
	"math/big"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tools"
	"github.com/anyswap/CrossChain-Bridge/tools/keystore"
)

const (
	aggPageLimit                 = 100
	aggInterval                  = 10 * time.Minute
	maxAggPlusGasPricePercentage = uint64(3000)
)

var (
	aggOffset                 int
	aggMaxGasPrice            *big.Int
	aggPlusGasPricePercentage uint64
	aggKeyWrapper             *keystore.Key
)

// StartAggregateJob aggregate job
func (b *Bridge) StartAggregateJob() {
	if !b.IsSrc {
		log.Error("[aggregate] bridge is not on source endpoint, stop aggregate")
		return
	}
	if !tokens.IsBip32Used() {
		log.Info("[aggregate] bip32 is not used, stop aggregate")
		return
	}
	err := b.initAggregate()
	if err != nil {
		log.Error("[aggregate] init aggregate failed", "err", err)
		return
	}
	for loop := 1; ; loop++ {
		log.Info("[aggregate] start aggregate job", "loop", loop)
		b.doAggregateJob()
		log.Info("[aggregate] finish aggregate job", "loop", loop)
		time.Sleep(aggInterval)
	}
}

func (b *Bridge) initAggregate() error {
	aggMaxGasPrice = b.ChainConfig.GetAggregateMaxGasPrice()
	aggPlusGasPricePercentage = b.ChainConfig.AggPlusGasPricePercentage
	if aggPlusGasPricePercentage > maxAggPlusGasPricePercentage {
		return fmt.Errorf("config value of 'AggPlusGasPricePercentage' is too large")
	}
	key, err := tools.LoadKeyStore(b.ChainConfig.AggGasKeystoreFile, b.ChainConfig.AggGasPasswordFile)
	if err != nil {
		return err
	}
	aggKeyWrapper = key
	aggGasFrom := aggKeyWrapper.Address.String()
	log.Info("[aggregate] init aggregate success", "aggMaxGasPrice", aggMaxGasPrice, "aggGasFrom", aggGasFrom)
	return nil
}

func (b *Bridge) waitSatisfiedGasPrice() *big.Int {
	for {
		gasPrice, err := b.getGasPrice()
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}
		gasPrice.Mul(gasPrice, big.NewInt(int64(100+aggPlusGasPricePercentage)))
		gasPrice.Div(gasPrice, big.NewInt(100))
		if gasPrice.Cmp(aggMaxGasPrice) <= 0 {
			log.Info("[aggregate] gas price satisfy", "current", gasPrice, "max", aggMaxGasPrice)
			return gasPrice
		}
		log.Info("[aggregate] gas price does not satisfy", "current", gasPrice, "max", aggMaxGasPrice)
		time.Sleep(60 * time.Second)
	}
}

func (b *Bridge) doAggregateJob() {
	aggOffset = 0
	for {
		regAddrs, err := mongodb.FindRegisteredAddresses(aggOffset, aggPageLimit)
		if err != nil {
			log.Error("[aggregate] FindRegisteredAddresses failed", "err", err, "offset", aggOffset, "limit", aggPageLimit)
			time.Sleep(3 * time.Second)
			continue
		}
		for _, regAddr := range regAddrs {
			b.doAggregate(regAddr)
		}
		if len(regAddrs) < aggPageLimit {
			break
		}
		aggOffset += aggPageLimit
	}
}

func (b *Bridge) doAggregate(regAddr *mongodb.MgoRegisteredAddress) {
	if regAddr == nil || regAddr.RootPublicKey == "" || regAddr.Bip32Adddress == "" {
		return
	}
	if regAddr.Address == "" {
		log.Warn("[aggregate] bip32 address without bind", "bip32Address", regAddr.Bip32Adddress, "rootPubkey", regAddr.RootPublicKey)
		return
	}

	for _, pairCfg := range tokens.GetTokenPairsConfig() {
		b.aggregate(regAddr, pairCfg)
	}
}

func (b *Bridge) aggregate(regAddr *mongodb.MgoRegisteredAddress, pairCfg *tokens.TokenPairConfig) {
	pairID := pairCfg.PairID
	tokenCfg := b.GetTokenConfig(pairID)

	if regAddr.RootPublicKey != tokenCfg.DcrmPubkey {
		return
	}

	gasPrice := b.waitSatisfiedGasPrice()

	balance, err := b.getAggregateValue(regAddr.Bip32Adddress, gasPrice, tokenCfg)
	if err != nil {
		log.Warn("[aggregate] get aggregate value failed", "address", regAddr.Bip32Adddress, "err", err)
		return
	}

	if balance == nil || balance.Cmp(tokenCfg.GetAggregateMinValue()) < 0 {
		log.Debug("[aggregate] ignore small value", "address", regAddr.Bip32Adddress, "balance", balance, "threshold", tokenCfg.GetAggregateMinValue())
		return
	}

	extra := &tokens.EthExtraArgs{
		GasPrice:       gasPrice,
		AggregateValue: balance,
	}

	args := &tokens.BuildTxArgs{
		From: regAddr.Bip32Adddress,
		SwapInfo: tokens.SwapInfo{
			Identifier: tokens.AggregateIdentifier,
			PairID:     pairID,
			Bind:       regAddr.Address,
		},
		Extra: &tokens.AllExtras{
			EthExtra: extra,
		},
	}
	rawTx, err := b.BuildAggregateTransaction(args)
	if err != nil {
		log.Warn("[aggregate] build tx failed", "err", err)
		return
	}
	b.signAndSendAggregateTx(rawTx, args, tokenCfg)
}

func (b *Bridge) getAggregateValue(account string, gasPrice *big.Int, tokenCfg *tokens.TokenConfig) (value *big.Int, err error) {
	value, err = b.getBalance(account)
	if err != nil {
		return nil, err
	}
	gasFee := new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(defGasLimit))
	if tokenCfg.ContractAddress == "" {
		value.Sub(value, gasFee)
		return value, nil
	}
	if value.Cmp(gasFee) < 0 {
		diffGasFee := new(big.Int).Sub(gasFee, value)
		err = b.prepareAggregateGasFee(account, diffGasFee, gasPrice)
		if err != nil {
			return nil, err
		}
	}
	return b.getErc20Balance(tokenCfg.ContractAddress, account)
}

func (b *Bridge) signAndSendAggregateTx(rawTx interface{}, args *tokens.BuildTxArgs, tokenCfg *tokens.TokenConfig) {
	var (
		signedTx interface{}
		txHash   string
		err      error
	)

	if tokenCfg.GetDcrmAddressPrivateKey() != nil {
		signedTx, txHash, err = b.SignTransaction(rawTx, args.PairID)
	} else {
		maxRetryDcrmSignCount := 5
		for i := 0; i < maxRetryDcrmSignCount; i++ {
			signedTx, txHash, err = b.DcrmSignTransaction(rawTx, args.GetExtraArgs())
			if err == nil {
				break
			}
			log.Warn("[aggregate] retry dcrm sign", "count", i+1, "err", err)
			time.Sleep(time.Second)
		}
	}
	if err != nil {
		log.Error("[aggregate] sign tx failed", "err", err)
		return
	}
	log.Info("[aggregate] sign tx success", "txHash", txHash)

	_, err = b.SendTransaction(signedTx)
	if err != nil {
		log.Info("[aggregate] send tx failed", "err", err)
		return
	}
	log.Info("[aggregate] send tx success", "txHash", txHash)
}

func (b *Bridge) prepareAggregateGasFee(account string, value, gasPrice *big.Int) error {
	args := &tokens.BuildTxArgs{
		To:    account,
		Value: value,
		Extra: &tokens.AllExtras{
			EthExtra: &tokens.EthExtraArgs{
				GasPrice: gasPrice,
			},
		},
	}
	rawTx, err := b.BuildRawTransaction(args)
	if err != nil {
		log.Warn("[aggregate] prepare gas fee build tx failed", "err", err)
		return err
	}
	signedTx, txHash, err := b.SignTransactionWithPrivateKey(rawTx, aggKeyWrapper.PrivateKey)
	if err != nil {
		log.Warn("[aggregate] prepare gas fee sign tx failed", "err", err)
		return err
	}
	_, err = b.SendTransaction(signedTx)
	if err != nil {
		log.Warn("[aggregate] prepare gas fee send tx failed", "err", err)
		return err
	}
	log.Info("[aggregate] prepare gas fee send tx", "account", account, "value", value, "txHash", txHash)
	checkLoops := 20
	for i := 0; i < checkLoops; i++ {
		txr, err := b.GetTransactionReceipt(txHash)
		if err == nil {
			log.Info("[aggregate] prepare gas fee send tx success",
				"account", account, "value", value, "txHash", txHash,
				"blockNumber", txr.BlockNumber, "blockHash", txr.BlockHash)
			return nil
		}
		time.Sleep(3 * time.Second)
	}
	return nil
}
