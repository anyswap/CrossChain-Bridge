package eth

import (
	"bytes"
	"fmt"
	"math/big"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/types"
)

func (b *Bridge) verifyErc20SwapinTx(txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	if allowUnstable {
		return b.verifyErc20SwapinTxUnstable(txHash)
	}
	return b.verifyErc20SwapinTxStable(txHash)
}

func (b *Bridge) verifyErc20SwapinTxStable(txHash string) (*tokens.TxSwapInfo, error) {
	swapInfo := &tokens.TxSwapInfo{}
	swapInfo.Hash = txHash // Hash
	token := b.TokenConfig
	dcrmAddress := token.DcrmAddress

	txStatus := b.GetTransactionStatus(txHash)
	swapInfo.Height = txStatus.BlockHeight  // Height
	swapInfo.Timestamp = txStatus.BlockTime // Timestamp
	receipt, ok := txStatus.Receipt.(*types.RPCTxReceipt)
	if !ok || receipt == nil {
		return swapInfo, tokens.ErrTxNotStable
	}
	if *receipt.Status != 1 {
		return swapInfo, tokens.ErrTxWithWrongReceipt
	}
	if txStatus.BlockHeight == 0 ||
		txStatus.Confirmations < *token.Confirmations {
		return swapInfo, tokens.ErrTxNotStable
	}
	swapInfo.From = strings.ToLower(receipt.From.String()) // From

	contractAddress := token.ContractAddress
	if receipt.Recipient == nil ||
		!common.IsEqualIgnoreCase(receipt.Recipient.String(), contractAddress) {
		return swapInfo, tokens.ErrTxWithWrongContract
	}

	from, to, value, err := parseErc20SwapinTxLogs(receipt.Logs)
	if err != nil {
		log.Debug(b.TokenConfig.BlockChain+" parseErc20SwapinTxLogs failed", "err", err)
		return swapInfo, tokens.ErrTxWithWrongInput
	}
	swapInfo.To = strings.ToLower(to)     // To
	swapInfo.Value = value                // Value
	swapInfo.Bind = strings.ToLower(from) // Bind

	if !common.IsEqualIgnoreCase(swapInfo.To, dcrmAddress) {
		return swapInfo, tokens.ErrTxWithWrongReceiver
	}

	// check sender
	if swapInfo.From == swapInfo.To {
		return swapInfo, tokens.ErrTxWithWrongSender
	}

	if !tokens.CheckSwapValue(swapInfo.Value, b.IsSrc) {
		return swapInfo, tokens.ErrTxWithWrongValue
	}

	log.Debug("verify erc20 swapin pass", "from", swapInfo.From, "to", swapInfo.To, "bind", swapInfo.Bind, "value", swapInfo.Value, "txid", txHash, "height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
	return swapInfo, nil
}

func (b *Bridge) verifyErc20SwapinTxUnstable(txHash string) (*tokens.TxSwapInfo, error) {
	swapInfo := &tokens.TxSwapInfo{}
	swapInfo.Hash = txHash // Hash
	tx, err := b.GetTransactionByHash(txHash)
	if err != nil {
		log.Debug(b.TokenConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txHash, "err", err)
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
	from, to, value, err := parseErc20SwapinTxInput(input)
	if err != nil {
		log.Debug(b.TokenConfig.BlockChain+" parseErc20SwapinTxInput failed", "err", err)
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

	// check sender
	if swapInfo.From == swapInfo.To {
		return swapInfo, tokens.ErrTxWithWrongSender
	}

	if !tokens.CheckSwapValue(swapInfo.Value, b.IsSrc) {
		return swapInfo, tokens.ErrTxWithWrongValue
	}

	return swapInfo, nil
}

func parseErc20SwapinTxInput(input *[]byte) (from, to string, value *big.Int, err error) {
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
	case bytes.Equal(funcHash, erc20CodeParts["transfer"]):
	case bytes.Equal(funcHash, erc20CodeParts["transferFrom"]):
		isTransferFrom = true
	default:
		return "", "", nil, fmt.Errorf("func hash of Erc20 Transfer is not found")
	}
	encData := data[4:]
	return parseErc20EncodedData(encData, isTransferFrom)
}

func parseErc20SwapinTxLogs(logs []*types.RPCLog) (from, to string, value *big.Int, err error) {
	for _, log := range logs {
		if log.Removed != nil && *log.Removed {
			continue
		}
		if len(log.Topics) != 3 || log.Data == nil {
			continue
		}
		if !bytes.Equal(log.Topics[0][:], erc20CodeParts["LogTransfer"]) {
			continue
		}
		from = common.BytesToAddress(log.Topics[1][:]).String()
		to = common.BytesToAddress(log.Topics[2][:]).String()
		value = new(big.Int).SetBytes(*log.Data)
		return from, to, value, nil
	}
	return "", "", nil, fmt.Errorf("log of Erc20 Transfer is not found or removed")
}

func parseErc20EncodedData(encData []byte, isTransferFrom bool) (from, to string, value *big.Int, err error) {
	if isTransferFrom {
		from = common.BytesToAddress(common.GetData(encData, 0, 32)).String()
		encData = encData[32:]
	}
	if len(encData) != 64 {
		return "", "", nil, fmt.Errorf("wrong length of encoded data")
	}
	to = common.BytesToAddress(common.GetData(encData, 0, 32)).String()
	value = common.GetBigInt(encData, 32, 32)
	return from, to, value, nil
}
