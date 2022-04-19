package near

import (
	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
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
	return nil, nil
}

// GetTxBlockInfo impl NonceSetter interface
func (b *Bridge) GetTxBlockInfo(txHash string) (blockHeight, blockTime uint64) {
	return 0, 0
}

// VerifyMsgHash verify msg hash
func (b *Bridge) VerifyMsgHash(rawTx interface{}, msgHashes []string) (err error) {
	return nil
}

// VerifyTransaction impl
func (b *Bridge) VerifyTransaction(pairID, txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	return nil, nil
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

//nolint:goconst // allow big check logic
func (b *Bridge) checkCoinDeposit(txres *TxResponse, token *tokens.TokenConfig) (from string, amount uint64, err error) {
	return "", 0, nil
}

func (b *Bridge) checkTokenDepist(txres *TxResponse, token *tokens.TokenConfig) (from string, amount uint64, err error) {
	return "", 0, nil
}

func filterEvents(txres *TxResponse, contractAddress string) (events StringEvents, from string) {
	return nil, ""
}

//nolint:gocyclo // allow big check logic
func (b *Bridge) checkEvents(events StringEvents, depositAddress string) (amount uint64, found bool) {
	return 0, true
}

func getBindAddressFromMemo(memo string) (string, bool) {

	return "", false
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
