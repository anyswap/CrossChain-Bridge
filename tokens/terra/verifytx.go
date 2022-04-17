package terra

import (
	"fmt"
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
	txr, err := b.GetTransactionByHash(txHash)
	if err != nil {
		return
	}
	height, err := common.GetInt64FromStr(txr.TxResponse.Height)
	if err != nil {
		return
	}
	blockHeight = uint64(height)
	return
}

// VerifyMsgHash verify msg hash
func (b *Bridge) VerifyMsgHash(rawTx interface{}, msgHashes []string) (err error) {
	txb, ok := rawTx.(*TxBuilder)
	if !ok {
		return tokens.ErrWrongRawTx
	}

	if len(msgHashes) < 1 {
		return tokens.ErrWrongCountOfMsgHashes
	}
	msgHash := msgHashes[0]

	signBytes, err := txb.GetSignBytes()
	if err != nil {
		return err
	}
	sigHash := fmt.Sprintf("%X", common.Sha256Sum(signBytes))

	if !strings.EqualFold(sigHash, msgHash) {
		logFunc := log.GetPrintFuncOr(params.IsDebugMode, log.Info, log.Trace)
		logFunc("message hash mismatch", "want", msgHash, "have", sigHash)
		return tokens.ErrMsgHashMismatch
	}
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

	txres := txr.TxResponse

	txHeight, err := b.checkTxStatus(&txres, allowUnstable)
	if err != nil {
		return swapInfo, err
	}
	swapInfo.Height = txHeight // Height

	bind, ok := getBindAddressFromMemo(txr.Tx.Body.Memo)
	if !ok {
		return swapInfo, tokens.ErrTxWithWrongMemo
	}
	swapInfo.Bind = bind // Bind

	var from string
	var amount *big.Int
	if token.ContractAddress != "" {
		from, amount, err = b.checkTokenDepist(&txres, token)
	}
	if err != nil {
		return swapInfo, err
	}

	swapInfo.From = strings.ToLower(from) // From
	swapInfo.To = token.DepositAddress    // To
	swapInfo.Value = amount               // Value

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

func (b *Bridge) checkTxStatus(txres *TxResponse, allowUnstable bool) (txHeight uint64, err error) {
	txHeight, err = common.GetUint64FromStr(txres.Height)
	if err != nil {
		return txHeight, err
	}

	if txres.Code != 0 {
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

func (b *Bridge) checkTokenDepist(txres *TxResponse, token *tokens.TokenConfig) (from string, amount *big.Int, err error) {
	var events StringEvents
	events, from = filterEvents(txres, token.ContractAddress)
	if from == "" || len(events) == 0 {
		return "", nil, tokens.ErrDepositLogNotFound
	}
	amount = b.checkEvents(events, token.DepositAddress)
	return from, amount, nil
}

func filterEvents(txres *TxResponse, contractAddress string) (events StringEvents, from string) {
	contractAddressKey := "contract_address"
	for _, log := range txres.Logs {
		for _, event := range log.Events {
			if len(event.Attributes) < 2 {
				continue
			}
			if from == "" &&
				event.Type == "execute_contract" &&
				event.Attributes[1].Key == contractAddressKey &&
				common.IsEqualIgnoreCase(event.Attributes[1].Value, contractAddress) {
				from = event.Attributes[0].Value
			}
			if event.Type == "from_contract" &&
				event.Attributes[1].Key == "action" &&
				event.Attributes[0].Key == contractAddressKey &&
				common.IsEqualIgnoreCase(event.Attributes[0].Value, contractAddress) {
				events = append(events, event)
			}
		}
	}
	return events, from
}

func (b *Bridge) checkEvents(events StringEvents, depositAddress string) (amount *big.Int) {
	amount = big.NewInt(0)
	var toAtIndex, amountAtIndex int
	for _, event := range events {
		switch {
		case event.Attributes[1].Value == "transfer" &&
			common.IsEqualIgnoreCase(event.Attributes[3].Value, depositAddress):
			toAtIndex = 3
			amountAtIndex = 4
		case event.Attributes[1].Value == "transfer_from" &&
			common.IsEqualIgnoreCase(event.Attributes[3].Value, depositAddress):
			toAtIndex = 3
			amountAtIndex = 5
		case event.Attributes[1].Value == "send" &&
			common.IsEqualIgnoreCase(event.Attributes[3].Value, depositAddress):
			toAtIndex = 3
			amountAtIndex = 4
		case event.Attributes[1].Value == "send_from" &&
			common.IsEqualIgnoreCase(event.Attributes[3].Value, depositAddress):
			toAtIndex = 3
			amountAtIndex = 5
		case event.Attributes[1].Value == "mint" &&
			common.IsEqualIgnoreCase(event.Attributes[2].Value, depositAddress):
			toAtIndex = 2
			amountAtIndex = 3
		default:
			continue
		}
		if event.Attributes[toAtIndex].Key != "to" ||
			event.Attributes[amountAtIndex].Key != "amount" {
			continue
		}
		amt, err := common.GetBigIntFromStr(event.Attributes[amountAtIndex].Value)
		if err == nil {
			amount.Add(amount, amt)
		}
	}
	return amount
}

func getBindAddressFromMemo(memo string) (string, bool) {
	bindStr := memo
	if tokens.DstBridge.IsValidAddress(bindStr) {
		return bindStr, true
	}
	return bindStr, false
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
