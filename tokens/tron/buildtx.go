package tron

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	proto "github.com/golang/protobuf/proto"

	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens/eth"
)

var (
	SwapinFeeLimit int64 = 300000000 // 300 TRX
	TransferTRXLimit int64 = 300000000 // 300 TRX
	TransferTRC20FeeLimit int64 = 300000000 // 300 TRX

	ExtraExpiration int64 = 900000 // 15 min
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
			//  mint mapping asset
			err = b.buildSwapinTxInput(args)
			if err != nil {
				return nil, err
			}
			rawTx, err = b.BuildSwapinTx(args.From, args.To, *args.Input)
			if err == nil {
				txmsg, _ := proto.Marshal(rawTx.(*core.Transaction))
				args.Extra = &tokens.AllExtras{
					TronExtra: &tokens.TronExtraArgs{
						RawTx: fmt.Sprintf("%X", txmsg),
					},
				}
			}
			return
		case tokens.SwapoutType:
			if !b.IsSrc {
				return nil, tokens.ErrBuildSwapTxInWrongEndpoint
			}
			if tokenCfg.IsTrc20() {
				// TRC20
				amount := tokens.CalcSwappedValue(args.PairID, args.OriginValue, false)
				args.To = tokenCfg.ContractAddress
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
				// TRX
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

// build input for calling `Swapin(bytes32 txhash, address account, uint256 amount)`
func (b *Bridge) buildSwapinTxInput(args *tokens.BuildTxArgs) error {
	pairID := args.PairID
	funcHash := eth.ExtCodeParts["SwapinFuncHash"]
	txHash := common.HexToHash(args.SwapID)
	address := common.HexToAddress(args.Bind)
	if address == (common.Address{}) || !common.IsHexAddress(args.Bind) {
		log.Warn("swapin to wrong address", "address", args.Bind)
		return errors.New("can not swapin to empty or invalid address")
	}
	amount := tokens.CalcSwappedValue(pairID, args.OriginValue, true)

	input := eth.PackDataWithFuncHash(funcHash, txHash, address, amount)
	args.Input = &input // input

	token := b.GetTokenConfig(pairID)
	if token == nil {
		return tokens.ErrUnknownPairID
	}
	args.To = token.ContractAddress // to
	return nil
}