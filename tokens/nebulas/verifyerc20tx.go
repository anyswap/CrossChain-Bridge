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
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

// verifyErc20SwapinTx verify erc20 swapin with pairID
func (b *Bridge) verifyErc20SwapinTx(swapInfo *tokens.TxSwapInfo, allowUnstable bool, token *tokens.TokenConfig, receipt *TransactionResponse) (*tokens.TxSwapInfo, error) {
	err := b.verifyErc20SwapinTxReceipt(swapInfo, receipt, token)
	if err != nil {
		return swapInfo, err
	}

	err = b.checkSwapinInfo(swapInfo)
	if err != nil {
		return swapInfo, err
	}

	if !allowUnstable {
		log.Info("verify erc20 swapin stable pass",
			"identifier", params.GetIdentifier(), "pairID", swapInfo.PairID,
			"from", swapInfo.From, "to", swapInfo.To, "bind", swapInfo.Bind,
			"value", swapInfo.Value, "txid", swapInfo.Hash,
			"height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
	}
	return swapInfo, nil
}

func (b *Bridge) verifyErc20SwapinTxReceipt(swapInfo *tokens.TxSwapInfo, receipt *TransactionResponse, token *tokens.TokenConfig) error {
	if len(receipt.To) == 0 {
		return tokens.ErrTxWithWrongContract
	}

	swapInfo.TxTo = strings.ToLower(receipt.To)   // TxTo
	swapInfo.From = strings.ToLower(receipt.From) // From

	if !token.AllowSwapinFromContract &&
		!common.IsEqualIgnoreCase(swapInfo.TxTo, token.ContractAddress) &&
		!b.ChainConfig.IsInCallByContractWhitelist(swapInfo.TxTo) {
		return tokens.ErrTxWithWrongContract
	}

	from, to, value, err := ParseErc20SwapinTx(receipt, token.ContractAddress, token.DepositAddress)
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

// ParseErc20SwapinTxLogs parse erc20 swapin tx logs
func ParseErc20SwapinTx(tx *TransactionResponse, contractAddress, checkToAddress string) (from, to string, value *big.Int, err error) {
	if tx.To != contractAddress {
		return "", "", nil, errors.New("failed to check to address.")
	}
	payload, err := LoadCallPayload(tx.Data)
	if err != nil {
		return "", "", nil, err
	}

	if payload.Function != "transfer" {
		return "", "", nil, errors.New("unsupported swap function")
	}
	var args []string
	err = json.Unmarshal([]byte(payload.Args), &args)
	if err != nil {
		return "", "", nil, errors.New("failed to parse payload args")
	}
	if args[0] != checkToAddress {
		return "", "", nil, errors.New("failed to check to address")
	}
	value, ok := new(big.Int).SetString(args[1], 10)
	if !ok {
		return "", "", nil, errors.New("failed to parse payload value")
	}
	return tx.From, args[1], value, nil
}

func (b *Bridge) checkSwapinInfo(swapInfo *tokens.TxSwapInfo) error {
	if swapInfo.Bind == swapInfo.To {
		return tokens.ErrTxWithWrongSender
	}
	if !tokens.CheckSwapValue(swapInfo.PairID, swapInfo.Value, b.IsSrc) {
		return tokens.ErrTxWithWrongValue
	}
	token := b.GetTokenConfig(swapInfo.PairID)
	if token == nil {
		return tokens.ErrUnknownPairID
	}
	bindAddr := swapInfo.Bind
	if !tokens.DstBridge.IsValidAddress(bindAddr) {
		log.Warn("wrong bind address in swapin", "bind", bindAddr)
		return tokens.ErrTxWithWrongMemo
	}
	if params.MustRegisterAccount() && !tools.IsAddressRegistered(bindAddr) {
		return tokens.ErrTxSenderNotRegistered
	}
	if params.IsSwapServer && token.ContractAddress != "" &&
		!common.IsEqualIgnoreCase(swapInfo.TxTo, token.ContractAddress) &&
		!b.ChainConfig.IsInCallByContractWhitelist(swapInfo.TxTo) {
		isContract, err := b.IsContractAddress(bindAddr)
		if err != nil {
			log.Warn("query is contract address failed", "bindAddr", bindAddr, "err", err)
			return err
		}
		if isContract {
			return tokens.ErrBindAddrIsContract
		}
	}
	return nil
}
