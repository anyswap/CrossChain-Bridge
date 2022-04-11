package terra

import (
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var (
	errTxResultType = errors.New("tx type is not TxResponse")
	errTxEvent      = errors.New("tx event is not support")
	errTxAmount     = errors.New("tx amount is zero")
	errTxLog        = errors.New("tx don't has execute_contract log")
)

// GetTransaction impl
func (b *Bridge) GetTransaction(txHash string) (interface{}, error) {
	return b.GetTransactionByHash(txHash)
}

// GetTransactionByHash get tx response by hash
func (b *Bridge) GetTransactionByHash(txHash string) (result *GetTxResult, err error) {
	urls := append(b.GatewayConfig.APIAddress, b.GatewayConfig.APIAddressExt...)
	for _, url := range urls {
		result, err = GetTransactionByHash(url, txHash)
		if err == nil {
			return result, nil
		}
	}
	return nil, wrapRPCQueryError(err, "GetTransactionByHash", txHash)
}

// GetTransactionStatus impl
func (b *Bridge) GetTransactionStatus(txHash string) (*tokens.TxStatus, error) {
	txr, err := b.GetTransactionByHash(txHash)
	if err != nil {
		return nil, err
	}

	blockHeight, err := common.GetInt64FromStr(txr.TxResponse.Height)
	if err != nil {
		return nil, err
	}

	txStatus := &tokens.TxStatus{}
	txStatus.BlockHeight = uint64(blockHeight)

	if txStatus.BlockHeight != 0 {
		for i := 0; i < 3; i++ {
			latest, errt := b.GetLatestBlockNumber()
			if errt == nil {
				if latest > txStatus.BlockHeight {
					txStatus.Confirmations = latest - txStatus.BlockHeight
				}
				break
			}
			time.Sleep(1 * time.Second)
		}
	}
	return txStatus, nil
}

// GetTxBlockInfo impl NonceSetter interface
func (b *Bridge) GetTxBlockInfo(txHash string) (blockHeight, blockTime uint64) {
	txStatus, err := b.GetTransactionStatus(txHash)
	if err != nil {
		return 0, 0
	}
	return txStatus.BlockHeight, txStatus.BlockTime
}

// VerifyMsgHash verify msg hash
func (b *Bridge) VerifyMsgHash(rawTx interface{}, msgHash []string) (err error) {
	return tokens.ErrTodo
}

// VerifyTransaction impl
func (b *Bridge) VerifyTransaction(pairID, txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	if !b.IsSrc {
		return nil, tokens.ErrBridgeDestinationNotSupported
	}
	swapInfo := &tokens.TxSwapInfo{}
	swapInfo.PairID = pairID // PairID
	swapInfo.Hash = txHash   // Hash

	token := b.GetTokenConfig(pairID)
	if token == nil {
		return swapInfo, tokens.ErrUnknownPairID
	}

	tx, err := b.GetTransaction(txHash)
	if err != nil {
		log.Debug("[verifySwapin] "+b.ChainConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txHash, "err", err)
		return swapInfo, tokens.ErrTxNotFound
	}

	txres, ok := tx.(*TxResponse)
	if !ok {
		return swapInfo, errTxResultType
	}

	if !allowUnstable {
		h, err := b.GetLatestBlockNumber()
		if err != nil {
			return swapInfo, err
		}
		height, errf := common.GetUint64FromStr(txres.Height)
		if errf != nil {
			return swapInfo, errf
		}
		if h < height+*b.GetChainConfig().Confirmations {
			return swapInfo, tokens.ErrTxNotStable
		}
		if h < *b.ChainConfig.InitialHeight {
			return swapInfo, tokens.ErrTxBeforeInitialHeight
		}
	}

	// Check tx status
	if txres.Code != 0 {
		return swapInfo, tokens.ErrTxWithWrongStatus
	}
	var events StringEvents
	from := ""
	for _, log := range txres.Logs {
		for _, event := range log.Events {
			if from == "" && event.Type == "execute_contract" && event.Attributes[1].Key == "contract_address" && common.IsEqualIgnoreCase(event.Attributes[1].Value, token.ContractAddress) {
				from = event.Attributes[0].Value
			}
			if event.Type == "from_contract" && event.Attributes[0].Key == "contract_address" && common.IsEqualIgnoreCase(event.Attributes[0].Value, token.ContractAddress) {
				events = append(events, event)
			}
		}
	}
	if from == "" {
		return swapInfo, errTxLog
	}

	if len(events) == 0 {
		return swapInfo, errTxEvent
	}

	amount := b.checkEvents(pairID, events)

	if amount.CmpAbs(big.NewInt(0)) == 0 {
		return swapInfo, errTxAmount
	}

	bind, ok2 := GetBindAddressFromMemos(tx.(*Tx).Body)
	if !ok2 {
		log.Debug("wrong memos", "memos", bind)
		return swapInfo, tokens.ErrWrongMemoBindAddress
	}

	swapInfo.To = token.DepositAddress    // To
	swapInfo.From = strings.ToLower(from) // From
	swapInfo.Bind = bind                  // Bin
	swapInfo.Value = amount

	if !allowUnstable {
		log.Info("verify swapin pass", "pairID", swapInfo.PairID, "from", swapInfo.From, "to", swapInfo.To, "bind", swapInfo.Bind, "value", swapInfo.Value, "txid", swapInfo.Hash, "height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
	}
	return swapInfo, nil
}

func (b *Bridge) checkEvents(pairID string, events StringEvents) (amount *big.Int) {
	token := b.GetTokenConfig(pairID)
	amount = big.NewInt(0)
	for _, event := range events {
		if event.Attributes[1].Value == "transfer" && common.IsEqualIgnoreCase(event.Attributes[3].Value, token.DepositAddress) {
			amt, _ := common.GetBigIntFromStr(event.Attributes[4].Value)
			amount.Add(amount, amt)
		} else if event.Attributes[1].Value == "transfer_from" && common.IsEqualIgnoreCase(event.Attributes[3].Value, token.DepositAddress) {
			amt, _ := common.GetBigIntFromStr(event.Attributes[5].Value)
			amount.Add(amount, amt)
		} else if event.Attributes[1].Value == "send" && common.IsEqualIgnoreCase(event.Attributes[3].Value, token.DepositAddress) {
			amt, _ := common.GetBigIntFromStr(event.Attributes[4].Value)
			amount.Add(amount, amt)
		} else if event.Attributes[1].Value == "send_from" && common.IsEqualIgnoreCase(event.Attributes[3].Value, token.DepositAddress) {
			amt, _ := common.GetBigIntFromStr(event.Attributes[5].Value)
			amount.Add(amount, amt)
		} else if event.Attributes[1].Value == "mint" && common.IsEqualIgnoreCase(event.Attributes[2].Value, token.DepositAddress) {
			amt, _ := common.GetBigIntFromStr(event.Attributes[3].Value)
			amount.Add(amount, amt)
		}
	}
	return amount
}

// GetBindAddressFromMemos get bind address
func GetBindAddressFromMemos(txBody TxBody) (string, bool) {
	bindStr := txBody.Memo
	if tokens.DstBridge.IsValidAddress(bindStr) {
		return bindStr, true
	}
	return bindStr, false
}
