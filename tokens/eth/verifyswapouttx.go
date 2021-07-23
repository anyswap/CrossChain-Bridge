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

// verifySwapoutTxWithPairID verify swapout with PairID
func (b *Bridge) verifySwapoutTxWithPairID(pairID, txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	swapInfo := &tokens.TxSwapInfo{}
	swapInfo.PairID = pairID // PairID
	swapInfo.Hash = txHash   // Hash

	token := b.GetTokenConfig(pairID)
	if token == nil {
		return swapInfo, tokens.ErrUnknownPairID
	}

	if token.DisableSwap {
		return swapInfo, tokens.ErrSwapIsClosed
	}

	receipt, err := b.getReceipt(swapInfo, allowUnstable)
	if err != nil {
		return swapInfo, err
	}

	if receipt == nil {
		return swapInfo, tokens.ErrTxNotFound
	}

	err = b.verifySwapoutTxReceipt(swapInfo, receipt, token)
	if err != nil {
		return swapInfo, err
	}

	err = b.checkSwapoutInfo(swapInfo)
	if err != nil {
		return swapInfo, err
	}

	if !allowUnstable {
		log.Info("verify swapout stable pass", "pairID", swapInfo.PairID, "from", swapInfo.From, "to", swapInfo.To, "bind", swapInfo.Bind, "value", swapInfo.Value, "txid", txHash, "height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
	}

	return swapInfo, nil
}

func (b *Bridge) verifySwapoutTxReceipt(swapInfo *tokens.TxSwapInfo, receipt *types.RPCTxReceipt, token *tokens.TokenConfig) error {
	if receipt.Recipient == nil {
		return tokens.ErrTxWithWrongContract
	}

	if !token.AllowSwapoutFromContract &&
		!common.IsEqualIgnoreCase(receipt.Recipient.String(), token.ContractAddress) {
		return tokens.ErrTxWithWrongContract
	}

	txRecipient := strings.ToLower(receipt.Recipient.String())
	swapInfo.TxTo = txRecipient                            // TxTo
	swapInfo.To = txRecipient                              // To
	swapInfo.From = strings.ToLower(receipt.From.String()) // From

	bindAddress, value, err := parseSwapoutTxLogs(receipt.Logs, token.ContractAddress)
	if err != nil {
		if !errors.Is(err, tokens.ErrSwapoutLogNotFound) {
			log.Debug(b.ChainConfig.BlockChain+" parseSwapoutTxLogs fail", "tx", swapInfo.Hash, "err", err)
		}
		return err
	}
	if bindAddress != "" {
		swapInfo.Bind = bindAddress // Bind
	} else {
		swapInfo.Bind = swapInfo.From // Bind
	}
	swapInfo.Value = value // Value
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
	isSwapoutToBtc := isMbtcSwapout()
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
		if isSwapoutToBtc {
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
