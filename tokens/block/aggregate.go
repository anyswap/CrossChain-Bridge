package block

import (
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc/electrs"
)

const (
	redeemAggregateP2SHInputSize = 198
)

/*var (
	aggSumVal uint64
	aggAddrs  []string
	aggUtxos  []*electrs.ElectUtxo
	aggOffset int
)*/

// StartAggregateJob aggregate job
/*
func (b *Bridge) StartAggregateJob() {
	for loop := 1; ; loop++ {
		log.Info("[aggregate] start aggregate job", "loop", loop)
		b.doAggregateJob()
		log.Info("[aggregate] finish aggregate job", "loop", loop)
		time.Sleep(aggInterval)
	}
}
*/

/*
func (b *Bridge) doAggregateJob() {
	aggOffset = 0
	for {
		p2shAddrs, err := mongodb.FindP2shAddresses(aggOffset, utxoPageLimit)
		if err != nil {
			log.Error("[aggregate] FindP2shAddresses failed", "err", err, "offset", aggOffset, "limit", utxoPageLimit)
			time.Sleep(3 * time.Second)
			continue
		}
		for _, p2shAddr := range p2shAddrs {
			b.findUtxosAndAggregate(p2shAddr.P2shAddress)
		}
		if len(p2shAddrs) < utxoPageLimit {
			break
		}
		aggOffset += utxoPageLimit
	}
}
*/

/*
func (b *Bridge) findUtxosAndAggregate(addr string) {
	findUtxos, _ := b.FindUtxos(addr)
	for _, utxo := range findUtxos {
		if utxo.Value == nil || *utxo.Value == 0 {
			continue
		}
		if isUtxoExist(utxo) {
			continue
		}
		log.Info("[aggregate] find utxo", "address", addr, "utxo", utxo.String())

		aggSumVal += *utxo.Value
		aggAddrs = append(aggAddrs, addr)
		aggUtxos = append(aggUtxos, utxo)

		if shouldAggregate(len(aggUtxos), aggSumVal) {
			b.aggregate()
		}
	}
}
*/

/*
func isUtxoExist(utxo *electrs.ElectUtxo) bool {
	for _, item := range aggUtxos {
		if *item.Txid == *utxo.Txid && *item.Vout == *utxo.Vout {
			return true
		}
	}
	return false
}
*/

/*
func (b *Bridge) aggregate() {
	txHash, err := b.AggregateUtxos(aggAddrs, aggUtxos)
	if err != nil {
		log.Error("[aggregate] AggregateUtxos failed", "err", err)
	} else {
		log.Info("[aggregate] AggregateUtxos succeed", "txHash", txHash, "utxos", len(aggUtxos), "sumVal", aggSumVal)
	}
	aggSumVal = 0
	aggAddrs = nil
	aggUtxos = nil
}
*/

// ShouldAggregate should aggregate
func (b *Bridge) ShouldAggregate(aggUtxoCount int, aggSumVal uint64) bool {
	if aggUtxoCount >= cfgUtxoAggregateMinCount {
		return true
	}
	if aggSumVal >= cfgUtxoAggregateMinValue {
		return true
	}
	return false
}

// AggregateUtxos aggregate uxtos
func (b *Bridge) AggregateUtxos(addrs []string, utxos []*electrs.ElectUtxo) (string, error) {
	relayFee := b.getRelayFeePerKb()

	authoredTx, err := b.BuildAggregateTransaction(relayFee, addrs, utxos)
	if err != nil {
		return "", err
	}

	args := &tokens.BuildTxArgs{
		SwapInfo: tokens.SwapInfo{
			PairID:     PairID,
			Identifier: tokens.AggregateIdentifier,
		},
		Extra: &tokens.AllExtras{
			BtcExtra: &tokens.BtcExtraArgs{},
		},
	}

	extra := args.Extra.BtcExtra
	extra.RelayFeePerKb = &relayFee
	extra.PreviousOutPoints = make([]*tokens.BtcOutPoint, len(authoredTx.Tx.TxIn))
	for i, txin := range authoredTx.Tx.TxIn {
		point := txin.PreviousOutPoint
		extra.PreviousOutPoints[i] = &tokens.BtcOutPoint{
			Hash:  point.Hash.String(),
			Index: point.Index,
		}
	}

	var signedTx interface{}
	var txHash string
	tokenCfg := b.GetTokenConfig(PairID)
	if tokenCfg.GetDcrmAddressPrivateKey() != nil {
		signedTx, txHash, err = b.SignTransaction(authoredTx, PairID)
	} else {
		maxRetryDcrmSignCount := 5
		for i := 0; i < maxRetryDcrmSignCount; i++ {
			signedTx, txHash, err = b.DcrmSignTransaction(authoredTx, args.GetExtraArgs())
			if err == nil {
				break
			}
			log.Warn("[aggregate] retry dcrm sign", "count", i+1, "err", err)
			time.Sleep(time.Second)
		}
	}
	if err != nil {
		return "", err
	}
	_, err = b.SendTransaction(signedTx)
	if err != nil {
		return "", err
	}
	return txHash, nil
}
