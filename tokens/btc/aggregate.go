package btc

import (
	"errors"

	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc/electrs"
)

const (
	// AggregateIdentifier used in accepting
	AggregateIdentifier = "aggregate"

	aggregateMemo = "aggregate"
)

// AggregateUtxos aggregate uxtos
func (b *Bridge) AggregateUtxos(addrs []string, utxos []*electrs.ElectUtxo) (string, error) {
	authoredTx, err := b.BuildAggregateTransaction(addrs, utxos)
	if err != nil {
		return "", err
	}

	args := &tokens.BuildTxArgs{
		Extra: &tokens.AllExtras{
			BtcExtra: &tokens.BtcExtraArgs{},
		},
	}

	args.Identifier = AggregateIdentifier
	extra := args.Extra.BtcExtra
	extra.PreviousOutPoints = make([]*tokens.BtcOutPoint, len(authoredTx.Tx.TxIn))
	for i, txin := range authoredTx.Tx.TxIn {
		point := txin.PreviousOutPoint
		extra.PreviousOutPoints[i] = &tokens.BtcOutPoint{
			Hash:  point.Hash.String(),
			Index: point.Index,
		}
	}

	signedTx, txHash, err := b.DcrmSignTransaction(authoredTx, args)
	if err != nil {
		return "", err
	}
	_, err = b.SendTransaction(signedTx)
	if err != nil {
		return "", err
	}
	return txHash, nil
}

// VerifyAggregateMsgHash verify aggregate msgHash
func (b *Bridge) VerifyAggregateMsgHash(msgHash []string, args *tokens.BuildTxArgs) error {
	if args == nil || args.Extra == nil || args.Extra.BtcExtra == nil || len(args.Extra.BtcExtra.PreviousOutPoints) == 0 {
		return errors.New("empty btc extra")
	}
	rawTx, err := b.rebuildAggregateTransaction(args.Extra.BtcExtra.PreviousOutPoints)
	if err != nil {
		return err
	}
	return b.VerifyMsgHash(rawTx, msgHash)
}
