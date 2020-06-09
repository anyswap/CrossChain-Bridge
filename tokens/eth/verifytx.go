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

func (b *EthBridge) GetTransaction(txHash string) (interface{}, error) {
	return b.GetTransactionByHash(txHash)
}

func (b *EthBridge) GetTransactionStatus(txHash string) *tokens.TxStatus {
	var txStatus tokens.TxStatus
	txr, err := b.GetTransactionReceipt(txHash)
	if err != nil {
		log.Debug("GetTransactionReceipt fail", "hash", txHash, "err", err)
		return &txStatus
	}
	if *txr.Status != 1 {
		log.Debug("transaction with wrong receipt status", "hash", txHash, "status", txr.Status)
	}
	txStatus.Block_height = txr.BlockNumber.ToInt().Uint64()
	txStatus.Block_hash = txr.BlockHash.String()
	block, err := b.GetBlockByHash(txStatus.Block_hash)
	if err == nil {
		txStatus.Block_time = block.Time.ToInt().Uint64()
	} else {
		log.Debug("GetBlockByHash fail", "hash", txStatus.Block_hash, "err", err)
	}
	if *txr.Status == 1 {
		latest, err := b.GetLatestBlockNumber()
		if err == nil {
			txStatus.Confirmations = latest - txStatus.Block_height
		} else {
			log.Debug("GetLatestBlockNumber fail", "err", err)
		}
	}
	txStatus.Receipt = txr
	return &txStatus
}

func (b *EthBridge) VerifyMsgHash(rawTx interface{}, msgHash string, extra interface{}) error {
	tx, ok := rawTx.(*types.Transaction)
	if !ok {
		return tokens.ErrWrongRawTx
	}
	signer := b.Signer
	sigHash := signer.Hash(tx)
	if sigHash.String() != msgHash {
		return tokens.ErrMsgHashMismatch
	}
	return nil
}

func (b *EthBridge) VerifyTransaction(txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	if !b.IsSrc {
		return b.verifySwapoutTx(txHash, allowUnstable)
	}
	return b.verifySwapinTx(txHash, allowUnstable)
}

func (b *EthBridge) verifySwapoutTx(txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	if allowUnstable {
		return b.verifySwapoutTxUnstable(txHash)
	}
	return b.verifySwapoutTxStable(txHash)
}

func (b *EthBridge) verifySwapoutTxStable(txHash string) (*tokens.TxSwapInfo, error) {
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
	if receipt.Recipient != nil {
		swapInfo.To = strings.ToLower(receipt.Recipient.String()) // To
	}
	swapInfo.From = strings.ToLower(receipt.From.String()) // From

	contractAddress := token.ContractAddress
	if !common.IsEqualIgnoreCase(swapInfo.To, contractAddress) {
		return swapInfo, tokens.ErrTxWithWrongReceiver
	}

	bindAddress, value, err := ParseSwapoutTxLogs(receipt.Logs)
	if err != nil {
		log.Debug("EthBridge ParseSwapoutTxLogs fail", "tx", txHash, "err", err)
		return swapInfo, tokens.ErrTxWithWrongInput
	}
	swapInfo.Bind = bindAddress // Bind
	swapInfo.Value = value      // Value

	// check sender
	if common.IsEqualIgnoreCase(swapInfo.From, dcrmAddress) {
		return swapInfo, tokens.ErrTxWithWrongSender
	}

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

func (b *EthBridge) verifySwapoutTxUnstable(txHash string) (*tokens.TxSwapInfo, error) {
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
	if tx.Recipient != nil {
		swapInfo.To = strings.ToLower(tx.Recipient.String()) // To
	}
	swapInfo.From = strings.ToLower(tx.From.String()) // From

	token := b.TokenConfig
	contractAddress := token.ContractAddress
	if !common.IsEqualIgnoreCase(swapInfo.To, contractAddress) {
		return swapInfo, tokens.ErrTxWithWrongReceiver
	}

	dcrmAddress := token.DcrmAddress
	if common.IsEqualIgnoreCase(swapInfo.From, dcrmAddress) {
		return swapInfo, tokens.ErrTxWithWrongSender
	}

	input := (*[]byte)(tx.Payload)
	bindAddress, value, err := ParseSwapoutTxInput(input)
	if err != nil {
		log.Debug("EthBridge ParseSwapoutTxInput fail", "tx", txHash, "input", input, "err", err)
		return swapInfo, tokens.ErrTxWithWrongInput
	}
	swapInfo.Bind = bindAddress // Bind
	swapInfo.Value = value      // Value

	if !tokens.CheckSwapValue(swapInfo.Value, b.IsSrc) {
		return swapInfo, tokens.ErrTxWithWrongValue
	}

	if !tokens.SrcBridge.IsValidAddress(swapInfo.Bind) {
		log.Debug("wrong bind address in swapout", "bind", swapInfo.Bind)
		return swapInfo, tokens.ErrTxWithWrongMemo
	}

	return swapInfo, nil
}

func ParseSwapoutTxInput(input *[]byte) (string, *big.Int, error) {
	if input == nil {
		return "", nil, fmt.Errorf("empty tx input")
	}
	data := *input
	if len(data) < 4 {
		return "", nil, fmt.Errorf("wrong tx input %x", data)
	}
	funcHash := data[:4]
	if !bytes.Equal(funcHash, tokens.SwapoutFuncHash[:]) {
		return "", nil, fmt.Errorf("wrong func hash, have %x want %x", funcHash, tokens.SwapoutFuncHash)
	}
	encData := data[4:]
	return ParseEncodedData(encData)
}

func ParseSwapoutTxLogs(logs []*types.RPCLog) (string, *big.Int, error) {
	for _, log := range logs {
		if log.Removed != nil && *log.Removed {
			continue
		}
		if len(log.Topics) != 2 {
			continue
		}
		if log.Topics[0].String() != tokens.LogSwapoutTopic {
			continue
		}
		if log.Data != nil {
			data := ([]byte)(*log.Data)
			return ParseEncodedData(data)
		}
	}
	return "", nil, fmt.Errorf("swapout log not found or removed")
}

func ParseEncodedData(encData []byte) (string, *big.Int, error) {
	if len(encData) < 96 {
		return "", nil, fmt.Errorf("wrong lenght of encoded data")
	}
	value := common.GetBigInt(encData, 0, 32)
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

func (b *EthBridge) verifySwapinTx(txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	swapInfo := &tokens.TxSwapInfo{}
	swapInfo.Hash = txHash // Hash
	token := b.TokenConfig
	dcrmAddress := token.DcrmAddress

	tx, err := b.GetTransactionByHash(txHash)
	if err != nil {
		log.Debug("EthBridge::GetTransaction fail", "tx", txHash, "err", err)
		return swapInfo, tokens.ErrTxNotFound
	}
	if tx.BlockNumber != nil {
		swapInfo.Height = tx.BlockNumber.ToInt().Uint64() // Height
	}
	if tx.Recipient != nil {
		swapInfo.To = strings.ToLower(tx.Recipient.String()) // To
	}
	swapInfo.From = strings.ToLower(tx.From.String()) // From
	swapInfo.Bind = swapInfo.From                     // Bind
	swapInfo.Value = tx.Amount.ToInt()                // Value

	if !allowUnstable {
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
	}

	if !common.IsEqualIgnoreCase(swapInfo.To, dcrmAddress) {
		return swapInfo, tokens.ErrTxWithWrongReceiver
	}

	// check sender
	if common.IsEqualIgnoreCase(swapInfo.From, dcrmAddress) {
		return swapInfo, tokens.ErrTxWithWrongSender
	}

	if !tokens.CheckSwapValue(swapInfo.Value, b.IsSrc) {
		return swapInfo, tokens.ErrTxWithWrongValue
	}

	log.Debug("verify swapout stable pass", "from", swapInfo.From, "to", swapInfo.To, "bind", swapInfo.Bind, "value", swapInfo.Value, "txid", txHash, "height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
	return swapInfo, nil
}
