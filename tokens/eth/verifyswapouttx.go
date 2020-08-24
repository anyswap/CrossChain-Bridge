package eth

import (
	"bytes"
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
		log.Debug(b.ChainConfig.BlockChain+" parseSwapoutTxLogs fail", "tx", txHash, "err", err)
		return swapInfo, err
	}
	if bindAddress != "" {
		swapInfo.Bind = bindAddress // Bind
	} else {
		swapInfo.Bind = swapInfo.From // Bind
	}
	swapInfo.Value = value // Value

	if !tokens.CheckSwapValue(swapInfo.Value, b.IsSrc) {
		return swapInfo, tokens.ErrTxWithWrongValue
	}

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
		log.Debug(b.ChainConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txHash, "err", err)
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
		log.Debug(b.ChainConfig.BlockChain+" parseSwapoutTxInput fail", "tx", txHash, "err", err)
		return swapInfo, err
	}
	if bindAddress != "" {
		swapInfo.Bind = bindAddress // Bind
	} else {
		swapInfo.Bind = swapInfo.From // Bind
	}
	swapInfo.Value = value // Value

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
	if input == nil || len(*input) < 4 {
		return "", nil, tokens.ErrTxWithWrongInput
	}
	data := *input
	funcHash := data[:4]
	swapoutFuncHash := getSwapoutFuncHash()
	if !bytes.Equal(funcHash, swapoutFuncHash) {
		return "", nil, tokens.ErrTxFuncHashMismatch
	}
	encData := data[4:]
	return parseTxInputEncodedData(encData)
}

func parseSwapoutTxLogs(logs []*types.RPCLog) (bind string, value *big.Int, err error) {
	if isMbtcSwapout() {
		return parseSwapoutToBtcTxLogs(logs)
	}
	logSwapoutTopic := getLogSwapoutTopic()
	for _, log := range logs {
		if log.Removed != nil && *log.Removed {
			continue
		}
		if len(log.Topics) != 3 || log.Data == nil {
			continue
		}
		if !bytes.Equal(log.Topics[0].Bytes(), logSwapoutTopic) {
			continue
		}
		bind = common.BytesToAddress(log.Topics[2].Bytes()).String()
		value = common.GetBigInt(*log.Data, 0, 32)
		return bind, value, nil
	}
	return "", nil, tokens.ErrSwapoutLogNotFound
}

func parseSwapoutToBtcTxLogs(logs []*types.RPCLog) (bind string, value *big.Int, err error) {
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
		return parseSwapoutToBtcEncodedData(*log.Data, false)
	}
	return "", nil, tokens.ErrSwapoutLogNotFound
}

func parseTxInputEncodedData(encData []byte) (bind string, value *big.Int, err error) {
	if isMbtcSwapout() {
		return parseSwapoutToBtcEncodedData(encData, true)
	}

	if len(encData) != 64 {
		return "", nil, tokens.ErrTxIncompatible
	}

	// get value
	value = common.GetBigInt(encData, 0, 32)

	// get bind address
	bind = common.BytesToAddress(common.GetData(encData, 32, 32)).String()
	return bind, value, nil
}

func parseSwapoutToBtcEncodedData(encData []byte, isInTxInput bool) (bind string, value *big.Int, err error) {
	if isInTxInput {
		err = tokens.ErrTxWithWrongInput
	} else {
		err = tokens.ErrTxWithWrongLogData
	}

	encDataLength := uint64(len(encData))
	if encDataLength < 96 || encDataLength%32 != 0 {
		return "", nil, err
	}

	// get value
	value = common.GetBigInt(encData, 0, 32)

	// get bind address
	offset, overflow := common.GetUint64(encData, 32, 32)
	if overflow {
		return "", nil, err
	}
	if encDataLength < offset+32 {
		return "", nil, err
	}
	length, overflow := common.GetUint64(encData, offset, 32)
	if overflow {
		return "", nil, err
	}
	if encDataLength < offset+32+length || encDataLength >= offset+32+length+32 {
		return "", nil, err
	}
	bind = string(common.GetData(encData, offset+32, length))
	return bind, value, nil
}
