package eth

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	"github.com/fsn-dev/crossChain-Bridge/types"
)

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

func (b *EthBridge) VerifyMsgHash(rawTx interface{}, msgHash string) error {
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
	return nil, tokens.ErrTodo
}

func (b *EthBridge) verifySwapoutTx(txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	if allowUnstable {
		return b.verifySwapoutTxUnstable(txHash)
	}
	return b.verifySwapoutTxStable(txHash)
}

func (b *EthBridge) verifySwapoutTxStable(txHash string) (*tokens.TxSwapInfo, error) {
	token := b.TokenConfig
	txStatus := b.GetTransactionStatus(txHash)
	receipt, _ := txStatus.Receipt.(*types.RPCTxReceipt)
	if *receipt.Status != 1 {
		return nil, tokens.ErrTxWithWrongReceipt
	}
	if txStatus.Block_height == 0 ||
		txStatus.Confirmations < *token.Confirmations {
		return nil, tokens.ErrTxNotStable
	}

	contractAddress := token.ContractAddress
	to := receipt.Recipient
	if to == nil || !common.IsEqualIgnoreCase(to.String(), contractAddress) {
		return nil, tokens.ErrTxWithWrongReceiver
	}

	dcrmAddress := token.DcrmAddress
	from := receipt.From.String()
	if common.IsEqualIgnoreCase(from, dcrmAddress) {
		return nil, tokens.ErrTxWithWrongSender
	}

	bindAddress, value, err := ParseSwapoutTxLogs(receipt.Logs)
	if err != nil {
		log.Debug("EthBridge ParseSwapoutTxLogs fail", "tx", txHash, "err", err)
		return nil, tokens.ErrTxWithWrongInput
	}

	if !tokens.CheckSwapValue(value, b.IsSrc) {
		return nil, tokens.ErrTxWithWrongValue
	}

	if !tokens.SrcBridge.IsValidAddress(bindAddress) {
		log.Debug("wrong bind address in swapout", "bind", bindAddress)
		err = tokens.ErrTxWithWrongMemo
	}

	blockHeight := txStatus.Block_height
	blockTimestamp := txStatus.Block_time
	log.Debug("verify swapout stable pass", "from", from, "to", to, "bind", bindAddress, "value", value, "txid", txHash, "height", blockHeight, "timestamp", blockTimestamp)
	return &tokens.TxSwapInfo{
		Hash:      txHash,
		Height:    blockHeight,
		Timestamp: blockTimestamp,
		From:      from,
		To:        contractAddress,
		Bind:      bindAddress,
		Value:     value.String(),
	}, err
}

func (b *EthBridge) verifySwapoutTxUnstable(txHash string) (*tokens.TxSwapInfo, error) {
	tx, err := b.GetTransaction(txHash)
	if err != nil {
		log.Debug("EthBridge::GetTransaction fail", "tx", txHash, "err", err)
		return nil, tokens.ErrTxNotFound
	}

	token := b.TokenConfig
	contractAddress := token.ContractAddress
	to := tx.Recipient
	if to == nil || !common.IsEqualIgnoreCase(to.String(), contractAddress) {
		return nil, tokens.ErrTxWithWrongReceiver
	}

	dcrmAddress := token.DcrmAddress
	from := tx.From.String()
	if common.IsEqualIgnoreCase(from, dcrmAddress) {
		return nil, tokens.ErrTxWithWrongSender
	}

	input := (*[]byte)(tx.Payload)
	bindAddress, value, err := ParseSwapoutTxInput(input)
	if err != nil {
		log.Debug("EthBridge ParseSwapoutTxInput fail", "tx", txHash, "input", input, "err", err)
		return nil, tokens.ErrTxWithWrongInput
	}

	if !tokens.CheckSwapValue(value, b.IsSrc) {
		return nil, tokens.ErrTxWithWrongValue
	}

	if !tokens.SrcBridge.IsValidAddress(bindAddress) {
		log.Debug("wrong bind address in swapout", "bind", bindAddress)
		err = tokens.ErrTxWithWrongMemo
	}

	blockHeight := tx.BlockNumber.ToInt().Uint64()
	log.Debug("verify swapout unstable pass", "from", from, "to", to, "bind", bindAddress, "value", value, "txid", txHash, "height", blockHeight)
	return &tokens.TxSwapInfo{
		Hash:      txHash,
		Height:    blockHeight,
		Timestamp: 0,
		From:      from,
		To:        contractAddress,
		Bind:      bindAddress,
		Value:     value.String(),
	}, err
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
