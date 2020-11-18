package eth

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/dcrm"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// BuildAggregateTransaction build aggregate tx
// `args` must include: PairID, Bind, AggregateValue
func (b *Bridge) BuildAggregateTransaction(args *tokens.BuildTxArgs) (rawTx interface{}, err error) {
	err = b.checkAndFillAggregateArgs(args)
	if err != nil {
		return nil, err
	}
	token := b.GetTokenConfig(args.PairID)
	contractAddr := token.ContractAddress
	if contractAddr != "" {
		amount := args.Value
		input := PackDataWithFuncHash(
			erc20CodeParts["transfer"],
			common.HexToAddress(token.DcrmAddress),
			amount)
		args.Input = &input
		args.To = contractAddr
		args.Value = big.NewInt(0)
		err = b.checkTokenBalance(contractAddr, args.From, amount)
		if err != nil {
			return nil, err
		}
	} else {
		args.To = token.DcrmAddress
	}
	return b.BuildRawTransaction(args)
}

// VerifyAggregateMsgHash verify aggregate msgHash
func (b *Bridge) VerifyAggregateMsgHash(msgHash []string, args *tokens.BuildTxArgs) error {
	rawTx, err := b.BuildAggregateTransaction(args)
	if err != nil {
		return err
	}
	return b.VerifyMsgHash(rawTx, msgHash)
}

func (b *Bridge) checkAndFillAggregateArgs(args *tokens.BuildTxArgs) error {
	if args == nil || args.Extra == nil || args.Extra.EthExtra == nil {
		return fmt.Errorf("aggregate: empty eth extra")
	}

	extra := args.Extra.EthExtra
	value := extra.AggregateValue
	if value == nil || value.Sign() <= 0 {
		return fmt.Errorf("aggregate: zero value")
	}

	inputCode, err := b.GetBip32InputCode(args.Bind)
	if err != nil {
		return err
	}

	pairID := args.PairID
	rootPubkey := b.GetDcrmPublicKey(pairID)
	childPubkey, err := dcrm.GetBip32ChildKey(rootPubkey, inputCode)
	if err != nil {
		return err
	}

	bip32Addr, err := b.PublicKeyToAddress(childPubkey)
	if err != nil {
		return err
	}

	if args.From != "" && !strings.EqualFold(bip32Addr, args.From) {
		return fmt.Errorf("aggregate: sender mismatch. have %v, want %v. pairID is %v, input code is %v, root public key is %v", args.From, bip32Addr, pairID, inputCode, rootPubkey)
	}

	// fill args
	args.InputCode = inputCode
	args.Value = value
	if args.From == "" {
		args.From = bip32Addr
	}
	return nil
}
