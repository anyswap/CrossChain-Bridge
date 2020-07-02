package btc

import (
	"fmt"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc/electrs"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/wallet/txauthor"
)

// BuildAggregateTransaction build aggregate tx (spend p2sh utxo)
func (b *Bridge) BuildAggregateTransaction(addrs []string, utxos []*electrs.ElectUtxo) (rawTx *txauthor.AuthoredTx, err error) {
	if len(addrs) != len(utxos) {
		return nil, fmt.Errorf("call BuildAggregateTransaction: count of addrs (%v) is not equal to count of utxos (%v)", len(addrs), len(utxos))
	}

	memo := "aggregate"
	txOuts, err := b.getTxOutputs("", nil, memo)
	if err != nil {
		return nil, err
	}

	inputSource := func(target btcutil.Amount) (total btcutil.Amount, inputs []*wire.TxIn, inputValues []btcutil.Amount, scripts [][]byte, err error) {
		return b.getUtxosFromElectUtxos(target, addrs, utxos)
	}

	changeSource := func() ([]byte, error) {
		return b.getPayToAddrScript(b.TokenConfig.DcrmAddress)
	}

	relayFeePerKb := btcutil.Amount(tokens.BtcRelayFeePerKb + 2000)

	return NewUnsignedTransaction(txOuts, relayFeePerKb, inputSource, changeSource)
}

func (b *Bridge) rebuildAggregateTransaction(prevOutPoints []*tokens.BtcOutPoint) (rawTx *txauthor.AuthoredTx, err error) {
	addrs, utxos, err := b.getUtxosFromOutPoints(prevOutPoints)
	if err != nil {
		return nil, err
	}
	return b.BuildAggregateTransaction(addrs, utxos)
}

func (b *Bridge) getUtxosFromElectUtxos(target btcutil.Amount, addrs []string, utxos []*electrs.ElectUtxo) (total btcutil.Amount, inputs []*wire.TxIn, inputValues []btcutil.Amount, scripts [][]byte, err error) {
	var (
		txHash   *chainhash.Hash
		value    btcutil.Amount
		pkScript []byte
		p2shAddr string
		errt     error
	)

	for i, utxo := range utxos {
		value = btcutil.Amount(*utxo.Value)
		if value == 0 {
			continue
		}

		address := addrs[i]
		if b.IsP2shAddress(address) {
			bindAddr := tools.GetP2shBindAddress(address)
			if bindAddr == "" {
				continue
			}
			p2shAddr, _, _ = b.GetP2shAddress(bindAddr)
			if p2shAddr != address {
				log.Warn("wrong registered p2sh address", "have", address, "bind", bindAddr, "want", p2shAddr)
				continue
			}
		}

		pkScript, errt = b.getPayToAddrScript(address)
		if errt != nil {
			continue
		}

		txHash, _ = chainhash.NewHashFromStr(*utxo.Txid)
		prevOutPoint := wire.NewOutPoint(txHash, *utxo.Vout)
		txIn := wire.NewTxIn(prevOutPoint, pkScript, nil)

		total += value
		inputs = append(inputs, txIn)
		inputValues = append(inputValues, value)
		scripts = append(scripts, pkScript)
	}

	if total < target {
		log.Warn("getUtxos total %v < target %v", total, target)
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

		addrs = append(addrs, *output.ScriptpubkeyAddress)
		utxos = append(utxos, &electrs.ElectUtxo{
			Txid:  &point.Hash,
			Vout:  &point.Index,
			Value: output.Value,
		})
	}
	return addrs, utxos, nil
}
