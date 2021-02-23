package cosmos

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

var defaultSwapoutGas uint64 = 300000

var GetFeeAmount = func() authtypes.StdFee {
	// TODO
	return sdk.Coins{sdk.Coin{"uatom", sdk.NewInt(3000)}}
}

// BuildRawTransaction build raw tx
func (b *Bridge) BuildRawTransaction(args *tokens.BuildTxArgs) (rawTx interface{}, err error) {
	var (
		pairID   = args.PairID
		tokenCfg = b.GetTokenConfig(pairID)
		from     = args.From
		to       = args.To
		amount   = args.Value
		memo     = args.Memo
	)
	tokenCfg = b.GetTokenConfig(pairID)
	if token == nil {
		return nil, fmt.Errorf("swap pair '%v' is not configed", pairID)
	}

	switch args.SwapType {
	case tokens.SwapinType:
		return nil, tokens.ErrSwapTypeNotSupported
	case tokens.SwapoutType:
		from = token.DcrmAddress                                          // from
		amount = tokens.CalcSwappedValue(pairID, args.OriginValue, false) // amount
		memo = tokens.UnlockMemoPrefix + args.SwapID
	}

	if from == "" {
		return nil, errors.New("no sender specified")
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
	sendcoin := sdk.Coin{TheCoin.Denom, sdk.NewIntFromBigInt(amount)}
	sendmsg := banktypes.NewMsgSend(fromAcc, toAcc, sendcoin)

	feeAmount := b.GetFeeAmount()
	fee := authtypes.NewStdFee(defaultSwapoutGas)

	accountNumber := b.GetAccountNumberCached(from)

	seq, err := b.getSequence(args.PairID, from, args.SwapType)
	if err != nil {
		return nil, err
	}

	stdmsg := authtypes.StdSignMsg{
		ChainID: b.ChainConfig.NetID,
		AccountNumber: accountNumber,
		Sequence: *seq,
		Fee: feeAmount
		Msgs: []sdk.Msg{sendmsg},
		Memo: memo,
	}
	return nil, nil
}

func (b *Bridge) getSequence(pairID, from string, swapType tokens.SwapType) (*uint64, error) {
	var seq uint64
	seq, err := b.GetPoolNonce
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
