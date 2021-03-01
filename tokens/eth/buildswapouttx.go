package eth

import (
	"errors"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

func (b *Bridge) buildErc20SwapoutTxInput(args *tokens.BuildTxArgs) (err error) {
	pairID := args.PairID
	token := b.GetTokenConfig(pairID)
	if token == nil {
		return tokens.ErrUnknownPairID
	}
	funcHash := erc20CodeParts["transfer"]
	address := common.HexToAddress(args.Bind)
	if address == (common.Address{}) || !common.IsHexAddress(args.Bind) {
		log.Warn("swapout to wrong address", "address", args.Bind)
		return errors.New("can not swapout to empty or invalid address")
	}
	amount := tokens.CalcSwappedValue(pairID, args.OriginValue, false)

	input := PackDataWithFuncHash(funcHash, address, amount)
	args.Input = &input // input

	args.To = token.ContractAddress // to

	return b.checkBalance(token.ContractAddress, token.DcrmAddress, amount)
}
