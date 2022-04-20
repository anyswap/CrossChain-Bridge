package near

import (
	"math/big"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// GetTransaction impl
func (b *Bridge) GetTransaction(txHash string) (interface{}, error) {
	return b.GetTransactionByHash(txHash)
}

// GetTransactionByHash get tx response by hash
func (b *Bridge) GetTransactionByHash(txHash string) (result *TransactionResult, err error) {
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
	tmpHeight, errf := b.GetBlockByHash(txr.ReceiptsOutcome[0].BlockHash)
	if errf != nil {
		return nil, errf
	}

	txHeight, errh := common.GetUint64FromStr(tmpHeight)
	if errh != nil {
		return nil, errh
	}

	txStatus := &tokens.TxStatus{}
	txStatus.BlockHeight = uint64(txHeight)

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
	txr, err := b.GetTransactionByHash(txHash)
	if err != nil {
		return
	}
	tmpHeight, errf := b.GetBlockByHash(txr.ReceiptsOutcome[0].BlockHash)
	if errf != nil {
		return
	}

	height, errh := common.GetUint64FromStr(tmpHeight)
	if errh != nil {
		return
	}
	blockHeight = uint64(height)
	return
}

// VerifyMsgHash verify msg hash
func (b *Bridge) VerifyMsgHash(rawTx interface{}, msgHashes []string) (err error) {
	return nil
}

// VerifyTransaction impl
func (b *Bridge) VerifyTransaction(pairID, txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	swapInfo := &tokens.TxSwapInfo{}
	swapInfo.PairID = pairID // PairID
	swapInfo.Hash = txHash   // Hash

	token := b.GetTokenConfig(pairID)
	if token == nil {
		return swapInfo, tokens.ErrUnknownPairID
	}

	txr, err := b.GetTransactionByHash(txHash)
	if err != nil {
		log.Debug("[verifySwapin] "+b.ChainConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txHash, "err", err)
		return swapInfo, tokens.ErrTxNotFound
	}
	txRes := &txr.ReceiptsOutcome[0]
	txStatus := &txr.Status
	txHeight, err := b.checkTxStatus(txRes, txStatus, allowUnstable)
	if err != nil {
		return swapInfo, err
	}
	swapInfo.Height = txHeight // Height

	var bind string
	var amount *big.Int
	if token.ContractAddress != "" {
		bind, amount, err = b.checkTokenDepist(txRes, token)
	} else {
		bind, amount, err = b.checkCoinDeposit(txRes, token)
	}
	if err != nil {
		return swapInfo, err
	}

	swapInfo.From = strings.ToLower(txr.Transaction.SignerID) // From
	swapInfo.To = token.DepositAddress                        // To
	swapInfo.Value = amount                                   // Value
	swapInfo.Bind = bind                                      // Bind

	err = b.checkSwapinInfo(swapInfo)
	if err != nil {
		return swapInfo, err
	}

	if !allowUnstable {
		log.Info("verify swapin stable pass",
			"identifier", params.GetIdentifier(), "pairID", swapInfo.PairID,
			"from", swapInfo.From, "to", swapInfo.To, "bind", swapInfo.Bind,
			"value", swapInfo.Value, "txid", swapInfo.Hash,
			"height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
	}
	return swapInfo, nil
}

func (b *Bridge) checkTxStatus(txres *ReceiptsOutcome, txstatus *Status, allowUnstable bool) (txHeight uint64, err error) {
	tmpHeight, errf := b.GetBlockByHash(txres.BlockHash)
	if errf != nil {
		return 0, errf
	}

	txHeight, err = common.GetUint64FromStr(tmpHeight)
	if err != nil {
		return txHeight, err
	}

	if txstatus.SuccessValue != "" {
		return txHeight, tokens.ErrTxWithWrongStatus
	}

	if !allowUnstable {
		h, errf := b.GetLatestBlockNumber()
		if errf != nil {
			return txHeight, errf
		}
		if h < txHeight+*b.GetChainConfig().Confirmations {
			return txHeight, tokens.ErrTxNotStable
		}
		if h < *b.ChainConfig.InitialHeight {
			return txHeight, tokens.ErrTxBeforeInitialHeight
		}
	}

	return txHeight, err
}

//nolint:goconst // allow big check logic
func (b *Bridge) checkCoinDeposit(txres *ReceiptsOutcome, token *tokens.TokenConfig) (bindAddr string, amount *big.Int, err error) {
	outcome := txres.Outcome
	if outcome.ExecutorID != token.ContractAddress {
		return "", big.NewInt(0), tokens.ErrTxWithWrongContract
	}
	if len(outcome.Logs) != 1 {
		return "", big.NewInt(0), tokens.ErrTxWithWrongLogData
	}
	logs := outcome.Logs[0].(*string)
	logArray := strings.Split(*logs, " ")
	if logArray[0] == "LogSwapOut_native" && logArray[1] == "bindAddr" && logArray[3] == "amount" {
		bindAddr = logArray[2]
		amount, err = common.GetBigIntFromStr(logArray[4])
		if err != nil {
			return "", big.NewInt(0), err
		}
	}
	return bindAddr, amount, nil
}

func (b *Bridge) checkTokenDepist(txres *ReceiptsOutcome, token *tokens.TokenConfig) (bindAddr string, amount *big.Int, err error) {
	outcome := txres.Outcome
	if outcome.ExecutorID != token.ContractAddress {
		return "", big.NewInt(0), tokens.ErrTxWithWrongContract
	}
	if len(outcome.Logs) != 1 {
		return "", big.NewInt(0), tokens.ErrTxWithWrongLogData
	}
	logs := outcome.Logs[0].(*string)
	logArray := strings.Split(*logs, " ")
	if logArray[0] == "LogSwapOut" && logArray[1] == "bindAddr" && logArray[3] == "amount" {
		bindAddr = logArray[2]
		amount, err = common.GetBigIntFromStr(logArray[4])
		if err != nil {
			return "", big.NewInt(0), err
		}
	}
	return bindAddr, amount, nil
}

func (b *Bridge) checkSwapinInfo(swapInfo *tokens.TxSwapInfo) error {
	if swapInfo.Bind == swapInfo.To {
		return tokens.ErrTxWithWrongSender
	}
	if !tokens.CheckSwapValue(swapInfo, b.IsSrc) {
		return tokens.ErrTxWithWrongValue
	}
	bindAddr := swapInfo.Bind
	if !tokens.DstBridge.IsValidAddress(bindAddr) {
		log.Warn("wrong bind address in swapin", "bind", bindAddr)
		return tokens.ErrTxWithWrongMemo
	}
	return nil
}
