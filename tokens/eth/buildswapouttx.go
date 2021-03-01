package eth

import (
	"errors"
	"math/big"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

func (b *Bridge) buildErc20SwapoutTxInput(args *tokens.BuildTxArgs) (err error) {
	pairID := args.PairID
	funcHash := erc20CodeParts["transfer"]
	address := common.HexToAddress(args.Bind)
	if address == (common.Address{}) || !common.IsHexAddress(args.Bind) {
		log.Warn("swapout to wrong address", "address", args.Bind)
		return errors.New("can not swapout to empty or invalid address")
	}
	amount := tokens.CalcSwappedValue(pairID, args.OriginValue, false)

	input := PackDataWithFuncHash(funcHash, address, amount)
	args.Input = &input // input

	token := b.GetTokenConfig(pairID)
	if token == nil {
		return tokens.ErrUnknownPairID
	}
	args.To = token.ContractAddress // to

	var balance *big.Int
	for i := 0; i < retryRPCCount; i++ {
		balance, err = b.GetErc20Balance(token.ContractAddress, token.DcrmAddress)
		if err == nil {
			break
		}
		time.Sleep(retryRPCInterval)
	}
	if err == nil && balance.Cmp(amount) < 0 {
		return errors.New("not enough token balance to swapout")
	}
	return err
}
