package nebulas

import (
	"encoding/json"
	"errors"
	"math/big"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// verifySwapoutTxWithPairID verify swapout with PairID
func (b *Bridge) verifySwapoutTx(swapInfo *tokens.TxSwapInfo, allowUnstable bool, token *tokens.TokenConfig, receipt *TransactionResponse) (*tokens.TxSwapInfo, error) {
	err := b.verifySwapoutTxReceipt(swapInfo, receipt, token)
	if err != nil {
		return swapInfo, err
	}

	err = b.checkSwapoutInfo(swapInfo)
	if err != nil {
		return swapInfo, err
	}

	if !allowUnstable {
		log.Info("verify swapout stable pass",
			"identifier", params.GetIdentifier(), "pairID", swapInfo.PairID,
			"from", swapInfo.From, "to", swapInfo.To, "bind", swapInfo.Bind,
			"value", swapInfo.Value, "txid", swapInfo.Hash,
			"height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
	}

	return swapInfo, nil
}

func (b *Bridge) verifySwapoutTxReceipt(swapInfo *tokens.TxSwapInfo, receipt *TransactionResponse, token *tokens.TokenConfig) error {
	if len(receipt.To) == 0 {
		return tokens.ErrTxWithWrongContract
	}

	txRecipient := strings.ToLower(receipt.To)
	swapInfo.TxTo = txRecipient                   // TxTo
	swapInfo.To = txRecipient                     // To
	swapInfo.From = strings.ToLower(receipt.From) // From

	if common.IsEqualIgnoreCase(swapInfo.From, token.DcrmAddress) {
		return tokens.ErrTxWithWrongSender
	}

	if !token.AllowSwapoutFromContract &&
		!common.IsEqualIgnoreCase(swapInfo.TxTo, token.ContractAddress) &&
		!b.ChainConfig.IsInCallByContractWhitelist(swapInfo.TxTo) {
		return tokens.ErrTxWithWrongContract
	}

	bindAddress, value, err := parseSwapoutTxData(receipt.Data, token.ContractAddress)
	if err != nil {
		if !errors.Is(err, tokens.ErrSwapoutLogNotFound) {
			log.Debug(b.ChainConfig.BlockChain+" parseSwapoutTxLogs fail", "tx", swapInfo.Hash, "err", err)
		}
		return err
	}
	swapInfo.Bind = bindAddress // Bind
	swapInfo.Value = value      // Value
	return nil
}

func (b *Bridge) checkSwapoutInfo(swapInfo *tokens.TxSwapInfo) error {
	if !tokens.CheckSwapValue(swapInfo, b.IsSrc) {
		return tokens.ErrTxWithWrongValue
	}
	if !tokens.SrcBridge.IsValidAddress(swapInfo.Bind) {
		log.Debug("wrong bind address in swapout", "bind", swapInfo.Bind)
		return tokens.ErrTxWithWrongMemo
	}
	return nil
}

func parseSwapoutTxData(data []byte, targetContract string) (bind string, value *big.Int, err error) {
	payload, err := LoadCallPayload(data)
	if err != nil {
		return "", nil, tokens.ErrSwapoutLogNotFound
	}
	if payload.Function != "transfer" {
		return "", nil, errors.New("unsupported swap function")
	}
	var args []string
	err = json.Unmarshal([]byte(payload.Args), &args)
	if err != nil {
		return "", nil, errors.New("failed to parse payload args")
	}
	value, ok := new(big.Int).SetString(args[1], 10)
	if !ok {
		return "", nil, errors.New("failed to parse payload value")
	}
	return args[0], value, nil
}
