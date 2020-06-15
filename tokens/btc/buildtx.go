package btc

import (
	"errors"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/wallet/txauthor"
	"github.com/btcsuite/btcwallet/wallet/txrules"
	"github.com/btcsuite/btcwallet/wallet/txsizes"
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/params"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	"github.com/fsn-dev/crossChain-Bridge/tokens/btc/electrs"
)

// BuildRawTransaction build raw tx
func (b *Bridge) BuildRawTransaction(args *tokens.BuildTxArgs) (rawTx interface{}, err error) {
	var (
		token         = b.TokenConfig
		from          = args.From
		to            = args.To
		amount        = args.Value
		memo          = args.Memo
		relayFeePerKb btcutil.Amount
		changeAddress string
		txOuts        []*wire.TxOut
	)

	switch args.SwapType {
	case tokens.SwapinType:
		return nil, tokens.ErrSwapTypeNotSupported
	case tokens.SwapoutType, tokens.SwapRecallType:
		from = token.DcrmAddress                          // from
		amount = tokens.CalcSwappedValue(amount, b.IsSrc) // amount
	}

	if from == "" {
		return nil, errors.New("no sender specified")
	}

	var extra *tokens.BtcExtraArgs
	if args.Extra == nil || args.Extra.BtcExtra == nil {
		extra = &tokens.BtcExtraArgs{}
		args.Extra = &tokens.AllExtras{BtcExtra: extra}
	} else {
		extra = args.Extra.BtcExtra
	}

	if extra.ChangeAddress != nil {
		changeAddress = *extra.ChangeAddress
	} else {
		changeAddress = from
	}

	if extra.RelayFeePerKb != nil {
		relayFeePerKb = btcutil.Amount(*extra.RelayFeePerKb)
	} else {
		relayFeePerKb = btcutil.Amount(tokens.BtcRelayFeePerKb)
	}

	pkscript, err := b.getPayToAddrScript(to)
	if err != nil {
		return nil, err
	}
	txOuts = append(txOuts, wire.NewTxOut(amount.Int64(), pkscript))

	if memo != "" {
		nullScript, err := txscript.NullDataScript([]byte(memo))
		if err != nil {
			return nil, err
		}
		txOuts = append(txOuts, wire.NewTxOut(0, nullScript))
	}

	inputSource := func(target btcutil.Amount) (
		total btcutil.Amount, inputs []*wire.TxIn,
		inputValues []btcutil.Amount, scripts [][]byte, err error) {

		if len(extra.PreviousOutPoints) != 0 {
			return b.getUtxos(from, target, extra.PreviousOutPoints)
		}
		return b.selectUtxos(from, target)
	}

	changeSource := func() ([]byte, error) {
		return b.getPayToAddrScript(changeAddress)
	}

	authoredTx, err := NewUnsignedTransaction(txOuts, relayFeePerKb, inputSource, changeSource)
	if err != nil {
		return nil, err
	}

	if len(extra.PreviousOutPoints) == 0 {
		extra.PreviousOutPoints = make([]*tokens.BtcOutPoint, len(authoredTx.Tx.TxIn))
		for i, txin := range authoredTx.Tx.TxIn {
			point := txin.PreviousOutPoint
			extra.PreviousOutPoints[i] = &tokens.BtcOutPoint{
				Hash:  point.Hash.String(),
				Index: point.Index,
			}
		}
	}

	if args.SwapType != tokens.NoSwapType {
		args.Identifier = params.GetIdentifier()
	}

	return authoredTx, nil
}

func (b *Bridge) getPayToAddrScript(address string) ([]byte, error) {
	chainConfig := b.GetChainConfig()
	toAddr, err := btcutil.DecodeAddress(address, chainConfig)
	if err != nil {
		return nil, err
	}
	return txscript.PayToAddrScript(toAddr)
}

func (b *Bridge) selectUtxos(from string, target btcutil.Amount) (
	total btcutil.Amount, inputs []*wire.TxIn, inputValues []btcutil.Amount, scripts [][]byte, err error) {

	var utxos []*electrs.ElectUtxo
	for i := 0; i < retryCount; i++ {
		utxos, err = b.FindUtxos(from)
		if err == nil {
			break
		}
		time.Sleep(retryInterval)
	}
	if err != nil {
		return
	}

	p2pkhScript, err := b.getPayToAddrScript(from)
	if err != nil {
		return
	}

	var (
		tx      *electrs.ElectTx
		success bool
	)

	for _, utxo := range utxos {
		value := btcutil.Amount(*utxo.Value)
		if value <= 0 {
			continue
		}
		if value > btcutil.MaxSatoshi {
			continue
		}
		for i := 0; i < retryCount; i++ {
			tx, err = b.GetTransactionByHash(*utxo.Txid)
			if err == nil {
				break
			}
			time.Sleep(retryInterval)
		}
		if err != nil {
			continue
		}
		if *utxo.Vout >= uint32(len(tx.Vout)) {
			continue
		}
		output := tx.Vout[*utxo.Vout]
		if *output.ScriptpubkeyType != "p2pkh" {
			continue
		}
		if *output.ScriptpubkeyAddress != from {
			continue
		}
		txHash, err := chainhash.NewHashFromStr(*utxo.Txid)
		if err != nil {
			continue
		}
		preOut := wire.NewOutPoint(txHash, *utxo.Vout)
		txIn := wire.NewTxIn(preOut, p2pkhScript, nil)

		total += value
		inputs = append(inputs, txIn)
		inputValues = append(inputValues, value)
		scripts = append(scripts, p2pkhScript)

		if total >= target {
			success = true
			break
		}
	}

	if !success {
		err = fmt.Errorf("Not enough balance, total %v < target %v", total, target)
		return
	}

	return total, inputs, inputValues, scripts, nil
}

func (b *Bridge) getUtxos(from string, target btcutil.Amount, prevOutPoints []*tokens.BtcOutPoint) (
	total btcutil.Amount, inputs []*wire.TxIn, inputValues []btcutil.Amount, scripts [][]byte, err error) {

	p2pkhScript, err := b.getPayToAddrScript(from)
	if err != nil {
		return
	}
	var (
		tx       *electrs.ElectTx
		outspend *electrs.ElectOutspend
		txHash   *chainhash.Hash
		value    btcutil.Amount
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
			return
		}
		if *outspend.Spent {
			if outspend.Status != nil && outspend.Status.BlockHeight != nil {
				spentHeight := *outspend.Status.BlockHeight
				err = fmt.Errorf("out point (%v, %v) is spent at %v", point.Hash, point.Index, spentHeight)
			} else {
				err = fmt.Errorf("out point (%v, %v) is spent at txpool", point.Hash, point.Index)
			}
			return
		}
		for i := 0; i < retryCount; i++ {
			tx, err = b.GetTransactionByHash(point.Hash)
			if err == nil {
				break
			}
			time.Sleep(retryInterval)
		}
		if err != nil {
			return
		}
		if point.Index >= uint32(len(tx.Vout)) {
			err = fmt.Errorf("out point (%v, %v) index overflow", point.Hash, point.Index)
			return
		}
		output := tx.Vout[point.Index]
		if *output.ScriptpubkeyType != "p2pkh" {
			err = fmt.Errorf("out point (%v, %v) script pubkey type %v is not p2pkh", point.Hash, point.Index, *output.ScriptpubkeyType)
			return
		}
		if *output.ScriptpubkeyAddress != from {
			err = fmt.Errorf("out point (%v, %v) script pubkey address %v is not %v", point.Hash, point.Index, *output.ScriptpubkeyAddress, from)
			return
		}
		value = btcutil.Amount(*output.Value)
		if value == 0 {
			err = fmt.Errorf("out point (%v, %v) with zero value", point.Hash, point.Index)
			return
		}

		txHash, _ = chainhash.NewHashFromStr(point.Hash)
		prevOutPoint := wire.NewOutPoint(txHash, point.Index)
		txIn := wire.NewTxIn(prevOutPoint, p2pkhScript, nil)

		total += value
		inputs = append(inputs, txIn)
		inputValues = append(inputValues, value)
		scripts = append(scripts, p2pkhScript)
	}
	if total < target {
		err = fmt.Errorf("Not enough balance, total %v < target %v", total, target)
	}
	return
}

type insufficientFundsError struct{}

func (insufficientFundsError) InputSourceError() {}
func (insufficientFundsError) Error() string {
	return "insufficient funds available to construct transaction"
}

// NewUnsignedTransaction ref btcwallet
// ref. https://github.com/btcsuite/btcwallet/blob/b07494fc2d662fdda2b8a9db2a3eacde3e1ef347/wallet/txauthor/author.go
// we only modify it to support P2PKH change script (the origin only support P2WPKH change script)
func NewUnsignedTransaction(outputs []*wire.TxOut, relayFeePerKb btcutil.Amount,
	fetchInputs txauthor.InputSource, fetchChange txauthor.ChangeSource) (*txauthor.AuthoredTx, error) {

	targetAmount := txauthor.SumOutputValues(outputs)
	estimatedSize := txsizes.EstimateVirtualSize(0, 1, 0, outputs, true)
	targetFee := txrules.FeeForSerializeSize(relayFeePerKb, estimatedSize)

	for {
		inputAmount, inputs, inputValues, scripts, err := fetchInputs(targetAmount + targetFee)
		if err != nil {
			return nil, err
		}
		if inputAmount < targetAmount+targetFee {
			return nil, insufficientFundsError{}
		}

		// We count the types of inputs, which we'll use to estimate
		// the vsize of the transaction.
		var nested, p2wpkh, p2pkh int
		for _, pkScript := range scripts {
			switch {
			// If this is a p2sh output, we assume this is a
			// nested P2WKH.
			case txscript.IsPayToScriptHash(pkScript):
				nested++
			case txscript.IsPayToWitnessPubKeyHash(pkScript):
				p2wpkh++
			default:
				p2pkh++
			}
		}

		maxSignedSize := txsizes.EstimateVirtualSize(p2pkh, p2wpkh, nested, outputs, true)
		maxRequiredFee := txrules.FeeForSerializeSize(relayFeePerKb, maxSignedSize)
		if maxRequiredFee < btcutil.Amount(tokens.BtcMinRelayFee) {
			maxRequiredFee = btcutil.Amount(tokens.BtcMinRelayFee)
		}
		remainingAmount := inputAmount - targetAmount
		if remainingAmount < maxRequiredFee {
			targetFee = maxRequiredFee
			continue
		}

		unsignedTransaction := &wire.MsgTx{
			Version:  wire.TxVersion,
			TxIn:     inputs,
			TxOut:    outputs,
			LockTime: 0,
		}
		changeIndex := -1
		changeAmount := inputAmount - targetAmount - maxRequiredFee
		if changeAmount != 0 {
			changeScript, err := fetchChange()
			if err != nil {
				return nil, err
			}
			// commont this to support P2PKH change script
			//if len(changeScript) > txsizes.P2WPKHPkScriptSize {
			//	return nil, errors.New("fee estimation requires change " +
			//		"scripts no larger than P2WPKH output scripts")
			//}
			threshold := txrules.GetDustThreshold(len(changeScript), txrules.DefaultRelayFeePerKb)
			if changeAmount < threshold {
				log.Debug("get rid of dust change", "amount", changeAmount, "threshold", threshold, "scriptsize", len(changeScript))
			} else {
				change := wire.NewTxOut(int64(changeAmount), changeScript)
				l := len(outputs)
				unsignedTransaction.TxOut = append(outputs[:l:l], change)
				changeIndex = l
			}
		}

		return &txauthor.AuthoredTx{
			Tx:              unsignedTransaction,
			PrevScripts:     scripts,
			PrevInputValues: inputValues,
			TotalInput:      inputAmount,
			ChangeIndex:     changeIndex,
		}, nil
	}
}
