package block

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc/electrs"
	"github.com/btcsuite/btcwallet/wallet/txauthor"
	"github.com/btcsuite/btcwallet/wallet/txrules"
	"github.com/btcsuite/btcwallet/wallet/txsizes"
)

const (
	p2pkhType    = "p2pkh"
	p2shType     = "p2sh"
	opReturnType = "op_return"

	retryCount    = 3
	retryInterval = 3 * time.Second
)

func (b *Bridge) getRelayFeePerKb() (estimateFee int64) {
	estimateFee = cfgMinRelayFee
	if cfgPlusFeePercentage > 0 {
		estimateFee += estimateFee * int64(cfgPlusFeePercentage) / 100
	}
	if estimateFee > cfgMaxRelayFeePerKb {
		estimateFee = cfgMaxRelayFeePerKb
	} else if estimateFee < cfgMinRelayFeePerKb {
		estimateFee = cfgMinRelayFeePerKb
	}
	return estimateFee
}

// BuildRawTransaction build raw tx
func (b *Bridge) BuildRawTransaction(args *tokens.BuildTxArgs) (rawTx interface{}, err error) {
	var (
		pairID        = args.PairID
		token         = b.GetTokenConfig(pairID)
		from          string
		to            string
		changeAddress string
		amount        *big.Int
		memo          string
		relayFeePerKb btcAmountType
	)

	if token == nil {
		return nil, fmt.Errorf("swap pair '%v' is not configed", pairID)
	}

	switch args.SwapType {
	case tokens.SwapinType:
		return nil, tokens.ErrSwapTypeNotSupported
	case tokens.SwapoutType:
		from = token.DcrmAddress                                          // from
		to = args.Bind                                                    // to
		changeAddress = token.DcrmAddress                                 // change
		amount = tokens.CalcSwappedValue(pairID, args.OriginValue, false) // amount
		memo = tokens.UnlockMemoPrefix + args.SwapID
	default:
		return nil, tokens.ErrUnknownSwapType
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
		if extra.ChangeAddress != nil && args.SwapType == tokens.NoSwapType {
			changeAddress = *extra.ChangeAddress
		}
	}

	if extra.RelayFeePerKb != nil {
		relayFeePerKb = btcAmountType(*extra.RelayFeePerKb)
	} else {
		relayFee := b.getRelayFeePerKb()
		extra.RelayFeePerKb = &relayFee
		relayFeePerKb = btcAmountType(relayFee)
	}

	txOuts, err := b.getTxOutputs(to, amount, memo)
	if err != nil {
		return nil, err
	}

	inputSource := func(target btcAmountType) (total btcAmountType, inputs []*wireTxInType, inputValues []btcAmountType, scripts [][]byte, err error) {
		if len(extra.PreviousOutPoints) != 0 {
			return b.getUtxos(from, target, extra.PreviousOutPoints)
		}
		return b.selectUtxos(from, target)
	}

	changeSource := func() ([]byte, error) {
		return b.GetPayToAddrScript(changeAddress)
	}

	authoredTx, err := b.NewUnsignedTransaction(txOuts, relayFeePerKb, inputSource, changeSource, false)
	if err != nil {
		return nil, err
	}

	updateExtraInfo(extra, authoredTx.Tx.TxIn)

	if args.SwapType != tokens.NoSwapType {
		args.Identifier = params.GetIdentifier()
	}

	return authoredTx, nil
}

func updateExtraInfo(extra *tokens.BtcExtraArgs, txins []*wireTxInType) {
	if len(extra.PreviousOutPoints) > 0 {
		return
	}
	extra.PreviousOutPoints = make([]*tokens.BtcOutPoint, len(txins))
	for i, txin := range txins {
		point := txin.PreviousOutPoint
		extra.PreviousOutPoints[i] = &tokens.BtcOutPoint{
			Hash:  point.Hash.String(),
			Index: point.Index,
		}
	}
}

// BuildTransaction build tx
func (b *Bridge) BuildTransaction(from string, receivers []string, amounts []int64, memo string, relayFeePerKb int64) (rawTx interface{}, err error) {
	if len(receivers) != len(amounts) {
		return nil, fmt.Errorf("count of receivers and amounts are not equal")
	}

	var txOuts []*wireTxOutType

	for i, receiver := range receivers {
		err = b.addPayToAddrOutput(&txOuts, receiver, amounts[i])
		if err != nil {
			return nil, err
		}
	}

	err = b.addMemoOutput(&txOuts, memo)
	if err != nil {
		return nil, err
	}

	inputSource := func(target btcAmountType) (total btcAmountType, inputs []*wireTxInType, inputValues []btcAmountType, scripts [][]byte, err error) {
		return b.selectUtxos(from, target)
	}

	changeSource := func() ([]byte, error) {
		return b.GetPayToAddrScript(from)
	}

	return b.NewUnsignedTransaction(txOuts, btcAmountType(relayFeePerKb), inputSource, changeSource, false)
}

func (b *Bridge) getTxOutputs(to string, amount *big.Int, memo string) (txOuts []*wireTxOutType, err error) {
	if amount != nil {
		err = b.addPayToAddrOutput(&txOuts, to, amount.Int64())
		if err != nil {
			return nil, err
		}
	}

	if memo != "" {
		err = b.addMemoOutput(&txOuts, memo)
		if err != nil {
			return nil, err
		}
	}

	return txOuts, err
}

func (b *Bridge) addPayToAddrOutput(txOuts *[]*wireTxOutType, to string, amount int64) error {
	if amount <= 0 {
		return nil
	}
	pkscript, err := b.GetPayToAddrScript(to)
	if err != nil {
		return err
	}
	*txOuts = append(*txOuts, b.NewTxOut(amount, pkscript))
	return nil
}

func (b *Bridge) addMemoOutput(txOuts *[]*wireTxOutType, memo string) error {
	if memo == "" {
		return nil
	}
	nullScript, err := b.NullDataScript(memo)
	if err != nil {
		return err
	}
	*txOuts = append(*txOuts, b.NewTxOut(0, nullScript))
	return nil
}

func (b *Bridge) findUxtosWithRetry(from string) (utxos []*electrs.ElectUtxo, err error) {
	for i := 0; i < retryCount; i++ {
		utxos, err = b.FindUtxos(from)
		if err == nil {
			break
		}
		time.Sleep(retryInterval)
	}
	return utxos, err
}

func (b *Bridge) getTransactionByHashWithRetry(txid string) (tx *electrs.ElectTx, err error) {
	for i := 0; i < retryCount; i++ {
		tx, err = b.GetTransactionByHash(txid)
		if err == nil {
			break
		}
		time.Sleep(retryInterval)
	}
	return tx, err
}

func (b *Bridge) getOutspendWithRetry(point *tokens.BtcOutPoint) (outspend *electrs.ElectOutspend, err error) {
	for i := 0; i < retryCount; i++ {
		outspend, err = b.GetOutspend(point.Hash, point.Index)
		if err == nil {
			break
		}
		time.Sleep(retryInterval)
	}
	return outspend, err
}

func (b *Bridge) selectUtxos(from string, target btcAmountType) (total btcAmountType, inputs []*wireTxInType, inputValues []btcAmountType, scripts [][]byte, err error) {
	p2pkhScript, err := b.GetPayToAddrScript(from)
	if err != nil {
		return 0, nil, nil, nil, err
	}

	utxos, err := b.findUxtosWithRetry(from)
	if err != nil {
		return 0, nil, nil, nil, err
	}

	var (
		tx      *electrs.ElectTx
		success bool
	)

	for _, utxo := range utxos {
		value := btcAmountType(*utxo.Value)
		if !isValidValue(value) {
			continue
		}
		tx, err = b.getTransactionByHashWithRetry(*utxo.Txid)
		if err != nil {
			continue
		}
		if *utxo.Vout >= uint32(len(tx.Vout)) {
			continue
		}
		output := tx.Vout[*utxo.Vout]
		if *output.ScriptpubkeyType != p2pkhType {
			continue
		}
		if output.ScriptpubkeyAddress == nil || *output.ScriptpubkeyAddress != from {
			continue
		}

		txIn, errf := b.NewTxIn(*utxo.Txid, *utxo.Vout, p2pkhScript)
		if errf != nil {
			continue
		}

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
		err = fmt.Errorf("not enough balance, total %v < target %v", total, target)
		return 0, nil, nil, nil, err
	}

	return total, inputs, inputValues, scripts, nil
}

func (b *Bridge) getUtxos(from string, target btcAmountType, prevOutPoints []*tokens.BtcOutPoint) (total btcAmountType, inputs []*wireTxInType, inputValues []btcAmountType, scripts [][]byte, err error) {
	p2pkhScript, err := b.GetPayToAddrScript(from)
	if err != nil {
		return 0, nil, nil, nil, err
	}

	for _, point := range prevOutPoints {
		outspend, errf := b.getOutspendWithRetry(point)
		if errf != nil {
			return 0, nil, nil, nil, errf
		}
		if *outspend.Spent {
			if outspend.Status != nil && outspend.Status.BlockHeight != nil {
				spentHeight := *outspend.Status.BlockHeight
				err = fmt.Errorf("out point (%v, %v) is spent at %v", point.Hash, point.Index, spentHeight)
			} else {
				err = fmt.Errorf("out point (%v, %v) is spent at txpool", point.Hash, point.Index)
			}
			return 0, nil, nil, nil, err
		}
		tx, errf := b.getTransactionByHashWithRetry(point.Hash)
		if errf != nil {
			return 0, nil, nil, nil, errf
		}
		if point.Index >= uint32(len(tx.Vout)) {
			err = fmt.Errorf("out point (%v, %v) index overflow", point.Hash, point.Index)
			return 0, nil, nil, nil, err
		}
		output := tx.Vout[point.Index]
		if *output.ScriptpubkeyType != p2pkhType {
			err = fmt.Errorf("out point (%v, %v) script pubkey type %v is not p2pkh", point.Hash, point.Index, *output.ScriptpubkeyType)
			return 0, nil, nil, nil, err
		}
		if output.ScriptpubkeyAddress == nil || *output.ScriptpubkeyAddress != from {
			err = fmt.Errorf("out point (%v, %v) script pubkey address %v is not %v", point.Hash, point.Index, *output.ScriptpubkeyAddress, from)
			return 0, nil, nil, nil, err
		}
		value := btcAmountType(*output.Value)
		if value == 0 {
			err = fmt.Errorf("out point (%v, %v) with zero value", point.Hash, point.Index)
			return 0, nil, nil, nil, err
		}

		txIn, errf := b.NewTxIn(point.Hash, point.Index, p2pkhScript)
		if errf != nil {
			return 0, nil, nil, nil, errf
		}

		total += value
		inputs = append(inputs, txIn)
		inputValues = append(inputValues, value)
		scripts = append(scripts, p2pkhScript)
	}
	if total < target {
		err = fmt.Errorf("not enough balance, total %v < target %v", total, target)
		return 0, nil, nil, nil, err
	}
	return total, inputs, inputValues, scripts, nil
}

type insufficientFundsError struct{}

func (insufficientFundsError) InputSourceError() {}
func (insufficientFundsError) Error() string {
	return "insufficient funds available to construct transaction"
}

// NewUnsignedTransaction ref btcwallet
// ref. https://github.com/btcsuite/btcwallet/blob/b07494fc2d662fdda2b8a9db2a3eacde3e1ef347/wallet/txauthor/author.go
// we only modify it to support P2PKH change script (the origin only support P2WPKH change script)
// and update estimate size because we are not use P2WKH
func (b *Bridge) NewUnsignedTransaction(outputs []*wireTxOutType, relayFeePerKb btcAmountType, fetchInputs txauthor.InputSource, fetchChange txauthor.ChangeSource, isAggregate bool) (*txauthor.AuthoredTx, error) {
	targetAmount := txauthor.SumOutputValues(outputs)
	estimatedSize := txsizes.EstimateSerializeSize(1, outputs, true)
	targetFee := txrules.FeeForSerializeSize(relayFeePerKb, estimatedSize)

	for {
		inputAmount, inputs, inputValues, scripts, err := fetchInputs(targetAmount + targetFee)
		if err != nil {
			return nil, err
		}
		if inputAmount < targetAmount+targetFee {
			return nil, insufficientFundsError{}
		}

		maxSignedSize := b.estimateSize(scripts, outputs, true, isAggregate)
		maxRequiredFee := txrules.FeeForSerializeSize(relayFeePerKb, maxSignedSize)
		if maxRequiredFee < btcAmountType(cfgMinRelayFee) {
			maxRequiredFee = btcAmountType(cfgMinRelayFee)
		}
		remainingAmount := inputAmount - targetAmount
		if remainingAmount < maxRequiredFee {
			if isAggregate {
				return nil, insufficientFundsError{}
			}
			targetFee = maxRequiredFee
			continue
		}

		unsignedTransaction := b.NewMsgTx(inputs, outputs, 0)

		changeIndex := -1
		changeAmount := inputAmount - targetAmount - maxRequiredFee
		if changeAmount != 0 {
			changeScript, err := fetchChange()
			if err != nil {
				return nil, err
			}
			// commont this to support P2PKH change script
			// if len(changeScript) > txsizes.P2WPKHPkScriptSize {
			//	return nil, errors.New("fee estimation requires change " +
			//		"scripts no larger than P2WPKH output scripts")
			//}
			threshold := txrules.GetDustThreshold(len(changeScript), txrules.DefaultRelayFeePerKb)
			if changeAmount < threshold {
				log.Debug("get rid of dust change", "amount", changeAmount, "threshold", threshold, "scriptsize", len(changeScript))
			} else {
				change := b.NewTxOut(int64(changeAmount), changeScript)
				l := len(outputs)
				outputs = append(outputs[:l:l], change)
				unsignedTransaction.TxOut = outputs
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

func (b *Bridge) estimateSize(scripts [][]byte, txOuts []*wireTxOutType, addChangeOutput, isAggregate bool) int {
	if !isAggregate {
		return txsizes.EstimateSerializeSize(len(scripts), txOuts, addChangeOutput)
	}

	var p2sh, p2pkh int
	for _, pkScript := range scripts {
		switch {
		case b.IsPayToScriptHash(pkScript):
			p2sh++
		default:
			p2pkh++
		}
	}

	size := txsizes.EstimateSerializeSize(p2pkh, txOuts, addChangeOutput)
	if p2sh > 0 {
		size += p2sh * redeemAggregateP2SHInputSize
	}

	return size
}
