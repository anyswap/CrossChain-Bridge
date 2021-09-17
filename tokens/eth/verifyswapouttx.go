package eth

import (
	"bytes"
	"errors"
	"math/big"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/types"
)

// verifySwapoutTxWithPairID verify swapout with PairID
func (b *Bridge) verifySwapoutTx(swapInfo *tokens.TxSwapInfo, allowUnstable bool, token *tokens.TokenConfig, receipt *types.RPCTxReceipt) (*tokens.TxSwapInfo, error) {
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

func (b *Bridge) verifySwapoutTxReceipt(swapInfo *tokens.TxSwapInfo, receipt *types.RPCTxReceipt, token *tokens.TokenConfig) error {
	if receipt.Recipient == nil {
		return tokens.ErrTxWithWrongContract
	}

	txRecipient := strings.ToLower(receipt.Recipient.String())
	swapInfo.TxTo = txRecipient                            // TxTo
	swapInfo.To = txRecipient                              // To
	swapInfo.From = strings.ToLower(receipt.From.String()) // From

	if common.IsEqualIgnoreCase(swapInfo.From, token.DcrmAddress) {
		return tokens.ErrTxWithWrongSender
	}

	if !token.AllowSwapoutFromContract &&
		!common.IsEqualIgnoreCase(swapInfo.TxTo, token.ContractAddress) &&
		!b.ChainConfig.IsInCallByContractWhitelist(swapInfo.TxTo) {
		return tokens.ErrTxWithWrongContract
	}

	bindAddress, value, err := parseSwapoutTxLogs(receipt.Logs, token.ContractAddress)
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
	if !tokens.CheckSwapValue(swapInfo.PairID, swapInfo.Value, b.IsSrc) {
		return tokens.ErrTxWithWrongValue
	}
	if !tokens.SrcBridge.IsValidAddress(swapInfo.Bind) {
		log.Debug("wrong bind address in swapout", "bind", swapInfo.Bind)
		return tokens.ErrTxWithWrongMemo
	}
	return nil
}

func parseSwapoutTxLogs(logs []*types.RPCLog, targetContract string) (bind string, value *big.Int, err error) {
	isSwapoutToStrAddr := isSwapoutToStringAddress()
	logSwapoutTopic, topicsLen := getLogSwapoutTopic()
	for _, log := range logs {
		if log.Removed != nil && *log.Removed {
			continue
		}
		if !common.IsEqualIgnoreCase(log.Address.String(), targetContract) {
			continue
		}
		if len(log.Topics) != topicsLen || log.Data == nil {
			continue
		}
		if !bytes.Equal(log.Topics[0].Bytes(), logSwapoutTopic) {
			continue
		}
		if isSwapoutToStrAddr {
			return parseSwapoutToBtcEncodedData(*log.Data, false)
		}
		bind = common.BytesToAddress(log.Topics[2].Bytes()).String()
		value = common.GetBigInt(*log.Data, 0, 32)
		return bind, value, nil
	}
	return "", nil, tokens.ErrSwapoutLogNotFound
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
