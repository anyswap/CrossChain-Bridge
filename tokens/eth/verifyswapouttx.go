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

// verifySwapoutTxWithPairID verify swapout with PairID
func (b *Bridge) verifySwapoutTxWithPairID(pairID, txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	swapInfo := &tokens.TxSwapInfo{}
	swapInfo.PairID = pairID // PairID
	swapInfo.Hash = txHash   // Hash

	token := b.GetTokenConfig(pairID)
	if token == nil {
		return swapInfo, tokens.ErrUnknownPairID
	}

	receipt, err := b.getReceipt(swapInfo, allowUnstable)
	if err != nil {
		return swapInfo, err
	}

	if !allowUnstable || receipt != nil {
		err = b.verifySwapoutTxReceipt(swapInfo, receipt, token)
	} else {
		err = b.verifySwapoutRawTx(swapInfo, token)
	}
	if err != nil {
		return swapInfo, err
	}

	err = b.checkSwapoutInfo(swapInfo)
	if err != nil {
		return swapInfo, err
	}

	if !allowUnstable {
		log.Debug("verify swapout stable pass", "pairID", swapInfo.PairID, "from", swapInfo.From, "to", swapInfo.To, "bind", swapInfo.Bind, "value", swapInfo.Value, "txid", txHash, "height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
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

	bindAddress, value, err := parseSwapoutTxLogs(receipt.Logs, token.ContractAddress)
	if err != nil {
		if err != tokens.ErrSwapoutLogNotFound {
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

func (b *Bridge) verifySwapoutRawTx(swapInfo *tokens.TxSwapInfo, token *tokens.TokenConfig) error {
	txHash := swapInfo.Hash
	tx, err := b.GetTransactionByHash(txHash)
	if err != nil {
		log.Debug("[verifySwapoutWithPairID] "+b.ChainConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txHash, "err", err)
		return tokens.ErrTxNotFound
	}
	if tx.Recipient == nil { // ignore contract creation tx
		return tokens.ErrTxWithWrongContract
	}

	txRecipient := strings.ToLower(tx.Recipient.String())
	if !common.IsEqualIgnoreCase(txRecipient, token.ContractAddress) {
		return tokens.ErrTxWithWrongContract
	}

	swapInfo.TxTo = txRecipient                       // TxTo
	swapInfo.To = txRecipient                         // To
	swapInfo.From = strings.ToLower(tx.From.String()) // From

	input := (*[]byte)(tx.Payload)
	bindAddress, value, err := ParseSwapoutTxInput(input)
	if err != nil {
		if err != tokens.ErrTxFuncHashMismatch {
			log.Debug(b.ChainConfig.BlockChain+" ParseSwapoutTxInput fail", "tx", txHash, "err", err)
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

// verifySwapoutTx verify swapout (in scan job)
func (b *Bridge) verifySwapoutTx(txHash string, allowUnstable bool) ([]*tokens.TxSwapInfo, []error) {
	if allowUnstable {
		return b.verifySwapoutTxUnstable(txHash)
	}
	return b.verifySwapoutTxStable(txHash)
}

func (b *Bridge) verifySwapoutTxWithReceipt(commonInfo *tokens.TxSwapInfo, receipt *types.RPCTxReceipt) (swapInfos []*tokens.TxSwapInfo, errs []error) {
	if receipt.Recipient == nil {
		addSwapInfoConsiderError(nil, tokens.ErrTxWithWrongContract, &swapInfos, &errs)
		return swapInfos, errs
	}
	txHash := commonInfo.Hash
	txRecipient := strings.ToLower(receipt.Recipient.String())
	tokenCfgs, pairIDs := tokens.FindTokenConfig(txRecipient, false)
	if len(pairIDs) == 0 {
		addSwapInfoConsiderError(nil, tokens.ErrTxWithWrongContract, &swapInfos, &errs)
		return swapInfos, errs
	}
	commonInfo.TxTo = txRecipient                            // TxTo
	commonInfo.To = txRecipient                              // To
	commonInfo.From = strings.ToLower(receipt.From.String()) // From

	for i, pairID := range pairIDs {
		token := tokenCfgs[i]

		swapInfo := &tokens.TxSwapInfo{}
		*swapInfo = *commonInfo

		swapInfo.PairID = pairID // PairID

		bindAddress, value, err := parseSwapoutTxLogs(receipt.Logs, token.ContractAddress)
		if err != nil {
			if err != tokens.ErrSwapoutLogNotFound {
				log.Debug(b.ChainConfig.BlockChain+" parseSwapoutTxLogs fail", "tx", txHash, "err", err)
			}
			addSwapInfoConsiderError(swapInfo, err, &swapInfos, &errs)
			continue
		}
		if bindAddress != "" {
			swapInfo.Bind = bindAddress // Bind
		} else {
			swapInfo.Bind = swapInfo.From // Bind
		}
		swapInfo.Value = value // Value

		err = b.checkSwapoutInfo(swapInfo)
		if err != nil {
			addSwapInfoConsiderError(swapInfo, err, &swapInfos, &errs)
			continue
		}

		addSwapInfoConsiderError(swapInfo, nil, &swapInfos, &errs)

		log.Debug("verify swapout stable pass", "pairID", swapInfo.PairID, "from", swapInfo.From, "to", swapInfo.To, "bind", swapInfo.Bind, "value", swapInfo.Value, "txid", txHash, "height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
	}
	return swapInfos, errs
}

func (b *Bridge) verifySwapoutTxStable(txHash string) (swapInfos []*tokens.TxSwapInfo, errs []error) {
	commonInfo := &tokens.TxSwapInfo{}
	commonInfo.Hash = txHash // Hash
	receipt, err := b.getStableReceipt(commonInfo)
	if err != nil {
		addSwapInfoConsiderError(nil, err, &swapInfos, &errs)
		return swapInfos, errs
	}
	return b.verifySwapoutTxWithReceipt(commonInfo, receipt)
}

func (b *Bridge) verifySwapoutTxUnstable(txHash string) (swapInfos []*tokens.TxSwapInfo, errs []error) {
	commonInfo := &tokens.TxSwapInfo{}
	commonInfo.Hash = txHash // Hash
	if b.ChainConfig.ScanReceipt {
		receipt, _, _ := b.GetTransactionReceipt(txHash)
		if receipt != nil {
			commonInfo.Height = receipt.BlockNumber.ToInt().Uint64() // Height
			return b.verifySwapoutTxWithReceipt(commonInfo, receipt)
		}
	}
	tx, err := b.GetTransactionByHash(txHash)
	if err != nil {
		log.Debug("[verifySwapout] "+b.ChainConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txHash, "err", err)
		addSwapInfoConsiderError(nil, tokens.ErrTxNotFound, &swapInfos, &errs)
		return swapInfos, errs
	}
	if tx.Recipient == nil { // ignore contract creation tx
		addSwapInfoConsiderError(nil, tokens.ErrTxWithWrongContract, &swapInfos, &errs)
		return swapInfos, errs
	}

	txRecipient := strings.ToLower(tx.Recipient.String())
	tokenCfgs, pairIDs := tokens.FindTokenConfig(txRecipient, false)
	if len(pairIDs) == 0 {
		addSwapInfoConsiderError(nil, tokens.ErrTxWithWrongContract, &swapInfos, &errs)
		return swapInfos, errs
	}
	commonInfo.TxTo = txRecipient                       // TxTo
	commonInfo.To = txRecipient                         // To
	commonInfo.From = strings.ToLower(tx.From.String()) // From

	for i, pairID := range pairIDs {
		token := tokenCfgs[i]
		if !common.IsEqualIgnoreCase(txRecipient, token.ContractAddress) {
			continue
		}

		swapInfo := &tokens.TxSwapInfo{}
		*swapInfo = *commonInfo

		swapInfo.PairID = pairID // PairID

		input := (*[]byte)(tx.Payload)
		bindAddress, value, err := ParseSwapoutTxInput(input)
		if err != nil {
			if err != tokens.ErrTxFuncHashMismatch {
				log.Debug(b.ChainConfig.BlockChain+" parseSwapoutTxInput fail", "tx", txHash, "err", err)
			}
			addSwapInfoConsiderError(swapInfo, err, &swapInfos, &errs)
			continue
		}
		if bindAddress != "" {
			swapInfo.Bind = bindAddress // Bind
		} else {
			swapInfo.Bind = swapInfo.From // Bind
		}
		swapInfo.Value = value // Value

		err = b.checkSwapoutInfo(swapInfo)
		if err != nil {
			addSwapInfoConsiderError(swapInfo, err, &swapInfos, &errs)
			continue
		}

		addSwapInfoConsiderError(swapInfo, nil, &swapInfos, &errs)
	}

	return swapInfos, errs
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

// ParseSwapoutTxInput parse swapout tx input
func ParseSwapoutTxInput(input *[]byte) (string, *big.Int, error) {
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

func parseSwapoutTxLogs(logs []*types.RPCLog, targetContract string) (bind string, value *big.Int, err error) {
	if isMbtcSwapout() {
		return parseSwapoutToBtcTxLogs(logs)
	}
	logSwapoutTopic := getLogSwapoutTopic()
	for _, log := range logs {
		if log.Removed != nil && *log.Removed {
			continue
		}
		if !common.IsEqualIgnoreCase(log.Address.String(), targetContract) {
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
	if isMbtcSwapout() || tokens.IsSwapoutToStringAddress {
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
