package eth

import (
	"bytes"
	"errors"
	"math/big"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/types"
)

// verifyErc20SwapinTx verify erc20 swapin with pairID
func (b *Bridge) verifyErc20SwapinTx(pairID, txHash string, allowUnstable bool, token *tokens.TokenConfig) (*tokens.TxSwapInfo, error) {
	swapInfo := &tokens.TxSwapInfo{}
	swapInfo.PairID = pairID // PairID
	swapInfo.Hash = txHash   // Hash

	receipt, err := b.getReceipt(swapInfo, allowUnstable)
	if err != nil {
		return swapInfo, err
	}

	if receipt == nil && (token.IsDelegateContract || token.AllowSwapinFromContract) {
		return swapInfo, tokens.ErrTxNotFound
	}

	if !allowUnstable || receipt != nil {
		err = b.verifyErc20SwapinTxReceipt(swapInfo, receipt, token)
	} else {
		err = b.verifySwapinRawTx(swapInfo, token)
	}
	if err != nil {
		return swapInfo, err
	}

	err = b.checkSwapinInfo(swapInfo)
	if err != nil {
		return swapInfo, err
	}

	if !allowUnstable {
		log.Info("verify erc20 swapin pass", "pairID", swapInfo.PairID, "from", swapInfo.From, "to", swapInfo.To, "bind", swapInfo.Bind, "value", swapInfo.Value, "txid", txHash, "height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
	}
	return swapInfo, nil
}

func (b *Bridge) verifyErc20SwapinTxReceipt(swapInfo *tokens.TxSwapInfo, receipt *types.RPCTxReceipt, token *tokens.TokenConfig) error {
	if receipt.Recipient == nil ||
		!common.IsEqualIgnoreCase(receipt.Recipient.String(), token.ContractAddress) {
		if !token.AllowSwapinFromContract {
			return tokens.ErrTxWithWrongContract
		}
	}

	swapInfo.TxTo = strings.ToLower(receipt.Recipient.String()) // TxTo
	swapInfo.From = strings.ToLower(receipt.From.String())      // From

	from, to, value, err := ParseErc20SwapinTxLogs(receipt.Logs, token.ContractAddress, token.DepositAddress)
	if err != nil {
		if !errors.Is(err, tokens.ErrTxWithWrongReceiver) {
			log.Debug(b.ChainConfig.BlockChain+" ParseErc20SwapinTxLogs failed", "tx", swapInfo.Hash, "err", err)
		}
		return err
	}
	swapInfo.To = strings.ToLower(to)     // To
	swapInfo.Value = value                // Value
	swapInfo.Bind = strings.ToLower(from) // Bind
	return nil
}

func (b *Bridge) verifySwapinRawTx(swapInfo *tokens.TxSwapInfo, token *tokens.TokenConfig) error {
	txHash := swapInfo.Hash
	tx, err := b.GetTransactionByHash(txHash)
	if err != nil {
		log.Debug("[verifySwapinRawTx] "+b.ChainConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txHash, "err", err)
		return tokens.ErrTxNotFound
	}
	if tx.Recipient == nil {
		return tokens.ErrTxWithWrongContract
	}

	txRecipient := strings.ToLower(tx.Recipient.String())
	if !common.IsEqualIgnoreCase(txRecipient, token.ContractAddress) {
		return tokens.ErrTxWithWrongContract
	}

	swapInfo.TxTo = txRecipient                       // TxTo
	swapInfo.From = strings.ToLower(tx.From.String()) // From

	input := (*[]byte)(tx.Payload)
	from, to, value, err := ParseErc20SwapinTxInput(input, token.DepositAddress)
	if err != nil {
		if !errors.Is(err, tokens.ErrTxWithWrongReceiver) {
			log.Debug(b.ChainConfig.BlockChain+" ParseErc20SwapinTxInput fail", "tx", swapInfo.Hash, "err", err)
		}
		return err
	}
	swapInfo.To = strings.ToLower(to) // To
	swapInfo.Value = value            // Value
	if from != "" {
		swapInfo.Bind = strings.ToLower(from) // Bind
	} else {
		swapInfo.Bind = swapInfo.From // Bind
	}
	return nil
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
func ParseErc20SwapinTxLogs(logs []*types.RPCLog, contractAddress, checkToAddress string) (from, to string, value *big.Int, err error) {
	transferLogExist := false
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
		transferLogExist = true
		to = common.BytesToAddress(log.Topics[2][:]).String()
		if !common.IsEqualIgnoreCase(to, checkToAddress) {
			continue
		}
		if log.Address == nil || !common.IsEqualIgnoreCase(log.Address.String(), contractAddress) {
			continue
		}
		from = common.BytesToAddress(log.Topics[1][:]).String()
		value = common.GetBigInt(*log.Data, 0, 32)
		return from, to, value, nil
	}
	if transferLogExist {
		err = tokens.ErrTxWithWrongReceiver
	} else {
		err = tokens.ErrDepositLogNotFound
	}
	return "", "", nil, err
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
