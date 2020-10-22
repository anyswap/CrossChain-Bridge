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

// verifyErc20SwapinTx verify erc20 swapin with pairID
func (b *Bridge) verifyErc20SwapinTx(tx *types.RPCTransaction, pairID string, token *tokens.TokenConfig, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	if allowUnstable {
		return b.verifyErc20SwapinTxUnstable(tx, pairID, token)
	}
	return b.verifyErc20SwapinTxStable(tx, pairID, token)
}

func (b *Bridge) verifyErc20SwapinTxStable(tx *types.RPCTransaction, pairID string, token *tokens.TokenConfig) (*tokens.TxSwapInfo, error) {
	txHash := tx.Hash.String()
	txRecipient := strings.ToLower(tx.Recipient.String())

	swapInfo := &tokens.TxSwapInfo{}
	swapInfo.PairID = pairID    // PairID
	swapInfo.Hash = txHash      // Hash
	swapInfo.TxTo = txRecipient // TxTo

	receipt, err := b.getStableReceipt(swapInfo)
	if err != nil {
		return swapInfo, err
	}

	if receipt.Recipient == nil ||
		!common.IsEqualIgnoreCase(receipt.Recipient.String(), token.ContractAddress) {
		return swapInfo, tokens.ErrTxWithWrongContract
	}

	swapInfo.From = strings.ToLower(receipt.From.String()) // From

	from, to, value, err := ParseErc20SwapinTxLogs(receipt.Logs, token.DepositAddress)
	if err != nil {
		if err != tokens.ErrTxWithWrongReceiver {
			log.Debug(b.ChainConfig.BlockChain+" ParseErc20SwapinTxLogs failed", "tx", txHash, "err", err)
		}
		return swapInfo, err
	}
	swapInfo.To = strings.ToLower(to)     // To
	swapInfo.Value = value                // Value
	swapInfo.Bind = strings.ToLower(from) // Bind

	err = b.checkSwapinInfo(swapInfo)
	if err != nil {
		return swapInfo, err
	}

	log.Debug("verify erc20 swapin pass", "from", "pairID", swapInfo.PairID, swapInfo.From, "to", swapInfo.To, "bind", swapInfo.Bind, "value", swapInfo.Value, "txid", txHash, "height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
	return swapInfo, nil
}

func (b *Bridge) verifyErc20SwapinTxUnstable(tx *types.RPCTransaction, pairID string, token *tokens.TokenConfig) (*tokens.TxSwapInfo, error) {
	txHash := tx.Hash.String()
	swapInfo := &tokens.TxSwapInfo{}
	swapInfo.PairID = pairID // PairID
	swapInfo.Hash = txHash   // Hash

	if tx.Recipient == nil {
		return swapInfo, tokens.ErrTxWithWrongContract
	}

	txRecipient := strings.ToLower(tx.Recipient.String())
	if !common.IsEqualIgnoreCase(txRecipient, token.ContractAddress) {
		return swapInfo, tokens.ErrTxWithWrongContract
	}

	swapInfo.TxTo = txRecipient                       // TxTo
	swapInfo.From = strings.ToLower(tx.From.String()) // From

	input := (*[]byte)(tx.Payload)
	from, to, value, err := ParseErc20SwapinTxInput(input, token.DepositAddress)
	if err != nil {
		if err != tokens.ErrTxWithWrongReceiver {
			log.Debug(b.ChainConfig.BlockChain+" ParseErc20SwapinTxInput fail", "tx", txHash, "err", err)
		}
		return swapInfo, err
	}
	swapInfo.To = strings.ToLower(to) // To
	swapInfo.Value = value            // Value
	if from != "" {
		swapInfo.Bind = strings.ToLower(from) // Bind
	} else {
		swapInfo.Bind = swapInfo.From // Bind
	}

	err = b.checkSwapinInfo(swapInfo)
	if err != nil {
		return swapInfo, err
	}

	return swapInfo, nil
}

// ParseErc20SwapinTxInput parse erc20 swapin tx input
func ParseErc20SwapinTxInput(input *[]byte, checkToAddress string) (from, to string, value *big.Int, err error) {
	if input == nil || len(*input) < 4 {
		return "", "", nil, tokens.ErrTxWithWrongInput
	}
	data := *input
	funcHash := data[:4]
	isTransferFrom := false
	switch {
	case bytes.Equal(funcHash, erc20CodeParts["transfer"]):
	case bytes.Equal(funcHash, erc20CodeParts["transferFrom"]):
		isTransferFrom = true
	default:
		return "", "", nil, tokens.ErrTxFuncHashMismatch
	}
	encData := data[4:]
	return parseErc20EncodedData(encData, isTransferFrom, checkToAddress)
}

// ParseErc20SwapinTxLogs parse erc20 swapin tx logs
func ParseErc20SwapinTxLogs(logs []*types.RPCLog, checkToAddress string) (from, to string, value *big.Int, err error) {
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
		value = common.GetBigInt(*log.Data, 0, 32)
		if !common.IsEqualIgnoreCase(to, checkToAddress) {
			err = tokens.ErrTxWithWrongReceiver
		}
		return from, to, value, err
	}
	return "", "", nil, tokens.ErrDepositLogNotFound
}

func parseErc20EncodedData(encData []byte, isTransferFrom bool, checkToAddress string) (from, to string, value *big.Int, err error) {
	// use common GetData and GetBigInt to prevent index overflow
	if isTransferFrom {
		from = common.BytesToAddress(common.GetData(encData, 0, 32)).String()
		to = common.BytesToAddress(common.GetData(encData, 32, 32)).String()
		value = common.GetBigInt(encData, 64, 32)
		if len(encData) != 96 {
			err = tokens.ErrTxIncompatible
		}
	} else {
		to = common.BytesToAddress(common.GetData(encData, 0, 32)).String()
		value = common.GetBigInt(encData, 32, 32)
		if len(encData) != 64 {
			err = tokens.ErrTxIncompatible
		}
	}
	// error ErrTxWithWrongReceiver has highest priority,
	// because this error means we don't care about this tx.
	if !common.IsEqualIgnoreCase(to, checkToAddress) {
		err = tokens.ErrTxWithWrongReceiver
	}
	return from, to, value, err
}
