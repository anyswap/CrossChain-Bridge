package cosmos

import (
	"errors"
	"fmt"
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
		from     = args.From
		to       = args.Bind
		amount   = args.Value
		memo     = args.Memo
	)
	args.Identifier = params.GetIdentifier()
	if tokenCfg == nil {
		return nil, fmt.Errorf("swap pair '%v' is not configed", pairID)
	}

	switch args.SwapType {
	case tokens.SwapinType:
		return nil, tokens.ErrSwapTypeNotSupported
	case tokens.SwapoutType:
		from = tokenCfg.DcrmAddress                                                 // from
		amount = tokens.CalcSwappedValue(pairID, args.OriginValue, false, from, to) // amount
		memo = tokens.UnlockMemoPrefix + args.SwapID
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

	fee := GetFeeAmount(pairID)

	accountNumber, err := b.GetAccountNumberCached(from)
	if err != nil {
		return nil, err
	}

	seq, err := b.getSequence(args.PairID, from, args.SwapType)
	if err != nil {
		return nil, err
	}

	rawTx = StdSignContent{
		ChainID:       b.ChainConfig.NetID,
		AccountNumber: accountNumber,
		Sequence:      *seq,
		Fee:           fee,
		Msgs:          []sdk.Msg{sendmsg},
		Memo:          memo,
	}
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

// AdjustNonce adjust account nonce (eth like chain)
func (b *Bridge) AdjustNonce(pairID string, value uint64) (nonce uint64) {
	tokenCfg := b.GetTokenConfig(pairID)
	account := strings.ToLower(tokenCfg.DcrmAddress)
	nonce = value
	if b.IsSrcEndpoint() {
		if b.SwapoutNonce[account] > value {
			nonce = b.SwapoutNonce[account]
		} else {
			b.SwapoutNonce[account] = value
		}
	} else {
		if b.SwapinNonce[account] > value {
			nonce = b.SwapinNonce[account]
		} else {
			b.SwapinNonce[account] = value
		}
	}
	return nonce
}
