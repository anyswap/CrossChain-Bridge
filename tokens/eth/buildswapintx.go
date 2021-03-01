package eth

import (
	"errors"
	"math/big"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// build input for calling `Swapin(bytes32 txhash, address account, uint256 amount)`
func (b *Bridge) buildSwapinTxInput(args *tokens.BuildTxArgs) error {
	pairID := args.PairID
	funcHash := getSwapinFuncHash()
	txHash := common.HexToHash(args.SwapID)
	address := common.HexToAddress(args.Bind)
	if address == (common.Address{}) || !common.IsHexAddress(args.Bind) {
		log.Warn("swapin to wrong address", "address", args.Bind)
		return errors.New("can not swapin to empty or invalid address")
	}
	amount := tokens.CalcSwappedValue(pairID, args.OriginValue, true)

	input := PackDataWithFuncHash(funcHash, txHash, address, amount)
	args.Input = &input // input

	token := b.GetTokenConfig(pairID)
	if token == nil {
		return tokens.ErrUnknownPairID
	}
	args.To = token.ContractAddress // to

	if !token.IsDelegateContract {
		return nil
	}

	var balance *big.Int
	var err error
	for i := 0; i < retryRPCCount; i++ {
		if token.DelegateToken != "" {
			balance, err = b.GetErc20Balance(token.DelegateToken, token.ContractAddress)
		} else {
			balance, err = b.GetBalance(token.ContractAddress)
		}
		if err == nil {
			break
		}
		time.Sleep(retryRPCInterval)
	}
	if err == nil && balance.Cmp(amount) < 0 {
		return errors.New("not enough balance to swapin")
	}
	return err
}
