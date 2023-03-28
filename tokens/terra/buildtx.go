package terra

import (
	"fmt"
	"math/big"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	wasmtypes "github.com/classic-terra/core/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

var (
	retryRPCCount    = 3
	retryRPCInterval = 1 * time.Second

	signMode = signing.SignMode_SIGN_MODE_DIRECT
)

// default values to calc tx gas and fees
var (
	DefaultFees   = "3000uluna"
	DefaultFeeCap = uint64(10000)

	DefaultGasLimit          = uint64(200000)
	DefaultPlusGasPercentage = uint64(20)

	DefaultGasPrice = 0.02
)

// GetDefaultExtras get default extras
func (b *Bridge) GetDefaultExtras() *tokens.AllExtras {
	return &tokens.AllExtras{TerraExtra: &tokens.TerraExtra{}}
}

// BuildRawTransaction build raw tx
func (b *Bridge) BuildRawTransaction(args *tokens.BuildTxArgs) (rawTx interface{}, err error) {
	tokenCfg, err := b.getAndInitTokenConfig(args.PairID)
	if err != nil {
		return nil, err
	}

	switch args.SwapType {
	case tokens.SwapinType:
		return nil, tokens.ErrSwapTypeNotSupported
	case tokens.SwapoutType:
		return b.buildSwapoutTx(args, tokenCfg)
	default:
		return nil, tokens.ErrUnknownSwapType
	}
}

func (b *Bridge) getAndInitTokenConfig(pairID string) (tokenCfg *tokens.TokenConfig, err error) {
	tokenCfg = b.GetTokenConfig(pairID)
	if tokenCfg == nil {
		return nil, fmt.Errorf("swap pair '%v' is not configed", pairID)
	}
	if tokenCfg.DcrmAccountNumber == 0 {
		tokenCfg.DcrmAccountNumber, err = b.GetAccountNumber(tokenCfg.DcrmAddress)
		if err != nil {
			return nil, fmt.Errorf("init dcrm account number failed: %w", err)
		}
	}
	return tokenCfg, nil
}

func (b *Bridge) buildSwapoutTx(args *tokens.BuildTxArgs, tokenCfg *tokens.TokenConfig) (txb *TxBuilder, err error) {
	from := tokenCfg.DcrmAddress
	if from == "" {
		return nil, tokens.ErrTxWithWrongSender
	}

	amount := tokens.CalcSwappedValue(args.PairID, args.OriginValue, false, args.OriginFrom, args.OriginTxTo)
	if amount.Sign() < 0 {
		return nil, fmt.Errorf("negative token amount")
	}

	burnAmount := big.NewInt(0)
	if shouldBurnTax(tokenCfg.Unit) && tokenCfg.ContractAddress == "" {
		// tax burn amount is equal to 1.2% total value
		burnAmount = new(big.Int).Mul(amount, big.NewInt(12))
		burnAmount.Div(burnAmount, big.NewInt(1000))
		// adjust receiver amount by deduct tax burn fee
		amount.Sub(amount, burnAmount)
	}
	args.SwapValue = amount // swap value

	minReserve := b.getMinReserveFee()
	err = b.checkCoinBalance(from, b.ChainConfig.MetaCoin.Unit, minReserve)
	if err != nil {
		return nil, err
	}

	needAmount := amount
	if tokenCfg.ContractAddress == "" {
		if tokenCfg.Unit == b.ChainConfig.MetaCoin.Unit {
			needAmount = new(big.Int).Add(amount, minReserve)
		}
		err = b.checkCoinBalance(from, tokenCfg.Unit, needAmount)
	} else {
		err = b.checkTokenBalance(tokenCfg.ContractAddress, from, needAmount)
	}
	if err != nil {
		return nil, err
	}

	extra, err := b.initExtra(args, tokenCfg)
	if err != nil {
		return nil, err
	}

	memo := tokens.UnlockMemoPrefix + args.SwapID
	txb, err = b.BuildTx(from, args.Bind, memo, amount, burnAmount, extra, tokenCfg)
	if err != nil {
		return nil, err
	}

	log.Info("build tx success",
		"identifier", args.Identifier, "pairID", args.PairID, "swapID", args.SwapID,
		"originValue", args.OriginValue, "swapValue", args.SwapValue,
		"from", from, "bind", args.Bind,
		"amount", amount, "burnAmount", burnAmount,
		"replaceNum", args.GetReplaceNum(),
		"gas", txb.GetGas(), "fees", txb.GetFee().String(),
		"chainID", txb.GetSignerData().ChainID,
		"sequence", txb.GetSignerData().Sequence,
		"accountNumber", txb.GetSignerData().AccountNumber,
	)
	return txb, nil
}

// BuildTx build tx
func (b *Bridge) BuildTx(
	from, to, memo string,
	amount, burnAmount *big.Int,
	extra *tokens.TerraExtra,
	tokenCfg *tokens.TokenConfig,
) (*TxBuilder, error) {
	txb := newBuilder()

	txb.SetSignerData(
		b.ChainConfig.NetID,
		tokenCfg.DcrmAccountNumber,
		*extra.Sequence)

	txb.SetMemo(memo)

	txb.SetGasLimit(*extra.Gas)

	parsedFees, err := sdk.ParseCoinsNormalized(*extra.Fees)
	if err != nil {
		return nil, err
	}
	txb.SetFeeAmount(parsedFees)

	accFrom, err := sdk.AccAddressFromBech32(from)
	if err != nil {
		return nil, err
	}
	txb.SetFeePayer(accFrom)

	accTo, err := sdk.AccAddressFromBech32(to)
	if err != nil {
		return nil, err
	}

	var msg sdk.Msg

	if tokenCfg.ContractAddress != "" {
		accContract, errf := sdk.AccAddressFromBech32(tokenCfg.ContractAddress)
		if errf != nil {
			return nil, errf
		}
		execMsg, errf := GetTokenTransferExecMsg(accTo.String(), amount.String())
		if errf != nil {
			return nil, errf
		}
		msg = wasmtypes.NewMsgExecuteContract(accFrom, accContract, execMsg, nil)
	} else {
		msg = &banktypes.MsgSend{
			FromAddress: accFrom.String(),
			ToAddress:   accTo.String(),
			Amount: sdk.Coins{
				sdk.NewCoin(tokenCfg.Unit, sdk.NewIntFromBigInt(amount)),
			},
		}
	}

	err = txb.SetMsgs(msg)
	if err != nil {
		return nil, err
	}

	pubkey, err := PubKeyFromStr(tokenCfg.DcrmPubkey)
	if err != nil {
		return nil, err
	}

	sigData := signing.SingleSignatureData{
		SignMode:  signMode,
		Signature: nil,
	}
	sig := signing.SignatureV2{
		PubKey:   pubkey,
		Data:     &sigData,
		Sequence: *extra.Sequence,
	}
	err = txb.SetSignatures(sig)
	if err != nil {
		return nil, err
	}

	err = txb.ValidateBasic()
	if err != nil {
		return nil, err
	}

	if params.IsSwapServer {
		err = b.adjustFees(txb, extra, tokenCfg, burnAmount)
		if err != nil {
			return nil, err
		}
	}

	return txb, nil
}

func (b *Bridge) adjustFees(txb *TxBuilder, extra *tokens.TerraExtra, tokenCfg *tokens.TokenConfig, burnAmount *big.Int) error {
	/*gasUsed, err := b.simulateTx(txb)
	if err != nil {
		return err
	}*/
	gasUsed := uint64(200000)
	plusGasPercentage := tokenCfg.PlusGasPercentage
	if plusGasPercentage == 0 {
		plusGasPercentage = DefaultPlusGasPercentage
	}
	gasNeed := gasUsed * (100 + plusGasPercentage) / 100
	gas := txb.GetGas()
	if gasNeed > gas {
		log.Info("build tx adjust gas limit", "old", gas, "new", gasNeed)
		gas = gasNeed
		txb.SetGasLimit(gas) // adjust gas limit
		*extra.Gas = gas     // update extra gas limit
	}

	fees := txb.GetFee()
	if len(fees) == 1 {
		denom := fees[0].Denom
		gasPrice := tokenCfg.DefaultGasPrice
		if gasPrice == 0 {
			gasPriceDec, errf := b.GetGasPrice(denom)
			if errf == nil {
				gasPrice, errf = gasPriceDec.Float64()
			}
			if errf != nil {
				return errf
			}
		}
		if gasPrice == 0 {
			gasPrice = DefaultGasPrice
		}
		feesNeed := uint64(float64(gas) * gasPrice)
		feeCap := tokenCfg.FeeCap
		if feeCap == 0 {
			feeCap = DefaultFeeCap
		}
		if feesNeed > feeCap {
			log.Info("build tx feeNeed is larger than feeCap", "gas", gas, "gasPrice", gasPrice, "feeNeed", feesNeed, "feeCap", feeCap)
			feesNeed = feeCap
		}

		if burnAmount.Sign() > 0 && tokenCfg.Unit == denom {
			feesNeed += burnAmount.Uint64()
		}

		if fees[0].Amount.Uint64() < feesNeed {
			log.Info("build tx adjust fees", "old", fees.String(), "new", feesNeed, "burnAmount", burnAmount)
			fees = sdk.NewCoins(sdk.NewCoin(denom, sdk.NewIntFromUint64(feesNeed)))
			txb.SetFeeAmount(fees)      // adjust fees
			*extra.Fees = fees.String() // update extra fees
		}

		if burnAmount.Sign() > 0 && tokenCfg.Unit != denom {
			fees = sdk.NewCoins(fees[0], sdk.NewCoin(tokenCfg.Unit, sdk.NewIntFromUint64(burnAmount.Uint64())))
			txb.SetFeeAmount(fees)      // adjust fees
			*extra.Fees = fees.String() // update extra fees
		}
	}

	return nil
}

func (b *Bridge) simulateTx(txb *TxBuilder) (gasUsed uint64, err error) {
	txBytes, err := txb.GetTxBytes()
	if err != nil {
		return 0, err
	}
	simRes, err := b.SimulateTx(&SimulateRequest{TxBytes: txBytes})
	if err != nil {
		return 0, err
	}
	gasUsed, err = common.GetUint64FromStr(simRes.GasInfo.GasUsed)
	if err != nil {
		log.Warn("simulate tx failed", "err", err)
		return 0, err
	}
	log.Info("simulate tx success", "gasUsed", gasUsed)
	return gasUsed, nil
}

func (b *Bridge) initExtra(args *tokens.BuildTxArgs, tokenCfg *tokens.TokenConfig) (extra *tokens.TerraExtra, err error) {
	extra = getOrInitExtra(args)
	if extra.Sequence == nil {
		extra.Sequence, err = b.getSequence(args)
		if err != nil {
			return nil, err
		}
	}
	if extra.Gas == nil {
		gas := tokenCfg.DefaultGasLimit
		if gas == 0 {
			gas = DefaultGasLimit
		}
		extra.Gas = &gas
	}
	if extra.Fees == nil {
		fees := tokenCfg.DefaultFees
		if fees == "" {
			fees = DefaultFees
		}
		extra.Fees = &fees
	}
	return extra, nil
}

func (b *Bridge) getMinReserveFee() *big.Int {
	minReserveFee := b.ChainConfig.GetMinReserveFee()
	if minReserveFee == nil {
		minReserveFee = big.NewInt(0)
	}
	return minReserveFee
}

func (b *Bridge) getSequence(args *tokens.BuildTxArgs) (*uint64, error) {
	var sequence uint64
	var err error
	for i := 0; i < retryRPCCount; i++ {
		sequence, err = b.GetAccountSequence(args.From)
		if err == nil {
			break
		}
		time.Sleep(retryRPCInterval)
	}
	if err != nil {
		return nil, err
	}
	if args.SwapType != tokens.NoSwapType {
		tokenCfg := b.GetTokenConfig(args.PairID)
		if tokenCfg != nil && args.From == tokenCfg.DcrmAddress {
			sequence = b.AdjustNonce(args.PairID, sequence)
		}
	}
	return &sequence, nil
}

func getOrInitExtra(args *tokens.BuildTxArgs) *tokens.TerraExtra {
	if args.Extra == nil || args.Extra.TerraExtra == nil {
		args.Extra = &tokens.AllExtras{TerraExtra: &tokens.TerraExtra{}}
	}
	return args.Extra.TerraExtra
}

// GetPoolNonce impl NonceSetter interface
func (b *Bridge) GetPoolNonce(address, _height string) (uint64, error) {
	return b.GetAccountSequence(address)
}

// GetAccountSequence get account sequence
func (b *Bridge) GetAccountSequence(address string) (uint64, error) {
	acc, err := b.GetBaseAccount(address)
	if err == nil && acc != nil {
		return common.GetUint64FromStr(acc.Sequence)
	}
	return 0, wrapRPCQueryError(err, "GetAccountSequence")
}

// GetAccountNumber get account number
func (b *Bridge) GetAccountNumber(address string) (uint64, error) {
	acc, err := b.GetBaseAccount(address)
	if err == nil && acc != nil {
		return common.GetUint64FromStr(acc.AccountNumber)
	}
	return 0, wrapRPCQueryError(err, "GetAccountNumber")
}

func (b *Bridge) checkCoinBalance(account, denom string, amount *big.Int) error {
	bal, err := b.GetBalance(account, denom)
	if err != nil {
		return err
	}
	if bal.BigInt().Cmp(amount) < 0 {
		return fmt.Errorf(
			"insufficient native balance, sender: %v, %v < %v",
			account, bal, amount)
	}
	return nil
}

func (b *Bridge) checkTokenBalance(token, account string, amount *big.Int) error {
	bal, err := b.GetTokenBalance(token, account)
	if err != nil {
		return err
	}
	if bal.BigInt().Cmp(amount) < 0 {
		return fmt.Errorf(
			"insufficient %v balance, account: %v,  %v < %v",
			token, account, bal, amount)
	}
	return nil
}
