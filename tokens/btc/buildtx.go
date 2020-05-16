package btc

import (
	"errors"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/wallet/txauthor"
	"github.com/btcsuite/btcwallet/wallet/txrules"
	"github.com/btcsuite/btcwallet/wallet/txsizes"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
)

var (
	minReserveAmount = btcutil.Amount(1000)
	defRelayFeePerKb = btcutil.Amount(2000)
)

func (b *BtcBridge) BuildRawTransaction(args *tokens.BuildTxArgs) (rawTx interface{}, err error) {
	var (
		token         = b.TokenConfig
		from          = args.From
		to            = args.To
		amount        = args.Value
		memo          = args.Memo
		relayFeePerKb = defRelayFeePerKb
		changeAddress string
		txOuts        []*wire.TxOut
	)

	switch args.SwapType {
	case tokens.Swap_Swapin:
		return nil, tokens.ErrSwapTypeNotSupported
	case tokens.Swap_Swapout, tokens.Swap_Recall:
		from = token.DcrmAddress
		amount = tokens.CalcSwappedValue(amount, b.IsSrc)
	}

	if from == "" {
		return nil, errors.New("no sender specified")
	}

	changeAddress = from
	if args.ChangeAddress != nil {
		changeAddress = *args.ChangeAddress
	}
	if args.RelayFeePerKb != nil {
		relayFeePerKb = btcutil.Amount(*args.RelayFeePerKb)
	}

	pkscript, err := b.getPayToAddrScript(to)
	if err != nil {
		return nil, err
	}
	txOut := wire.NewTxOut(amount.Int64(), pkscript)
	txOuts = append(txOuts, txOut)

	if memo != "" {
		nullScript, err := txscript.NullDataScript([]byte(memo))
		if err != nil {
			return nil, err
		}
		txOut = wire.NewTxOut(0, nullScript)
		txOuts = append(txOuts, txOut)
	}

	estimatedSize := txsizes.EstimateVirtualSize(0, 1, 0, txOuts, true)
	targetFee := txrules.FeeForSerializeSize(relayFeePerKb, estimatedSize)

	inputSource := func(target btcutil.Amount) (
		total btcutil.Amount, inputs []*wire.TxIn, inputValues []btcutil.Amount, scripts [][]byte, err error) {
		return b.selectUtxos(from, target, targetFee, relayFeePerKb, txOuts)
	}
	changeSource := func() ([]byte, error) {
		return b.getPayToAddrScript(changeAddress)
	}
	return NewUnsignedTransaction(txOuts, relayFeePerKb, inputSource, changeSource)
}

func (b *BtcBridge) getPayToAddrScript(address string) ([]byte, error) {
	chainConfig := b.GetChainConfig()
	toAddr, err := btcutil.DecodeAddress(address, chainConfig)
	if err != nil {
		return nil, err
	}
	return txscript.PayToAddrScript(toAddr)
}

func (b *BtcBridge) selectUtxos(from string, target, targetFee, relayFeePerKb btcutil.Amount, txOuts []*wire.TxOut) (
	total btcutil.Amount, inputs []*wire.TxIn, inputValues []btcutil.Amount, scripts [][]byte, err error) {

	latest, err := b.GetLatestBlockNumber()
	if err != nil {
		return
	}

	utxos, err := b.FindUtxos(from)
	if err != nil {
		return
	}

	p2pkhScript, err := b.getPayToAddrScript(from)
	if err != nil {
		return
	}

	needConfirmations := *b.TokenConfig.Confirmations
	success := false

	for _, utxo := range utxos {
		value := btcutil.Amount(*utxo.Value)
		if value <= 0 {
			continue
		}
		if value > btcutil.MaxSatoshi {
			continue
		}
		status := utxo.Status
		if !*status.Confirmed {
			continue
		}
		if *status.Block_height+needConfirmations > latest {
			continue
		}
		tx, err := b.GetTransaction(*utxo.Txid)
		if err != nil {
			continue
		}
		if *utxo.Vout >= uint32(len(tx.Vout)) {
			continue
		}
		output := tx.Vout[*utxo.Vout]
		if *output.Scriptpubkey_type != "p2pkh" {
			continue
		}
		if *output.Scriptpubkey_address != from {
			continue
		}
		txHash, err := chainhash.NewHashFromStr(*utxo.Txid)
		if err != nil {
			continue
		}
		preOut := wire.NewOutPoint(txHash, *utxo.Vout)
		if txrules.IsDustAmount(value, len(p2pkhScript), relayFeePerKb) {
			continue
		}
		txIn := wire.NewTxIn(preOut, p2pkhScript, nil)

		total += value
		inputs = append(inputs, txIn)
		inputValues = append(inputValues, value)
		scripts = append(scripts, p2pkhScript)

		if total > target+targetFee+minReserveAmount {
			success = true
			break
		}
	}

	if !success {
		err = errors.New("Not enough balance")
		return
	}

	return total, inputs, inputValues, scripts, nil
}

type insufficientFundsError struct{}

func (insufficientFundsError) InputSourceError() {}
func (insufficientFundsError) Error() string {
	return "insufficient funds available to construct transaction"
}

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

		maxSignedSize := txsizes.EstimateVirtualSize(p2pkh, p2wpkh,
			nested, outputs, true)
		maxRequiredFee := txrules.FeeForSerializeSize(relayFeePerKb, maxSignedSize)
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
		if changeAmount != 0 && !txrules.IsDustAmount(changeAmount,
			txsizes.P2WPKHPkScriptSize, relayFeePerKb) {
			changeScript, err := fetchChange()
			if err != nil {
				return nil, err
			}
			// commont this to support P2PKH change script
			//if len(changeScript) > txsizes.P2WPKHPkScriptSize {
			//	return nil, errors.New("fee estimation requires change " +
			//		"scripts no larger than P2WPKH output scripts")
			//}
			change := wire.NewTxOut(int64(changeAmount), changeScript)
			l := len(outputs)
			unsignedTransaction.TxOut = append(outputs[:l:l], change)
			changeIndex = l
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
