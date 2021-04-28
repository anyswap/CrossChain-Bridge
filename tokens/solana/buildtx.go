package solana

import (
	"errors"
	"fmt"

	bin "github.com/dfuse-io/binary"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/programs/system"

	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// BuildRawTransaction build raw tx
func (b *Bridge) BuildRawTransaction(args *tokens.BuildTxArgs) (rawTx interface{}, err error) {
	var (
		pairID = args.PairID
		from   = args.From
		to     = args.Bind
		amount = args.Value
	)
	args.Identifier = params.GetIdentifier()
	tokenCfg := b.GetTokenConfig(pairID)
	if tokenCfg == nil {
		return nil, fmt.Errorf("swap pair '%v' is not configed", pairID)
	}
	switch args.SwapType {
	case tokens.SwapinType:
		return nil, tokens.ErrSwapTypeNotSupported
	case tokens.SwapoutType:
		from = tokenCfg.DcrmAddress                                       // from
		amount = tokens.CalcSwappedValue(pairID, args.OriginValue, false) // amount
	}

	if from == "" {
		return nil, errors.New("no sender specified")
	}

	fromPubkey, err := solana.PublicKeyFromBase58(from)
	if err != nil {
		return nil, errors.New("from address error")
	}
	toPubkey, err := solana.PublicKeyFromBase58(to)
	if err != nil {
		return nil, errors.New("to address error")
	}
	lamports := amount.Uint64()
	transfer := &system.Instruction{
		BaseVariant: bin.BaseVariant{
			TypeID: 2, // 0 表示 create account，1 空缺，2 表示 transfer
			Impl: &system.Transfer{
				Lamports: bin.Uint64(lamports),
				Accounts: &system.TransferAccounts{
					From: &solana.AccountMeta{PublicKey: fromPubkey, IsSigner: true, IsWritable: true},
					To:   &solana.AccountMeta{PublicKey: toPubkey, IsSigner: false, IsWritable: true},
				},
			},
		},
	}
	rbh, err := b.GetRecentBlockhash()
	if err != nil {
		return nil, errors.New("get recent blockhash error")
	}
	blockHash, err := solana.PublicKeyFromBase58(rbh)
	if err != nil {
		return nil, errors.New("get recent blockhash error")
	}
	opt := &solana.Options{
		Payer: fromPubkey,
	}
	tx, err := solana.TransactionWithInstructions([]solana.TransactionInstruction{transfer}, blockHash, opt)
	if err != nil {
		return nil, fmt.Errorf("Solana transaction with instructions error: %v", err)
	}
	return tx, nil
}
