package tron

import (
	"fmt"
	"math/big"
	"time"

	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var (
	retryRPCCount    = 3
	retryRPCInterval = 1 * time.Second

	defReserveFee = big.NewInt(1e16) // 0.01 TRX
)

// BuildRawTransaction build raw tx
func (b *Bridge) BuildRawTransaction(args *tokens.BuildTxArgs) (rawTx interface{}, err error) {
	var input []byte
	var tokenCfg *tokens.TokenConfig
	if args.Input == nil {
		if args.SwapType != tokens.NoSwapType {
			pairID := args.PairID
			tokenCfg = b.GetTokenConfig(pairID)
			if tokenCfg == nil {
				return nil, tokens.ErrUnknownPairID
			}
			if args.From == "" {
				args.From = tokenCfg.DcrmAddress // from
			}
		}
		switch args.SwapType {
		case tokens.SwapinType:
			if b.IsSrc {
				return nil, tokens.ErrBuildSwapTxInWrongEndpoint
			}
			amount := tokens.CalcSwappedValue(args.PairID, args.OriginValue, true)
			//  mint mapping asset
			return b.BuildSwapinTx(args.From, args.Bind, args.To, amount, args.SwapInfo.SwapID)
		case tokens.SwapoutType:
			if !b.IsSrc {
				return nil, tokens.ErrBuildSwapTxInWrongEndpoint
			}
			if tokenCfg.IsTrc20() {
				amount := tokens.CalcSwappedValue(args.PairID, args.OriginValue, false)
				//  transfer trc20
				return b.BuildTRC20Transfer(args.From, args.Bind, args.To, amount)
			} else {
				args.To = args.Bind
				input = []byte(tokens.UnlockMemoPrefix + args.SwapID)
				amount := tokens.CalcSwappedValue(args.PairID, args.OriginValue, false)
				// transfer trx
				return b.BuildTransfer(args.From, args.To, amount, input)
			}
		}
	} else {
		input = *args.Input
		if args.SwapType != tokens.NoSwapType {
			return nil, fmt.Errorf("forbid build raw swap tx with input data")
		}
	}
	return nil, fmt.Errorf("Cannot build tron transaction")
}
