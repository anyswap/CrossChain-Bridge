package eth

import (
	"errors"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// build input for calling `Swapin(bytes32 txhash, address account, uint256 amount)`
func (b *Bridge) buildSwapinTxInput(args *tokens.BuildTxArgs) error {
	pairID := args.PairID
	token := b.GetTokenConfig(pairID)
	if token == nil {
		return tokens.ErrUnknownPairID
	}
	funcHash := getSwapinFuncHash()
	txHash := common.HexToHash(args.SwapID)
	address := common.HexToAddress(args.Bind)
	if address == (common.Address{}) || !common.IsHexAddress(args.Bind) {
		log.Warn("swapin to wrong address", "address", args.Bind)
		return errors.New("can not swapin to empty or invalid address")
	}
	amount := tokens.CalcSwappedValue(pairID, args.OriginValue, true, args.OriginFrom, args.OriginTxTo) // amount

	input := PackDataWithFuncHash(funcHash, txHash, address, amount)
	args.Input = &input // input

	args.To = token.ContractAddress // to

	if token.IsDelegateContract && !token.IsAnyswapAdapter {
		return b.checkBalance(token.DelegateToken, token.ContractAddress, amount)
	}
	return nil
}
