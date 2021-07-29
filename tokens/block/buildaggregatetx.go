package block

import (
	"fmt"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc/electrs"
	"github.com/btcsuite/btcwallet/wallet/txauthor"
)

// BuildAggregateTransaction build aggregate tx (spend p2sh utxo)
func (b *Bridge) BuildAggregateTransaction(relayFeePerKb int64, addrs []string, utxos []*electrs.ElectUtxo) (rawTx *txauthor.AuthoredTx, err error) {
	if len(addrs) != len(utxos) {
		return nil, fmt.Errorf("call BuildAggregateTransaction: count of addrs (%v) is not equal to count of utxos (%v)", len(addrs), len(utxos))
	}

	txOuts, err := b.getTxOutputs("", nil, tokens.AggregateMemo)
	if err != nil {
		return nil, err
	}

	inputSource := func(target btcAmountType) (total btcAmountType, inputs []*wireTxInType, inputValues []btcAmountType, scripts [][]byte, err error) {
		return b.getUtxosFromElectUtxos(target, addrs, utxos)
	}

	log.Info("", "cfgUtxoAggregateToAddress", cfgUtxoAggregateToAddress)
	changeSource := func() ([]byte, error) {
		return b.GetPayToAddrScript(cfgUtxoAggregateToAddress)
	}

	return b.NewUnsignedTransaction(txOuts, btcAmountType(relayFeePerKb), inputSource, changeSource, true)
}

// VerifyAggregateMsgHash verify aggregate msgHash
func (b *Bridge) VerifyAggregateMsgHash(msgHash []string, args *tokens.BuildTxArgs) error {
	if args == nil || args.Extra == nil || args.Extra.BtcExtra == nil || len(args.Extra.BtcExtra.PreviousOutPoints) == 0 {
		return fmt.Errorf("empty ltc extra")
	}
	if args.Extra.BtcExtra.RelayFeePerKb == nil {
		return fmt.Errorf("empty relay fee")
	}
	rawTx, err := b.rebuildAggregateTransaction(args.Extra.BtcExtra)
	if err != nil {
		return err
	}
	return b.VerifyMsgHash(rawTx, msgHash)
}

func (b *Bridge) rebuildAggregateTransaction(extra *tokens.BtcExtraArgs) (rawTx *txauthor.AuthoredTx, err error) {
	log.Info("rebuildAggregateTransaction", "extra", extra)
	addrs, utxos, err := b.getUtxosFromOutPoints(extra.PreviousOutPoints)
	if err != nil {
		return nil, err
	}
	return b.BuildAggregateTransaction(*extra.RelayFeePerKb, addrs, utxos)
}

func (b *Bridge) getUtxosFromElectUtxos(target btcAmountType, addrs []string, utxos []*electrs.ElectUtxo) (total btcAmountType, inputs []*wireTxInType, inputValues []btcAmountType, scripts [][]byte, err error) {
	if len(addrs) != len(utxos) {
		return btcAmountType(0), nil, nil, nil, fmt.Errorf("call BuildAggregateTransaction: count of addrs (%v) is not equal to count of utxos (%v)", len(addrs), len(utxos))
	}

	for i, utxo := range utxos {
		value := btcAmountType(*utxo.Value)
		if value == 0 {
			continue
		}

		address := addrs[i]
		pkScript, errt := b.GetPayToAddrScript(address)
		if errt != nil {
			continue
		}

		txIn, errf := b.NewTxIn(*utxo.Txid, *utxo.Vout, pkScript)
		if errf != nil {
			continue
		}

		total += value
		inputs = append(inputs, txIn)
		inputValues = append(inputValues, value)
		scripts = append(scripts, pkScript)
	}

	if total < target {
		log.Warnf("getUtxos total %v < target %v", total, target)
	}

	return total, inputs, inputValues, scripts, nil
}

func (b *Bridge) getUtxosFromOutPoints(prevOutPoints []*tokens.BtcOutPoint) (addrs []string, utxos []*electrs.ElectUtxo, err error) {
	var (
		tx       *electrs.ElectTx
		outspend *electrs.ElectOutspend
	)

	for _, point := range prevOutPoints {
		for i := 0; i < retryCount; i++ {
			outspend, err = b.GetOutspend(point.Hash, point.Index)
			if err == nil {
				break
			}
			time.Sleep(retryInterval)
		}
		if err != nil {
			return nil, nil, err
		}
		if *outspend.Spent {
			if outspend.Status != nil && outspend.Status.BlockHeight != nil {
				spentHeight := *outspend.Status.BlockHeight
				err = fmt.Errorf("out point (%v, %v) is spent at %v", point.Hash, point.Index, spentHeight)
			} else {
				err = fmt.Errorf("out point (%v, %v) is spent at txpool", point.Hash, point.Index)
			}
			return nil, nil, err
		}
		for i := 0; i < retryCount; i++ {
			tx, err = b.GetTransactionByHash(point.Hash)
			if err == nil {
				break
			}
			time.Sleep(retryInterval)
		}
		if err != nil {
			return nil, nil, err
		}
		if point.Index >= uint32(len(tx.Vout)) {
			err = fmt.Errorf("out point (%v, %v) index overflow", point.Hash, point.Index)
			return nil, nil, err
		}
		output := tx.Vout[point.Index]
		if *output.Value == 0 {
			err = fmt.Errorf("out point (%v, %v) with zero value", point.Hash, point.Index)
			return nil, nil, err
		}
		if output.ScriptpubkeyAddress == nil {
			continue
		}

		addrs = append(addrs, *output.ScriptpubkeyAddress)
		utxos = append(utxos, &electrs.ElectUtxo{
			Txid:  &point.Hash,
			Vout:  &point.Index,
			Value: output.Value,
		})
	}
	return addrs, utxos, nil
}
