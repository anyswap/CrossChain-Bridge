package ripple

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/crypto"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/data"
)

var (
	defaultFee     int64 = 10
	accountReserve       = big.NewInt(10000000)
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

	remain, err := b.GetBalance(from)
	if err != nil {
		log.Warn("Get from address balance error", "error", err)
	}
	if pairID == "XRP" {
		remain = new(big.Int).Sub(remain, amount)
	}
	if remain.Cmp(accountReserve) < 0 {
		return nil, fmt.Errorf("insufficient xrp balance")
	}

	rawtx, _, err := b.BuildUnsignedTransaction(from, pubkey, sequence, to, amount, sequence, fee)
	return rawtx, err
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

// BuildUnsignedTransaction build ripple unsigned transaction
func (b *Bridge) BuildUnsignedTransaction(fromAddress, fromPublicKey string, txseq uint32, toAddress string, amount *big.Int, sequence uint32, fee int64) (transaction interface{}, digests []string, err error) {
	pub, err := hex.DecodeString(fromPublicKey)
	ripplePubKey := ImportPublicKey(pub)
	amt := amount.String()
	memo := ""
	transaction, hash, _ := NewUnsignedPaymentTransaction(ripplePubKey, nil, txseq, toAddress, amt, fee, memo, "", false, false, false)
	digests = append(digests, hash.String())
	return
}

// GetSeq returns account tx sequence
func (b *Bridge) GetSeq(args *tokens.BuildTxArgs, address string) (nonceptr *uint32, err error) {
	var nonce uint32
	account, err := b.GetAccount(address)
	if err != nil {
		return nil, fmt.Errorf("cannot get account, %w", err)
	}
	if seq := account.AccountData.Sequence; seq != nil {
		nonce = *seq
	}
	if args == nil {
		return &nonce, nil
	}
	if args.SwapType != tokens.NoSwapType {
		tokenCfg := b.GetTokenConfig(args.PairID)
		if tokenCfg != nil && args.From == tokenCfg.DcrmAddress {
			nonce = uint32(b.AdjustNonce(args.PairID, uint64(nonce)))
		}
	}
	return &nonce, nil // unexpected
}

// NewUnsignedPaymentTransaction build ripple payment tx
// Partial and limit must be false
func NewUnsignedPaymentTransaction(key crypto.Key, keyseq *uint32, txseq uint32, dest string, amt string, fee int64, memo string, path string, nodirect bool, partial bool, limit bool) (data.Transaction, data.Hash256, []byte) {
	if partial {
		log.Warn("Building tx with partial")
	}
	if limit {
		log.Warn("Building tx with limit")
	}

	destination, amount := parseAccount(dest), parseAmount(amt)
	payment := &data.Payment{
		Destination: *destination,
		Amount:      *amount,
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
