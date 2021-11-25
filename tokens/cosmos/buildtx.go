package cosmos

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	TerraNative = "TerraNative"
)

// BuildRawTransaction build raw tx
func (b *Bridge) BuildRawTransaction(args *tokens.BuildTxArgs) (rawTx interface{}, err error) {
	var (
		pairID   = args.PairID
		denom    = b.SupportedCoins[strings.ToUpper(pairID)].Denom
		tokenCfg = b.GetTokenConfig(pairID)
		from     string
		to       string
		amount   *big.Int
		memo     string
	)
	extra := getOrInitExtra(args)
	args.Identifier = params.GetIdentifier()
	if tokenCfg == nil {
		return nil, fmt.Errorf("swap pair '%v' is not configed", pairID)
	}

	switch args.SwapType {
	case tokens.SwapinType:
		return nil, tokens.ErrSwapTypeNotSupported
	case tokens.SwapoutType:
		from = tokenCfg.DcrmAddress // from
		to = args.Bind              //to

		amount = tokens.CalcSwappedValue(pairID, args.OriginValue, false, args.OriginFrom, args.OriginTxTo) // amount
		memo = tokens.UnlockMemoPrefix + args.SwapID
	default:
		return nil, tokens.ErrUnknownSwapType
	}

	if from == "" {
		return nil, errors.New("no sender specified")
	}

	bal, getbalerr := b.GetTokenBalance(TerraNative, pairID, from)
	if getbalerr != nil {
		return nil, getbalerr
	}
	if bal.Cmp(amount) < 0 {
		return nil, fmt.Errorf("insufficient %v balance", pairID)
	}

	fromAcc, err := sdk.AccAddressFromBech32(from)
	if err != nil {
		// Never happens
		return nil, errors.New("From address error")
	}
	toAcc, err := sdk.AccAddressFromBech32(to)
	if err != nil {
		return nil, errors.New("To address does not refer to a cosmos account")
	}
	sendcoin := sdk.Coin{Denom: denom, Amount: sdk.NewIntFromBigInt(amount)}
	sendmsg := NewMsgSend(fromAcc, toAcc, sdk.Coins{sendcoin})

	accountNumber, err := b.GetAccountNumberCached(from)
	if err != nil {
		return nil, err
	}

	if extra.Nonce == nil {
		extra.Nonce, err = b.getSequence(args.PairID, from, args.SwapType)
		if err != nil {
			return nil, err
		}
	}
	seq := extra.Nonce

	tx := StdSignContent{
		ChainID:       b.ChainConfig.NetID,
		AccountNumber: accountNumber,
		Sequence:      *seq,
		Msgs:          []sdk.Msg{sendmsg},
		Memo:          memo,
	}
	fee := GetFeeAmount(pairID, &tx)
	tx.Fee = fee
	rawTx = tx
	return
}

func (b *Bridge) getSequence(pairID, from string, swapType tokens.SwapType) (*uint64, error) {
	var seq uint64
	seq, err := b.GetPoolNonce(from, "pending")
	if err != nil {
		return nil, err
	}
	if swapType != tokens.NoSwapType {
		tokenCfg := b.GetTokenConfig(pairID)
		if tokenCfg != nil && from == tokenCfg.DcrmAddress {
			seq = b.AdjustNonce(pairID, seq)
		}
	}
	return &seq, nil
}

func getOrInitExtra(args *tokens.BuildTxArgs) *tokens.EthExtraArgs {
	if args.Extra == nil || args.Extra.EthExtra == nil {
		args.Extra = &tokens.AllExtras{EthExtra: &tokens.EthExtraArgs{}}
	}
	return args.Extra.EthExtra
}
