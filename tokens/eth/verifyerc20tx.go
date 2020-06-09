package eth

import (
	"bytes"
	"fmt"
	"math/big"
	"strings"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	"github.com/fsn-dev/crossChain-Bridge/types"
)

var (
	Erc20TrasferFuncHash     = common.FromHex("0xa9059cbb")
	Erc20TrasferFromFuncHash = common.FromHex("0x23b872dd")
	Erc20TrasferLogTopic     = common.FromHex("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
)

func (b *EthBridge) verifyErc20SwapinTx(txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	if allowUnstable {
		return b.verifyErc20SwapinTxUnstable(txHash)
	}
	return b.verifyErc20SwapinTxStable(txHash)
}

func (b *EthBridge) verifyErc20SwapinTxStable(txHash string) (*tokens.TxSwapInfo, error) {
	swapInfo := &tokens.TxSwapInfo{}
	swapInfo.Hash = txHash // Hash
	token := b.TokenConfig
	dcrmAddress := token.DcrmAddress

	txStatus := b.GetTransactionStatus(txHash)
	swapInfo.Height = txStatus.Block_height  // Height
	swapInfo.Timestamp = txStatus.Block_time // Timestamp
	receipt, ok := txStatus.Receipt.(*types.RPCTxReceipt)
	if !ok || receipt == nil || *receipt.Status != 1 {
		return swapInfo, tokens.ErrTxWithWrongReceipt
	}
	if txStatus.Block_height == 0 ||
		txStatus.Confirmations < *token.Confirmations {
		return swapInfo, tokens.ErrTxNotStable
	}
	swapInfo.From = strings.ToLower(receipt.From.String()) // From

	contractAddress := token.ContractAddress
	if receipt.Recipient == nil ||
		!common.IsEqualIgnoreCase(receipt.Recipient.String(), contractAddress) {
		return swapInfo, tokens.ErrTxWithWrongContract
	}

	from, to, value, err := ParseErc20SwapinTxLogs(receipt.Logs)
	if err != nil {
		return swapInfo, tokens.ErrTxWithWrongInput
	}
	swapInfo.To = strings.ToLower(to)     // To
	swapInfo.Value = value                // Value
	swapInfo.Bind = strings.ToLower(from) // Bind

	if !common.IsEqualIgnoreCase(swapInfo.To, dcrmAddress) {
		return swapInfo, tokens.ErrTxWithWrongReceiver
	}

	if common.IsEqualIgnoreCase(swapInfo.Bind, dcrmAddress) {
		return swapInfo, tokens.ErrTxWithWrongSender
	}

	if !tokens.CheckSwapValue(swapInfo.Value, b.IsSrc) {
		return swapInfo, tokens.ErrTxWithWrongValue
	}

	log.Debug("verify swapout stable pass", "from", swapInfo.From, "to", swapInfo.To, "bind", swapInfo.Bind, "value", swapInfo.Value, "txid", txHash, "height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
	return swapInfo, nil
}

func (b *EthBridge) verifyErc20SwapinTxUnstable(txHash string) (*tokens.TxSwapInfo, error) {
	swapInfo := &tokens.TxSwapInfo{}
	swapInfo.Hash = txHash // Hash
	tx, err := b.GetTransactionByHash(txHash)
	if err != nil {
		log.Debug("EthBridge::GetTransaction fail", "tx", txHash, "err", err)
		return swapInfo, tokens.ErrTxNotFound
	}
	if tx.BlockNumber != nil {
		swapInfo.Height = tx.BlockNumber.ToInt().Uint64() // Height
	}
	swapInfo.From = strings.ToLower(tx.From.String()) // From

	token := b.TokenConfig
	contractAddress := token.ContractAddress
	if tx.Recipient == nil || !common.IsEqualIgnoreCase(tx.Recipient.String(), contractAddress) {
		return swapInfo, tokens.ErrTxWithWrongContract
	}

	input := (*[]byte)(tx.Payload)
	from, to, value, err := ParseErc20SwapinTxInput(input)
	if err != nil {
		return swapInfo, tokens.ErrTxWithWrongInput
	}
	swapInfo.To = strings.ToLower(to) // To
	swapInfo.Value = value            // Value
	if from != "" {
		swapInfo.Bind = strings.ToLower(from) // Bind
	} else {
		swapInfo.Bind = swapInfo.From // Bind
	}

	dcrmAddress := token.DcrmAddress
	if !common.IsEqualIgnoreCase(swapInfo.To, dcrmAddress) {
		return swapInfo, tokens.ErrTxWithWrongReceiver
	}

	if common.IsEqualIgnoreCase(swapInfo.Bind, dcrmAddress) {
		return swapInfo, tokens.ErrTxWithWrongSender
	}

	if !tokens.CheckSwapValue(swapInfo.Value, b.IsSrc) {
		return swapInfo, tokens.ErrTxWithWrongValue
	}

	return swapInfo, nil
}

func ParseErc20SwapinTxInput(input *[]byte) (string, string, *big.Int, error) {
	if input == nil {
		return "", "", nil, fmt.Errorf("empty tx input")
	}
	data := *input
	if len(data) < 4 {
		return "", "", nil, fmt.Errorf("wrong tx input %x", data)
	}
	funcHash := data[:4]
	isTransferFrom := false
	switch {
	case bytes.Equal(funcHash, Erc20TrasferFuncHash):
	case bytes.Equal(funcHash, Erc20TrasferFromFuncHash):
		isTransferFrom = true
	default:
		return "", "", nil, fmt.Errorf("Erc20 Transfer func hash not found")
	}
	encData := data[4:]
	return ParseErc20EncodedData(encData, isTransferFrom)
}

func ParseErc20SwapinTxLogs(logs []*types.RPCLog) (string, string, *big.Int, error) {
	for _, log := range logs {
		if log.Removed != nil && *log.Removed {
			continue
		}
		if len(log.Topics) != 3 || log.Data == nil {
			continue
		}
		if bytes.Equal(log.Topics[0][:], Erc20TrasferLogTopic) {
			continue
		}
		from := common.BytesToAddress(log.Topics[1][:]).String()
		to := common.BytesToAddress(log.Topics[2][:]).String()
		value := new(big.Int).SetBytes(*log.Data)
		return from, to, value, nil
	}
	return "", "", nil, fmt.Errorf("Erc20 Transfer log not found or removed")
}

func ParseErc20EncodedData(encData []byte, isTransferFrom bool) (string, string, *big.Int, error) {
	from := ""
	if isTransferFrom {
		from = common.BytesToAddress(common.GetData(encData, 0, 32)).String()
		encData = encData[32:]
	}
	if len(encData) < 64 {
		return "", "", nil, fmt.Errorf("wrong lenght of encoded data")
	}
	to := common.BytesToAddress(common.GetData(encData, 0, 32)).String()
	value := common.GetBigInt(encData, 32, 32)
	return from, to, value, nil
}
