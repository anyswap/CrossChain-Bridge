package eth

import (
	"fmt"
	"math/big"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/common/hexutil"
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/rpc/client"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	"github.com/fsn-dev/crossChain-Bridge/types"
)

// BuildSwapoutTx build swapout tx
func (b *Bridge) BuildSwapoutTx(from, contract string, extraArgs *tokens.EthExtraArgs, swapoutVal *big.Int, bindAddr string) (*types.Transaction, error) {
	if swapoutVal == nil || swapoutVal.Sign() == 0 {
		return nil, fmt.Errorf("swapout value must be greater than zero")
	}
	balance, err := b.GetMBtcBalance(contract, from)
	if err != nil {
		return nil, err
	}
	if balance.Cmp(swapoutVal) < 0 {
		return nil, fmt.Errorf("not enough balance, %v < %v", balance, swapoutVal)
	}
	token := b.TokenConfig
	if token != nil && !tokens.CheckSwapValue(swapoutVal, b.IsSrc) {
		decimals := *token.Decimals
		minValue := tokens.ToBits(*token.MinimumSwap, decimals)
		maxValue := tokens.ToBits(*token.MaximumSwap, decimals)
		return nil, fmt.Errorf("wrong swapout value, not in range [%v, %v]", minValue, maxValue)
	}
	if tokens.SrcBridge != nil && !tokens.SrcBridge.IsValidAddress(bindAddr) {
		return nil, fmt.Errorf("wrong swapout bind address %v", bindAddr)
	}
	input, err := BuildSwapoutTxInput(swapoutVal, bindAddr)
	if err != nil {
		return nil, err
	}
	args := &tokens.BuildTxArgs{
		From:  from,
		To:    contract,
		Value: big.NewInt(0),
		Input: &input,
	}
	if extraArgs != nil {
		args.Extra = &tokens.AllExtras{
			EthExtra: extraArgs,
		}
	}
	rawtx, err := b.BuildRawTransaction(args)
	if err != nil {
		return nil, err
	}
	tx, ok := rawtx.(*types.Transaction)
	if !ok {
		return nil, tokens.ErrWrongRawTx
	}
	return tx, nil
}

// BuildSwapoutTxInput build swapout tx input
func BuildSwapoutTxInput(swapoutVal *big.Int, bindAddr string) ([]byte, error) {
	strLen := len(bindAddr)

	// encode params
	bs := make(hexutil.Bytes, 0)
	bs = append(bs, common.LeftPadBytes(swapoutVal.Bytes(), 32)...)
	bs = append(bs, common.LeftPadBytes(big.NewInt(int64(len(bs)+32)).Bytes(), 32)...)
	bs = append(bs, common.LeftPadBytes(big.NewInt(int64(strLen)).Bytes(), 32)...)

	lastPad := strLen - strLen%32
	bs = append(bs, bindAddr[:lastPad]...)
	if lastPad != strLen {
		bs = append(bs, common.RightPadBytes([]byte(bindAddr[lastPad:]), 32)...)
	}

	// add func hash
	input := make([]byte, len(bs)+4)
	copy(input[:4], tokens.SwapoutFuncHash[:])
	copy(input[4:], bs)

	// verify input
	bindAddress, swapoutvalue, err := parseSwapoutTxInput(&input)
	if err != nil {
		log.Error("parseSwapoutTxInput error", "err", err)
		return nil, err
	}
	log.Info("parseSwapoutTxInput", "bindAddress", bindAddress, "swapoutvalue", swapoutvalue)

	return input, nil
}

// GetMBtcBalance get mbtc balacne
func (b *Bridge) GetMBtcBalance(contract string, address string) (*big.Int, error) {
	balanceOfFuncHash := common.FromHex("0x70a08231")
	addr := common.HexToAddress(address)
	data := make(hexutil.Bytes, 36)
	copy(data[:4], balanceOfFuncHash)
	copy(data[4:], common.LeftPadBytes(addr[:], 32))
	reqArgs := map[string]interface{}{
		"to":   contract,
		"data": data,
	}
	var result string
	url := b.GatewayConfig.APIAddress
	err := client.RPCPost(&result, url, "eth_call", reqArgs, "pending")
	if err != nil {
		return nil, err
	}
	return common.GetBigIntFromStr(result)
}
