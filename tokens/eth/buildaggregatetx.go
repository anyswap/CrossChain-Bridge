package eth

import (
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/dcrm"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// BuildSpendFromBip32Tx build tx spending from bip32 address
func (b *Bridge) BuildSpendFromBip32Tx(from, to, inputCode string, extra *tokens.EthExtraArgs) (rawTx interface{}, err error) {
	if extra == nil || extra.AggregateValue == nil || extra.AggregateValue.Sign() <= 0 {
		return nil, fmt.Errorf("build spend bip32 tx with zero value")
	}
	value := extra.AggregateValue
	args := &tokens.BuildTxArgs{
		From:      from,
		To:        to,
		Value:     value,
		InputCode: inputCode,
		Extra: &tokens.AllExtras{
			EthExtra: extra,
		},
	}
	return b.BuildRawTransaction(args)
}

// BuildAggregateTransaction build aggregate tx
// `args` must include: PairID, Bind, AggregateValue
func (b *Bridge) BuildAggregateTransaction(args *tokens.BuildTxArgs) (rawTx interface{}, err error) {
	if args == nil || args.Extra == nil || args.Extra.EthExtra == nil {
		return nil, fmt.Errorf("aggregate: empty eth extra")
	}

	extra := args.Extra.EthExtra
	value := extra.AggregateValue
	if value == nil || value.Sign() <= 0 {
		return nil, fmt.Errorf("aggregate: zero value")
	}

	inputCode, err := b.GetBip32InputCode(args.Bind)
	if err != nil {
		return nil, err
	}

	pairID := args.PairID
	rootPubkey := b.GetDcrmPublicKey(pairID)
	childPubkey, err := dcrm.GetBip32ChildKey(rootPubkey, inputCode)
	if err != nil {
		return nil, err
	}

	bip32Addr, err := b.PublicKeyToAddress(childPubkey)
	if err != nil {
		return nil, err
	}

	token := b.GetTokenConfig(pairID)
	to := token.DcrmAddress
	return b.BuildSpendFromBip32Tx(bip32Addr, to, inputCode, extra)
}

// VerifyAggregateMsgHash verify aggregate msgHash
func (b *Bridge) VerifyAggregateMsgHash(msgHash []string, args *tokens.BuildTxArgs) error {
	rawTx, err := b.BuildAggregateTransaction(args)
	if err != nil {
		return err
	}
	return b.VerifyMsgHash(rawTx, msgHash)
}
