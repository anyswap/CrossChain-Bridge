package eth

import (
	"math/big"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

const (
	aggPageLimit = 100
	aggInterval  = 10 * time.Minute
)

var aggOffset int

// StartAggregateJob aggregate job
func (b *Bridge) StartAggregateJob() {
	for loop := 1; ; loop++ {
		log.Info("[aggregate] start aggregate job", "loop", loop)
		b.doAggregateJob()
		log.Info("[aggregate] finish aggregate job", "loop", loop)
		time.Sleep(aggInterval)
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

	balance, err := b.getAggregateValue(regAddr.Bip32Adddress, tokenCfg)
	if err != nil {
		log.Warn("[aggregate] get aggregate value failed", "address", regAddr.Bip32Adddress, "err", err)
		return
	}

	if balance == nil || balance.Cmp(tokenCfg.GetAggregateMinValue()) < 0 {
		log.Debug("[aggregate] ignore small value", "address", regAddr.Bip32Adddress, "balance", balance, "threshold", tokenCfg.GetAggregateMinValue())
		return
	}

	extra := &tokens.EthExtraArgs{
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

func (b *Bridge) getAggregateValue(account string, tokenCfg *tokens.TokenConfig) (value *big.Int, err error) {
	if tokenCfg.ContractAddress != "" {
		return b.getErc20Balance(tokenCfg.ContractAddress, account)
	}
	value, err = b.getBalance(account)
	if err != nil {
		return nil, err
	}
	gasPrice, err := b.getGasPrice()
	if err != nil {
		return nil, err
	}
	gasFee := new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(defGasLimit))
	gasFee.Mul(gasFee, big.NewInt(2)) // double to allow slippage
	value.Sub(value, gasFee)
	return value, nil
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
