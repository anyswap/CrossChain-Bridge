package rpcapi

import (
	"net/http"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/common/hexutil"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/eth"
	"github.com/anyswap/CrossChain-Bridge/types"
)

// BuildSwapoutTxArgs build swapout tx args
type BuildSwapoutTxArgs struct {
	From     common.Address  `json:"from"`
	Value    *hexutil.Big    `json:"value"`
	Bind     string          `json:"bind"`
	Gas      *hexutil.Uint64 `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	Nonce    *hexutil.Uint64 `json:"nonce"`
}

// BuildSwapoutTx build swapout tx
func (s *RPCAPI) BuildSwapoutTx(r *http.Request, args *BuildSwapoutTxArgs, result *types.Transaction) error {
	from := args.From.String()
	token, gateway := tokens.DstBridge.GetTokenAndGateway()
	contract := token.ContractAddress
	extraArgs := &tokens.EthExtraArgs{
		Gas:      (*uint64)(args.Gas),
		GasPrice: args.GasPrice.ToInt(),
		Nonce:    (*uint64)(args.Nonce),
	}
	swapoutVal := args.Value.ToInt()
	bindAddr := args.Bind

	ethBridge := eth.NewCrossChainBridge(false)
	ethBridge.TokenConfig = token
	ethBridge.GatewayConfig = gateway
	tx, err := ethBridge.BuildSwapoutTx(from, contract, extraArgs, swapoutVal, bindAddr)
	if err != nil {
		return err
	}
	*result = *tx
	return nil
}
