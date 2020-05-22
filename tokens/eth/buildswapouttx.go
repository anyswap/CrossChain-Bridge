package eth

import (
	"math/big"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/common/hexutil"
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	"github.com/fsn-dev/crossChain-Bridge/types"
)

func (b *EthBridge) BuildSwapoutTx(from, to string, extraArgs *tokens.EthExtraArgs, swapoutVal *big.Int, bindAddr string) (*types.Transaction, error) {
	input, err := BuildSwapoutTxInput(swapoutVal, bindAddr)
	if err != nil {
		return nil, err
	}
	args := &tokens.BuildTxArgs{
		From:  from,
		To:    to,
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
	bindAddress, swapoutvalue, err := ParseSwapoutTxInput(&input)
	if err != nil {
		log.Error("ParseSwapoutTxInput error", "err", err)
		return nil, err
	}
	log.Info("ParseSwapoutTxInput", "bindAddress", bindAddress, "swapoutvalue", swapoutvalue)

	return input, nil
}
