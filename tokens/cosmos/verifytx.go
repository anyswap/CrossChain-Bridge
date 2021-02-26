package cosmos

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"

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
	if strings.EqualFold(txHash, msgHash[0]) == true {
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
	if len(errs) == 0 {
		return swapInfos[0], nil
	}
	return nil, fmt.Errorf("%+v", errs)
}

func (b *Bridge) verifySwapinTx(pairID string, txresp sdk.TxResponse, allowUnstable bool) (swapInfos []*tokens.TxSwapInfo, errs []error) {
	tokenCfg := b.GetTokenConfig(pairID)
	if tokenCfg == nil {
		return nil, []error{tokens.ErrUnknownPairID}
	}
	swapInfo := &tokens.TxSwapInfo{}
	swapInfo.PairID = pairID      // PairID
	swapInfo.Hash = txresp.TxHash // Hash
	swapInfo.TxTo = ""

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

	// aggregate msgs in tx
	// check every msg
	// if type is bank/send or bank/multisend
	// and To address equals deposit address
	// add to swapinfo
	depositAddress := tokenCfg.DepositAddress
	msgs := cosmostx.GetMsgs()
	depositamount := big.NewInt(0)
	for _, msg := range msgs {
		msgtype := msg.Type()
		msgamount := big.NewInt(0) // deposit amount in one msg
		if msgtype == TypeMsgSend {
			// MsgSend
			msgsend, ok := msg.(MsgSend)
			if !ok {
				continue
			}
			if b.EqualAddress(msgsend.ToAddress.String(), depositAddress) {
				msgamount = b.getAmountFromCoins(msgsend.Amount)
			}
		} else if msgtype == TypeMsgMultiSend {
			// MsgMultisend
			msgmultisend, ok := msg.(MsgMultiSend)
			if !ok {
				continue
			}
			for _, output := range msgmultisend.Outputs {
				if b.EqualAddress(output.Address.String(), depositAddress) {
					msgamount = new(big.Int).Add(msgamount, b.getAmountFromCoins(output.Coins))
				}
			}
		} else {
			continue
		}
		if err := msg.ValidateBasic(); err != nil {
			continue
		}

		depositamount = new(big.Int).Add(depositamount, msgamount)
	}

	swapInfo.To = depositAddress
	swapInfo.Value = depositamount
	swapInfo.Bind = bindaddress
	swapInfo.From = bindaddress
	swapInfos = []*tokens.TxSwapInfo{swapInfo}
	return swapInfos, nil
}

func (b *Bridge) verifySwapinTxWithHash(pairID, txHash string, allowUnstable bool) (swapInfos []*tokens.TxSwapInfo, errs []error) {
	tokenCfg := b.GetTokenConfig(pairID)
	if tokenCfg == nil {
		return nil, []error{tokens.ErrUnknownPairID}
	}
	swapInfo := &tokens.TxSwapInfo{}
	swapInfo.PairID = pairID // PairID
	swapInfo.Hash = txHash   // Hash
	swapInfo.TxTo = ""       // cosmos tx does not have this field
	/*if !allowUnstable && !b.checkStable(txHash) {
		errs = []error{tokens.ErrTxNotStable}
		return swapInfos, errs
	}*/
	// sdk.Tx
	tx, err := b.GetTransaction(txHash)
	if err != nil {
		log.Debug("[verifySwapin] "+b.ChainConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txHash, "err", err)
		errs = []error{tokens.ErrTxNotStable}
		return swapInfos, errs
	}
	cosmostx, ok := tx.(sdk.Tx)
	if !ok {
		log.Debug("[verifySwapin] cosmos tx type assertion error")
		return swapInfos, []error{fmt.Errorf("Cosmos tx type assertion error")}
	}

	// get bind address from memo
	bindaddress, ok := b.GetBindAddressFromMemo(cosmostx)
	if !ok {
		return swapInfos, []error{fmt.Errorf("Cannot get bind address")}
	}
	if err := b.checkSwapinBindAddress(bindaddress); err != nil {
		errs = []error{err}
		return swapInfos, errs
	}

	// aggregate msgs in tx
	// check every msg
	// if type is bank/send or bank/multisend
	// and To address equals deposit address
	// add to swapinfo
	depositAddress := tokenCfg.DepositAddress
	msgs := cosmostx.GetMsgs()
	depositamount := big.NewInt(0)
	for _, msg := range msgs {
		msgtype := msg.Type()
		msgamount := big.NewInt(0) // deposit amount in one msg
		if msgtype == TypeMsgSend {
			// MsgSend
			msgsend, ok := msg.(MsgSend)
			if !ok {
				continue
			}
			if b.EqualAddress(msgsend.ToAddress.String(), depositAddress) {
				msgamount = b.getAmountFromCoins(msgsend.Amount)
			}
		} else if msgtype == TypeMsgMultiSend {
			// MsgMultisend
			msgmultisend, ok := msg.(MsgMultiSend)
			if !ok {
				continue
			}
			for _, output := range msgmultisend.Outputs {
				if b.EqualAddress(output.Address.String(), depositAddress) {
					msgamount = new(big.Int).Add(msgamount, b.getAmountFromCoins(output.Coins))
				}
			}
		} else {
			continue
		}
		if err := msg.ValidateBasic(); err != nil {
			continue
		}

		depositamount = new(big.Int).Add(depositamount, msgamount)
	}

	swapInfo.To = depositAddress
	swapInfo.Value = depositamount
	swapInfo.Bind = bindaddress
	swapInfo.From = bindaddress
	swapInfos = []*tokens.TxSwapInfo{swapInfo}
	return swapInfos, nil
}

func (b *Bridge) GetBindAddressFromMemo(tx sdk.Tx) (address string, ok bool) {
	authtx, ok := tx.(authtypes.StdTx)
	if !ok {
		return "", false
	}
	memo := authtx.Memo
	if ok = b.IsValidAddress(memo); ok {
		return memo, ok
	} else {
		return "", false
	}
}

func (b *Bridge) getAmountFromCoins(coins sdk.Coins) *big.Int {
	amount := big.NewInt(0)
	for _, coin := range coins {
		if strings.EqualFold(coin.Denom, TheCoin.Denom) {
			amount = new(big.Int).Add(amount, coin.Amount.BigInt())
		}
	}
	return amount
}

func (b *Bridge) checkSwapinBindAddress(bindAddr string) error {
	if !tokens.DstBridge.IsValidAddress(bindAddr) {
		log.Warn("wrong bind address in swapin", "bind", bindAddr)
		return tokens.ErrTxWithWrongMemo
	}
	if !tools.IsAddressRegistered(bindAddr) {
		return tokens.ErrTxSenderNotRegistered
	}
	return nil
}
