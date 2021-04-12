package tron

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	proto "github.com/golang/protobuf/proto"

	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var (
	retryRPCCount    = 3
	retryRPCInterval = 1 * time.Second

	defReserveFee = big.NewInt(1e16) // 0.01 TRX
)

// BuildRawTransaction build raw tx
func (b *Bridge) BuildRawTransaction(args *tokens.BuildTxArgs) (rawTx interface{}, err error) {
	args.Identifier = params.GetIdentifier()
	//	For server, build tx and set TronExtra with marshal tx
	//	For oracle nodes, TronExtra should exists in args
	//		unmarshal raw tx from TronExtra,
	// 		verify tx with main args such as PairID, From, To, Amount and omit other args
	//		return the decoded raw tx so that it afford to pass msg hash check
	if args.Extra != nil && args.Extra.TronExtra != nil {
		rawtx, decodeErr := hex.DecodeString(args.Extra.TronExtra.RawTx)
		if decodeErr != nil {
			return nil, decodeErr
		} 

		var coretx core.Transaction
		unmarshalErr := proto.Unmarshal(rawtx, &coretx)
		if unmarshalErr != nil {
			return nil, unmarshalErr
		}

		verifyErr := b.verifyTransactionWithArgs(&coretx, args)
		if verifyErr == nil {
			return &coretx, nil
		}
		return nil, verifyErr
	}
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
				args.Value = amount
				//  transfer trc20
				rawTx, err =  b.BuildTRC20Transfer(args.From, args.Bind, args.To, amount)
				if err == nil {
					txmsg, _ := proto.Marshal(rawTx.(*core.Transaction))
					args.Extra = &tokens.AllExtras{
						TronExtra: &tokens.TronExtraArgs{
							RawTx: fmt.Sprintf("%X", txmsg),
						},
					}
				}
				return
			} else {
				args.To = args.Bind
				input = []byte(tokens.UnlockMemoPrefix + args.SwapID)
				amount := tokens.CalcSwappedValue(args.PairID, args.OriginValue, false)
				args.Value = amount
				// transfer trx
				rawTx, err = b.BuildTransfer(args.From, args.To, amount, input)
				if err == nil {
					txmsg, _ := proto.Marshal(rawTx.(*core.Transaction))
					args.Extra = &tokens.AllExtras{
						TronExtra: &tokens.TronExtraArgs{
							RawTx: fmt.Sprintf("%X", txmsg),
						},
					}
				}
				return
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
