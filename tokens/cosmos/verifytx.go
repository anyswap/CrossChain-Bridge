package cosmos

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"

	//"github.com/anyswap/CrossChain-Bridge/tokens/tools"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// VerifyMsgHash verify msg hash
func (b *Bridge) VerifyMsgHash(rawTx interface{}, msgHash []string) (err error) {
	tx, ok := rawTx.(StdSignContent)
	if !ok {
		return errors.New("raw tx type assertion error")
	}

	txHash := tx.Hash()
	if strings.EqualFold(txHash, msgHash[0]) {
		return nil
	}
	return errors.New("msg hash not match")
}

// VerifyTransaction impl
func (b *Bridge) VerifyTransaction(pairID, txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	if !b.IsSrc {
		return nil, tokens.ErrBridgeDestinationNotSupported
	}
	swapInfos, errs := b.verifySwapinTxWithHash(pairID, txHash, allowUnstable)
	// swapinfos have already aggregated
	for i, swapInfo := range swapInfos {
		if strings.EqualFold(swapInfo.PairID, pairID) {
			return swapInfo, errs[i]
		}
	}
	log.Warn("No such swapInfo")
	return nil, fmt.Errorf("No swap info, errors: %v", errs)
}

func (b *Bridge) verifySwapinTx(txresp sdk.TxResponse, allowUnstable bool) (swapInfos []*tokens.TxSwapInfo, errs []error) {
	swapInfos = make([]*tokens.TxSwapInfo, 0)
	swapInfoMap := make(map[string][]*tokens.TxSwapInfo)
	txid := strings.ToLower(txresp.TxHash)
	cosmostx := txresp.Tx

	// get bind address from memo
	bindaddress, ok := b.GetBindAddressFromMemo(cosmostx)
	if !ok {
		return swapInfos, []error{fmt.Errorf("Cannot get bind address")}
	}
	if err := b.checkSwapinBindAddress(bindaddress); err != nil {
		errs = []error{err}
		return swapInfos, errs
	}

	// check every msg
	// if type is bank/send or bank/multisend, check every coin in every output
	// add to swapinfo
	msgs := cosmostx.GetMsgs()
	for _, msg := range msgs {
		if err := msg.ValidateBasic(); err != nil {
			continue
		}
		msgtype := msg.Type()
		if msgtype == TypeMsgSend {
			// MsgSend
			msgsend, ok := msg.(MsgSend)
			if !ok {
				continue
			}

			for _, coin := range msgsend.Amount {
				pairID, err := b.getPairID(coin)
				if err != nil {
					continue
				}
				tokenCfg := b.GetTokenConfig(pairID)
				if tokenCfg == nil {
					continue
				}
				if !b.EqualAddress(msgsend.ToAddress.String(), tokenCfg.DepositAddress) {
					continue
				}
				swapInfo := &tokens.TxSwapInfo{}
				swapInfo.PairID = pairID
				swapInfo.To = tokenCfg.DepositAddress
				swapInfo.Bind = bindaddress
				swapInfo.From = bindaddress
				swapInfo.Value = coin.Amount.BigInt()
				if swapInfoMap[pairID] == nil {
					swapInfoMap[pairID] = make([]*tokens.TxSwapInfo, 0)
				}
				swapInfoMap[pairID] = append(swapInfoMap[pairID], swapInfo)
			}

		} else if msgtype == TypeMsgMultiSend {
			// MsgMultisend
			msgmultisend, ok := msg.(MsgMultiSend)
			if !ok {
				continue
			}
			for _, output := range msgmultisend.Outputs {
				for _, coin := range output.Coins {
					pairID, err := b.getPairID(coin)
					if err != nil {
						continue
					}
					tokenCfg := b.GetTokenConfig(pairID)
					if tokenCfg == nil {
						continue
					}
					if !b.EqualAddress(output.Address.String(), tokenCfg.DepositAddress) {
						continue
					}
					swapInfo := &tokens.TxSwapInfo{}
					swapInfo.PairID = pairID
					swapInfo.To = tokenCfg.DepositAddress
					swapInfo.Bind = bindaddress
					swapInfo.From = bindaddress
					swapInfo.Value = coin.Amount.BigInt()
					if swapInfoMap[pairID] == nil {
						swapInfoMap[pairID] = make([]*tokens.TxSwapInfo, 0)
					}
					swapInfoMap[pairID] = append(swapInfoMap[pairID], swapInfo)
				}
			}
		} else {
			continue
		}
	}

	// aggregate by pairID
	for k, v := range swapInfoMap {
		if len(v) < 1 {
			continue
		}
		aggSwapInfo := &tokens.TxSwapInfo{}
		aggSwapInfo.PairID = k
		aggSwapInfo.To = v[0].To
		aggSwapInfo.Bind = v[0].Bind
		aggSwapInfo.Hash = txid
		aggSwapInfo.Value = big.NewInt(0)
		for _, swapInfo := range v {
			aggSwapInfo.Value = new(big.Int).Add(aggSwapInfo.Value, swapInfo.Value)
		}
		swapInfos = append(swapInfos, aggSwapInfo)
		errs = append(errs, nil)
	}

	return swapInfos, errs
}

// NotSupportedCoinErr is an error
var NotSupportedCoinErr = errors.New("coin not supported")

// getPairID returns pairID corresponding to given coin
// returns error when coin type not supported
func (b *Bridge) getPairID(coin sdk.Coin) (string, error) {
	for k, v := range b.SupportedCoins {
		if strings.EqualFold(v.Denom, coin.Denom) {
			return strings.ToLower(k), nil
		}
	}
	// if not exists, reload coins and find it
	b.LoadCoins()
	for k, v := range b.SupportedCoins {
		if strings.EqualFold(v.Denom, coin.Denom) {
			return strings.ToLower(k), nil
		}
	}
	return "", NotSupportedCoinErr
}

func (b *Bridge) verifySwapinTxWithHash(pairID, txHash string, allowUnstable bool) (swapInfos []*tokens.TxSwapInfo, errs []error) {
	log.Warn("!!!! 000000", "pairID", pairID)
	txid := strings.ToLower(txHash)
	tx, err := b.GetTransaction(txHash)
	log.Warn("!!!! 111111", "tx", tx, "err", err)
	if err != nil {
		log.Debug("[verifySwapin] "+b.ChainConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txHash, "err", err)
		errs = []error{tokens.ErrTxNotStable}
		return nil, errs
	}
	cosmostx, ok := tx.(sdk.Tx)
	log.Warn("!!!! 222222", "tx", tx, "cosmostx", ok)
	if !ok {
		log.Debug("[verifySwapin] "+b.ChainConfig.BlockChain+" Bridge::Transacton is of wrong type", "tx", txHash)
		return nil, []error{errors.New("Tx is of wrong type")}
	}
	swapInfos = make([]*tokens.TxSwapInfo, 0)
	swapInfoMap := make(map[string][]*tokens.TxSwapInfo)

	// get bind address from memo
	bindaddress, ok := b.GetBindAddressFromMemo(cosmostx)
	log.Warn("!!!! 333333", "bindaddress", bindaddress)
	if !ok {
		return swapInfos, []error{fmt.Errorf("Cannot get bind address")}
	}
	if err := b.checkSwapinBindAddress(bindaddress); err != nil {
		errs = []error{err}
		return swapInfos, errs
	}

	// check every msg
	// if type is bank/send or bank/multisend, check every coin in every output
	// add to swapinfo
	msgs := cosmostx.GetMsgs()
	for _, msg := range msgs {
		if err := msg.ValidateBasic(); err != nil {
			continue
		}
		msgtype := msg.Type()
		if msgtype == TypeMsgSend {
			// MsgSend
			msgsend, ok := msg.(MsgSend)
			if !ok {
				continue
			}

			for _, coin := range msgsend.Amount {
				log.Warn("!!!! 444444", "coin", coin)
				pairID, err := b.getPairID(coin)
				log.Warn("!!!! 555555", "pairID", pairID)
				if err != nil {
					continue
				}
				tokenCfg := b.GetTokenConfig(pairID)
				if tokenCfg == nil {
					continue
				}
				if !b.EqualAddress(msgsend.ToAddress.String(), tokenCfg.DepositAddress) {
					continue
				}
				swapInfo := &tokens.TxSwapInfo{}
				swapInfo.PairID = pairID
				swapInfo.To = tokenCfg.DepositAddress
				swapInfo.Bind = bindaddress
				swapInfo.From = msgsend.FromAddress.String()
				swapInfo.Value = coin.Amount.BigInt()
				if swapInfoMap[pairID] == nil {
					swapInfoMap[pairID] = make([]*tokens.TxSwapInfo, 0)
				}
				swapInfoMap[pairID] = append(swapInfoMap[pairID], swapInfo)
			}

		} else if msgtype == TypeMsgMultiSend {
			// MsgMultisend
			msgmultisend, ok := msg.(MsgMultiSend)
			if !ok {
				continue
			}
			for _, output := range msgmultisend.Outputs {
				for _, coin := range output.Coins {
					pairID, err := b.getPairID(coin)
					if err != nil {
						continue
					}
					tokenCfg := b.GetTokenConfig(pairID)
					if tokenCfg == nil {
						continue
					}
					if !b.EqualAddress(output.Address.String(), tokenCfg.DepositAddress) {
						continue
					}
					swapInfo := &tokens.TxSwapInfo{}
					swapInfo.PairID = pairID
					swapInfo.To = tokenCfg.DepositAddress
					swapInfo.Bind = bindaddress
					//swapInfo.From = bindaddress
					swapInfo.Value = coin.Amount.BigInt()
					if swapInfoMap[pairID] == nil {
						swapInfoMap[pairID] = make([]*tokens.TxSwapInfo, 0)
					}
					swapInfoMap[pairID] = append(swapInfoMap[pairID], swapInfo)
				}
			}
		} else {
			continue
		}
	}

	// aggregate by pairID
	for k, v := range swapInfoMap {
		if len(v) < 1 {
			continue
		}
		aggSwapInfo := &tokens.TxSwapInfo{}
		aggSwapInfo.PairID = k
		aggSwapInfo.Hash = txid
		aggSwapInfo.To = v[0].To
		aggSwapInfo.Bind = v[0].Bind
		aggSwapInfo.From = v[0].From
		// aggSwapInfo.TxId = v[0].TxId
		aggSwapInfo.Value = big.NewInt(0)
		for _, swapInfo := range v {
			aggSwapInfo.Value = new(big.Int).Add(aggSwapInfo.Value, swapInfo.Value)
		}
		swapInfos = append(swapInfos, aggSwapInfo)
		errs = append(errs, nil)
	}

	return swapInfos, errs
}

// GetBindAddressFromMemo get tx memo from an sdk.Tx
func (b *Bridge) GetBindAddressFromMemo(tx sdk.Tx) (address string, ok bool) {
	authtx, ok := tx.(authtypes.StdTx)
	if !ok {
		log.Warn("GetBindAddressFromMemo: Tx is not auth StdTx", "Tx", tx)
		return "", false
	}
	memo := authtx.Memo
	dstBridge := tokens.DstBridge
	if ok = dstBridge.IsValidAddress(memo); ok {
		memo = strings.ToLower(memo)
		return memo, ok
	} else {
		log.Warn("GetBindAddressFromMemo: memo is not a valid address", "memo", memo)
		return "", false
	}
}

func (b *Bridge) checkSwapinBindAddress(bindAddr string) error {
	if !tokens.DstBridge.IsValidAddress(bindAddr) {
		log.Warn("wrong bind address in swapin", "bind", bindAddr)
		return tokens.ErrTxWithWrongMemo
	}
	/*if !tools.IsAddressRegistered(bindAddr) {
		return tokens.ErrTxSenderNotRegistered
	}*/
	return nil
}
