package cosmos

import (
	"encoding/hex"
	"errors"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// VerifyMsgHash verify msg hash
func (b *Bridge) VerifyMsgHash(rawTx interface{}, msgHash []string) (err error) {
	// TODO
	stdmsg, ok := rawTx.(authtypes.StdSignMsg)
	if !ok {
		return 
	}
	return nil, errors.New("raw tx type assertion error")
}

// VerifyTransaction impl
func (b *Bridge) VerifyTransaction(pairID, txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	if !b.IsSrc {
		return nil, tokens.ErrBridgeDestinationNotSupported
	}
	return b.verifySwapinTx(pairID, txHash, allowUnstable)
}

func (b *Bridge) verifySwapinTx(pairID, txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	tokenCfg := b.GetTokenConfig(pairID)
	if tokenCfg == nil {
		return nil, tokens.ErrUnknownPairID
	}
	swapInfo := &tokens.TxSwapInfo{}
	swapInfo.PairID = pairID // PairID
	swapInfo.Hash = txHash   // Hash
	swapInfo.TxTo = "" // cosmos tx does not have this field
	if !allowUnstable && !b.checkStable(txHash) {
		return swapInfo, tokens.ErrTxNotStable
	}
	// sdk.Tx
	tx, err := b.GetTransaction(txHash)
	if err != nil {
		log.Debug("[verifySwapin] "+b.ChainConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txHash, "err", err)
		return swapInfo, tokens.ErrTxNotFound
	}
	cosmostx, ok := tx.(sdk.Tx)
	if !ok {
		log.Debug("[verifySwapin] cosmos tx type assertion error")
		return swapInfo, fmt.Errorf("Cosmos tx type assertion error")
	}

	// get bind address from memo
	bindaddress, ok := b.GetBindAddressFromMemo(cosmostx)
	if !ok {
		return swapInfo, fmt.Errorf("Cannot get bind address")
	}
	if err := b.checkSwapinBindAddress(bindaddress); err != nil {
		return swapInfo, err
	}

	// aggregate msgs in tx
	// check every msg
	// if type is bank/send or bank/multisend
	// and To address equals deposit address
	// add to swapinfo
	depositAddress := tokenCfg.DepositAddress
	msgs := cosmostx.GetMsgs()
	depositamount := big.NewInt(0)
	for _, msg :=  range msgs {
		msgtype := msg.Type()
		msgamount := big.NewInt(0) // deposit amount in one msg
		if msgtype == banktypes.TypeMsgSend {
			// MsgSend
			msgsend, ok := msg.(banktypes.MsgSend)
			if !ok {
				continue
			}
			if b.EqualAddress(msgsend.ToAddress, depositAddress) {
				msgamount = b.getAmountFromCoins(msgsend.Amount)
			}
		} else is msgtype == banktypes.TypeMsgMultiSend {
			// MsgMultisend
			msgmultisend, ok := msg.(banktypes.MsgSend)
			if !ok {
				continue
			}
			for _, output := range msgmultisend.Outputs {
				if b.EqualAddress(output.Address, depositAddress){
					msgamount = new(big.Int).Add(msgamount, b.getAmountFromCoins(output.Coins))
				}
			}
		} else {
			continue
		}
		if err := msg.ValidateBasic(); err != nil {
			continue
		}

		if b.EqualAddress(to, depositAddress) {
			// add to tx deposit amount
			depositamount = new(big.Int).Add(depositamount, msgamount)
		}
	}

	swapInfo.To = depositAddress
	swapInfo.Value = depositamount
	swapInfo.Bind = bindaddress
	swapInfo.From = bindaddress
	return swapInfo, nil
}

func (b *Bridge) GetBindAddressFromMemo(tx sdk.Tx) (address string, ok bool) {
	if txWithMemo, ok := (tx).(sdk.TxWithMemo); ok {
		address := txWithMemo.Memo()
		ok = b.IsValidAddress(memo)
		return address, ok
	} else {
		return "", false
	}
}

func (b *Bridge) getAmountFromCoins(coins sdk.Coins) *big.Int {
	amount := big.NewInt(0)
	for _, coin := range coins {
		if strings.EqualFold(coin.Denom, b.TheCoin.Denom) {
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
	isContract, err := b.IsContractAddress(bindAddr)
	if err != nil {
		log.Warn("query is contract address failed", "bindAddr", bindAddr, "err", err)
		return tokens.ErrRPCQueryError
	}
	if isContract {
		return tokens.ErrBindAddrIsContract
	}
	return nil
}