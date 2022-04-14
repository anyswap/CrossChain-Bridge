package terra

import (
	"fmt"
	"math/big"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

var (
	retryRPCCount    = 3
	retryRPCInterval = 1 * time.Second

	signMode = signing.SignMode_SIGN_MODE_DIRECT

	DefaultGasLimit = uint64(200000)
	DefaultFees     = "10000uluna"
)

// BuildRawTransaction build raw tx
func (b *Bridge) BuildRawTransaction(args *tokens.BuildTxArgs) (rawTx interface{}, err error) {
	pairID := args.PairID
	tokenCfg := b.GetTokenConfig(pairID)
	if tokenCfg == nil {
		return nil, fmt.Errorf("swap pair '%v' is not configed", pairID)
	}
	if tokenCfg.DcrmAccountNumber == 0 {
		tokenCfg.DcrmAccountNumber, err = b.GetAccountNumber(tokenCfg.DcrmAddress)
		return nil, fmt.Errorf("init dcrm account number failed: %w", err)
	}

	var from, to, memo string
	var amount *big.Int

	switch args.SwapType {
	case tokens.SwapinType:
		return nil, tokens.ErrSwapTypeNotSupported
	case tokens.SwapoutType:
		from = tokenCfg.DcrmAddress // from
		to = args.Bind              // to

		amount = tokens.CalcSwappedValue(pairID, args.OriginValue, false, args.OriginFrom, args.OriginTxTo) // amount
		memo = tokens.UnlockMemoPrefix + args.SwapID
	default:
		return nil, tokens.ErrUnknownSwapType
	}

	if from == "" {
		return nil, tokens.ErrTxWithWrongSender
	}

	if amount.Sign() < 0 {
		return nil, fmt.Errorf("negative token amount")
	}

	neededValue := b.getMinReserveFee()
	err = b.checkCoinBalance(from, b.ChainConfig.MetaCoin.Unit, neededValue)
	if err != nil {
		return nil, err
	}

	err = b.checkTokenBalance(tokenCfg.ContractAddress, from, amount)
	if err != nil {
		return nil, err
	}

	extra, err := b.initExtra(args, tokenCfg)
	if err != nil {
		return nil, err
	}

	return b.BuildTx(from, to, memo, amount, extra, tokenCfg)
}

// BuildTx build tx
func (b *Bridge) BuildTx(
	from, to, memo string,
	amount *big.Int,
	extra *tokens.TerraExtra,
	tokenCfg *tokens.TokenConfig,
) (*wrapper, error) {
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

	feePayer, err := sdk.AccAddressFromBech32(from)
	if err != nil {
		return nil, err
	}
	txb.SetFeePayer(feePayer)

	accFrom, err := sdk.AccAddressFromBech32(from)
	if err != nil {
		return nil, err
	}
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
	if err == nil {
		return nil, err
	}

	if params.IsSwapServer {
		err = b.simulateTx(txb)
		if err != nil {
			return nil, err
		}
	}

	return txb, nil
}

func (b *Bridge) simulateTx(txb *wrapper) error {
	txBytes, err := txb.GetTxBytes()
	if err != nil {
		return err
	}
	simRes, err := b.SimulateTx(&SimulateRequest{TxBytes: txBytes})
	if err != nil {
		return err
	}
	gasUsed, err := common.GetUint64FromStr(simRes.GasInfo.GasUsed)
	if err != nil {
		return err
	}
	gasWanted := txb.GetGas()
	if gasWanted < gasUsed {
		return fmt.Errorf("simulate tx gas exceeded, wanted %v used %v", gasWanted, gasUsed)
	}
	return nil
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
