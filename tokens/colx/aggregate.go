package colx

import (
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc/electrs"
)

const (
	redeemAggregateP2SHInputSize = 200
)

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
	relayFee, err := b.getRelayFeePerKb()
	if err != nil {
		return "", err
	}

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
		signedTx, txHash, err = b.DcrmSignTransaction(authoredTx, args.GetExtraArgs())
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
