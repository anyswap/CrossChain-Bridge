package ripple

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/crypto"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/data"
)

var (
	defaultFee     int64 = 10
	accountReserve       = big.NewInt(10000000)
	minReserveFee  *big.Int
)

// BuildRawTransaction build raw tx
func (b *Bridge) BuildRawTransaction(args *tokens.BuildTxArgs) (rawTx interface{}, err error) {
	var (
		pairID   = args.PairID
		sequence uint32
		fee      int64
		pubkey   string
		from     string
		to       string
		toTag    *uint32
		amount   *big.Int
	)

	token := b.GetTokenConfig(pairID)
	if token == nil {
		return nil, fmt.Errorf("swap pair '%v' is not configed", pairID)
	}

	switch args.SwapType {
	case tokens.SwapinType:
		return nil, tokens.ErrSwapTypeNotSupported
	case tokens.SwapoutType:
		from = token.DcrmAddress                                                    // from
		to = args.Bind                                                              // to
		amount = tokens.CalcSwappedValue(pairID, args.OriginValue, false, from, to) // amount
		pubkey = b.GetDcrmPublicKey(pairID)
	default:
		return nil, tokens.ErrUnknownSwapType
	}

	if from == "" {
		return nil, errors.New("no sender specified")
	}

	var extra *tokens.RippleExtra
	if args.Extra == nil || args.Extra.RippleExtra == nil {
		extra = b.swapoutDefaultArgs(args)
		args.Extra = &tokens.AllExtras{RippleExtra: extra}
		sequence = *extra.Sequence
		fee = *extra.Fee
	} else {
		extra = args.Extra.RippleExtra
		if extra.Sequence != nil {
			sequence = *extra.Sequence
		}
		if extra.Fee != nil {
			fee = *extra.Fee
		}
	}

	if args.SwapType != tokens.NoSwapType {
		args.Identifier = params.GetIdentifier()
	}

	amt, err := getPaymentAmount(amount, token)
	if err != nil {
		return nil, err
	}

	if token.RippleExtra.IsNative() {
		needAmount := amount
		if args.SwapType != tokens.NoSwapType {
			needAmount = new(big.Int).Add(amount, b.getMinReserveFee())
		}
		if err = b.checkNativeBalance(from, needAmount, true); err != nil {
			return nil, err
		}
		if err = b.checkNativeBalance(to, amount, false); err != nil {
			return nil, err
		}
	} else {
		if err = b.checkNativeBalance(to, nil, false); err != nil {
			return nil, err
		}
		if err = b.checkNonNativeBalance(token.RippleExtra.Currency, token.RippleExtra.Issuer, from, amt); err != nil {
			return nil, err
		}
	}

	ripplePubKey := ImportPublicKey(common.FromHex(pubkey))
	rawtx, _, _ := NewUnsignedPaymentTransaction(ripplePubKey, nil, sequence, to, toTag, amt.String(), fee, "", "", false, false, false)

	return rawtx, err
}

func getPaymentAmount(amount *big.Int, token *tokens.TokenConfig) (*data.Amount, error) {
	currency, exist := currencyMap[token.RippleExtra.Currency]
	if !exist {
		return nil, fmt.Errorf("non exist currency %v", token.RippleExtra.Currency)
	}

	if !amount.IsInt64() {
		return nil, fmt.Errorf("amount value %v is overflow of type int64", amount)
	}

	if currency.IsNative() { // native XRP
		return data.NewAmount(amount.Int64())
	}

	issuer, exist := issuerMap[token.RippleExtra.Issuer]
	if !exist {
		return nil, fmt.Errorf("non exist issuer %v", token.RippleExtra.Issuer)
	}

	// get a Value of amount*10^(-decimals)
	value, err := data.NewNonNativeValue(amount.Int64(), -int64(*token.Decimals))
	if err != nil {
		log.Error("getPaymentAmount failed", "currency", token.RippleExtra.Currency, "issuer", token.RippleExtra.Issuer, "amount", amount, "decimals", *token.Decimals, "err", err)
		return nil, err
	}

	return &data.Amount{
		Value:    value,
		Currency: currency,
		Issuer:   *issuer,
	}, nil
}

func (b *Bridge) getMinReserveFee() *big.Int {
	if minReserveFee != nil {
		return minReserveFee
	}
	minReserveFee = b.ChainConfig.GetMinReserveFee()
	if minReserveFee == nil {
		minReserveFee = big.NewInt(100000) // default 0.1 XRP
	}
	return minReserveFee
}

func (b *Bridge) swapoutDefaultArgs(txargs *tokens.BuildTxArgs) *tokens.RippleExtra {
	args := &tokens.RippleExtra{
		Sequence: new(uint32),
		Fee:      new(int64),
	}

	token := b.GetTokenConfig(txargs.PairID)
	if token == nil {
		log.Warn("Swap pair id not configed", "pairID", txargs.PairID)
		return args
	}

	dcrmAddr := token.DcrmAddress

	seq, err := b.GetSeq(txargs, dcrmAddr)
	if err != nil {
		log.Warn("Get sequence error when setting default ripple args", "error", err)
	}
	*args.Sequence = *seq
	addPercent := token.PlusGasPricePercentage
	if addPercent > 0 {
		*args.Fee = *args.Fee * (int64(100 + addPercent)) / 100
	}
	if *args.Fee < defaultFee {
		*args.Fee = defaultFee
	}
	return args
}

func (b *Bridge) checkNativeBalance(account string, amount *big.Int, isPay bool) error {
	balance, err := b.GetBalance(account)
	if err != nil && balance == nil {
		balance = big.NewInt(0)
	}

	remain := balance
	if amount != nil {
		if isPay {
			remain = new(big.Int).Sub(balance, amount)
		} else {
			remain = new(big.Int).Add(balance, amount)
		}
	}

	if remain.Cmp(accountReserve) < 0 {
		if isPay {
			return fmt.Errorf("insufficient native balance, sender: %v", account)
		}
		return fmt.Errorf("insufficient native balance, receiver: %v", account)
	}

	return nil
}

func (b *Bridge) checkNonNativeBalance(currency, issuer, account string, amount *data.Amount) error {
	accl, err := b.GetAccountLine(currency, issuer, account)
	if err != nil {
		return err
	}
	if accl.Balance.Value.Compare(*amount.Value) < 0 {
		return fmt.Errorf("insufficient %v balance, issuer: %v, account: %v", currency, issuer, account)
	}
	return nil
}

// GetTxBlockInfo impl NonceSetter interface
func (b *Bridge) GetTxBlockInfo(txHash string) (blockHeight, blockTime uint64) {
	txStatus, err := b.GetTransactionStatus(txHash)
	if err != nil {
		return 0, 0
	}
	return txStatus.BlockHeight, txStatus.BlockTime
}

// GetPoolNonce impl NonceSetter interface
func (b *Bridge) GetPoolNonce(address, _height string) (uint64, error) {
	var nonce uint32
	account, err := b.GetAccount(address)
	if err != nil {
		return 0, fmt.Errorf("cannot get account, %w", err)
	}
	if seq := account.AccountData.Sequence; seq != nil {
		nonce = *seq
	}
	return uint64(nonce), nil
}

// GetSeq returns account tx sequence
func (b *Bridge) GetSeq(args *tokens.BuildTxArgs, address string) (nonceptr *uint32, err error) {
	nonceVal, err := b.GetPoolNonce(address, "")
	if err != nil {
		return nil, err
	}
	nonce := uint32(nonceVal)
	if args == nil {
		return &nonce, nil
	}
	if args.SwapType != tokens.NoSwapType {
		tokenCfg := b.GetTokenConfig(args.PairID)
		if tokenCfg != nil && args.From == tokenCfg.DcrmAddress {
			nonce = uint32(b.AdjustNonce(args.PairID, uint64(nonce)))
		}
	}
	return &nonce, nil
}

// NewUnsignedPaymentTransaction build ripple payment tx
// Partial and limit must be false
func NewUnsignedPaymentTransaction(key crypto.Key, keyseq *uint32, txseq uint32, dest string, destinationTag *uint32, amt string, fee int64, memo, path string, nodirect, partial, limit bool) (data.Transaction, data.Hash256, []byte) {
	if partial {
		log.Warn("Building tx with partial")
	}
	if limit {
		log.Warn("Building tx with limit")
	}

	destination, amount := parseAccount(dest), parseAmount(amt)
	payment := &data.Payment{
		Destination:    *destination,
		Amount:         *amount,
		DestinationTag: destinationTag,
	}
	payment.TransactionType = data.PAYMENT

	if memo != "" {
		memoStr := new(data.Memo)
		memoStr.Memo.MemoType = []byte("BIND")
		memoStr.Memo.MemoData = []byte(memo)
		payment.Memos = append(payment.Memos, *memoStr)
	}

	if path != "" {
		payment.Paths = parsePaths(path)
	}
	payment.Flags = new(data.TransactionFlag)
	if nodirect {
		*payment.Flags = *payment.Flags | data.TxNoDirectRipple
	}
	if partial {
		*payment.Flags = *payment.Flags | data.TxPartialPayment
	}
	if limit {
		*payment.Flags = *payment.Flags | data.TxLimitQuality
	}

	base := payment.GetBase()

	base.Sequence = txseq

	fei, err := data.NewNativeValue(fee)
	if err != nil {
		return nil, data.Hash256{}, nil
	}
	base.Fee = *fei

	copy(base.Account[:], key.Id(keyseq))

	payment.InitialiseForSigning()
	copy(payment.GetPublicKey().Bytes(), key.Public(keyseq))
	hash, msg, err := data.SigningHash(payment)
	if err != nil {
		log.Warn("Generate ripple tx signing hash error", "error", err)
		return nil, data.Hash256{}, nil
	}
	log.Info("Build unsigned tx success", "signing hash", hash.String(), "blob", fmt.Sprintf("%X", msg))

	return payment, hash, msg
}
