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

func (b *Bridge) verifySwapoutTx(txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	if allowUnstable {
		return b.verifySwapoutTxUnstable(txHash)
	}
	return b.verifySwapoutTxStable(txHash)
}

func (b *Bridge) verifySwapoutTxStable(txHash string) (*tokens.TxSwapInfo, error) {
	swapInfo := &tokens.TxSwapInfo{}
	swapInfo.Hash = txHash // Hash
	token := b.TokenConfig

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
	if receipt.Recipient != nil {
		swapInfo.To = strings.ToLower(receipt.Recipient.String()) // To
	}
	swapInfo.From = strings.ToLower(receipt.From.String()) // From

	contractAddress := token.ContractAddress
	if !common.IsEqualIgnoreCase(swapInfo.To, contractAddress) {
		return swapInfo, tokens.ErrTxWithWrongReceiver
	}

	bindAddress, value, err := parseSwapoutTxLogs(receipt.Logs)
	if err != nil {
		log.Debug(b.TokenConfig.BlockChain+" parseSwapoutTxLogs fail", "tx", txHash, "err", err)
		return swapInfo, tokens.ErrTxWithWrongInput
	}
	if bindAddress != "" {
		swapInfo.Bind = bindAddress // Bind
	} else {
		swapInfo.Bind = swapInfo.From // Bind
	}
	swapInfo.Value = value // Value

	// check sender
	// if common.IsEqualIgnoreCase(swapInfo.From, token.DcrmAddress) {
	// 	return swapInfo, tokens.ErrTxWithWrongSender
	// }

	if !tokens.CheckSwapValue(swapInfo.Value, b.IsSrc) {
		return swapInfo, tokens.ErrTxWithWrongValue
	}

	// NOTE: must verify memo at last step (as it can be recall)
	if !tokens.SrcBridge.IsValidAddress(swapInfo.Bind) {
		log.Debug("wrong bind address in swapout", "bind", swapInfo.Bind)
		return swapInfo, tokens.ErrTxWithWrongMemo
	}

	log.Debug("verify swapout stable pass", "from", swapInfo.From, "to", swapInfo.To, "bind", swapInfo.Bind, "value", swapInfo.Value, "txid", txHash, "height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
	return swapInfo, nil
}

func (b *Bridge) verifySwapoutTxUnstable(txHash string) (*tokens.TxSwapInfo, error) {
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
	if tx.Recipient != nil {
		swapInfo.To = strings.ToLower(tx.Recipient.String()) // To
	}
	swapInfo.From = strings.ToLower(tx.From.String()) // From

	token := b.TokenConfig
	contractAddress := token.ContractAddress
	if !common.IsEqualIgnoreCase(swapInfo.To, contractAddress) {
		return swapInfo, tokens.ErrTxWithWrongReceiver
	}

	input := (*[]byte)(tx.Payload)
	bindAddress, value, err := parseSwapoutTxInput(input)
	if err != nil {
		log.Debug(b.TokenConfig.BlockChain+" parseSwapoutTxInput fail", "tx", txHash, "err", err)
		return swapInfo, tokens.ErrTxWithWrongInput
	}
	if bindAddress != "" {
		swapInfo.Bind = bindAddress // Bind
	} else {
		swapInfo.Bind = swapInfo.From // Bind
	}
	swapInfo.Value = value // Value

	// check sender
	// if common.IsEqualIgnoreCase(swapInfo.From, token.DcrmAddress) {
	// 	return swapInfo, tokens.ErrTxWithWrongSender
	// }

	if !tokens.CheckSwapValue(swapInfo.Value, b.IsSrc) {
		return swapInfo, tokens.ErrTxWithWrongValue
	}

	if !tokens.SrcBridge.IsValidAddress(swapInfo.Bind) {
		log.Debug("wrong bind address in swapout", "bind", swapInfo.Bind)
		return swapInfo, tokens.ErrTxWithWrongMemo
	}

	return swapInfo, nil
}

func parseSwapoutTxInput(input *[]byte) (string, *big.Int, error) {
	if input == nil {
		return "", nil, fmt.Errorf("empty tx input")
	}
	data := *input
	if len(data) < 4 {
		return "", nil, fmt.Errorf("wrong tx input %x", data)
	}
	funcHash := data[:4]
	swapoutFuncHash := getSwapoutFuncHash()
	if !bytes.Equal(funcHash, swapoutFuncHash) {
		return "", nil, fmt.Errorf("wrong func hash, have %x want %x", funcHash, swapoutFuncHash)
	}
	encData := data[4:]
	return parseEncodedData(encData)
}

func parseSwapoutTxLogs(logs []*types.RPCLog) (string, *big.Int, error) {
	logSwapoutTopic := getLogSwapoutTopic()
	for _, log := range logs {
		if log.Removed != nil && *log.Removed {
			continue
		}
		if len(log.Topics) != 2 || log.Data == nil {
			continue
		}
		if !bytes.Equal(log.Topics[0].Bytes(), logSwapoutTopic) {
			continue
		}
		return parseEncodedData(*log.Data)
	}
	return "", nil, fmt.Errorf("swapout log not found or removed")
}

func parseEncodedData(encData []byte) (string, *big.Int, error) {
	isMbtc := isMbtcSwapout()
	if isMbtc {
		if len(encData) < 96 {
			return "", nil, fmt.Errorf("wrong length of encoded data")
		}
	} else {
		if len(encData) != 32 {
			return "", nil, fmt.Errorf("wrong length of encoded data")
		}
	}

	// get value
	value := common.GetBigInt(encData, 0, 32)
	if !isMbtc {
		return "", value, nil
	}

	// get bind address
	offset, overflow := common.GetUint64(encData, 32, 32)
	if overflow {
		return "", nil, fmt.Errorf("string offset overflow")
	}
	length, overflow := common.GetUint64(encData, offset, 32)
	if overflow {
		return "", nil, fmt.Errorf("string length overflow")
	}
	bind := string(common.GetData(encData, offset+32, length))
	return bind, value, nil
}
