package solana

import (
	"bytes"
	"errors"
	"strings"

	"github.com/dfuse-io/solana-go/programs/system"
	solanarpc "github.com/dfuse-io/solana-go/rpc"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// VerifyMsgHash verify msg hash
func (b *Bridge) VerifyMsgHash(rawTx interface{}, msgHash []string) (err error) {
	tx, ok := rawTx.(*solana.Transaction)
	if !ok {
		return errors.New("verify msg hash tx type error")
	}

	if len(msgHash) < 1 {
		return errors.New("no msg hash")
	}
	mh := msgHash[0]

	m := tx.Message
	buf := new(bytes.Buffer)
	err := bin.NewEncoder(buf).Encode(m)
	if err != nil {
		return err
	}
	messageCnt := buf.Bytes()

	if strings.EqualFold(string(messageCnt), ,mh) == false {
		return errors.New("msg hash not match")
	}
	return nil
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
	return nil, nil
}

func (b *Bridge) verifySwapinTx(pairID string, tx *GetConfirmedTransactonResult, allowUnstable bool) (swapInfos []*tokens.TxSwapInfo, errs []error) {
	tokenCfg := b.GetTokenConfig(pairID)
	depositAddress := tokenCfg.DepositAddress
	if tx.Meta.Err != nil {
		return nil, []error{fmt.Errorf("%+v", tx.Meta.Err)}
	}
	if len(tx.Transaction.Signatures) < 1 {
		return nil, []error{fmt.Errorf("Unexpected error, no signature")}
	}
	for _, ins := range tx.Transaction.Message.Instructions {
		if ins.ProgramIDIndex >= len(tx.Transaction.Message.AccountKeys) {
			continue
		}
		if tx.Transaction.Message.AccountKeys[ins.ProgramIDIndex] != system.PROGRAM_ID {
			continue
		}
		if len(ins.Accounts) != 2 {
			continue
		}
		to := tx.Transaction.Message.AccountKeys[ins.Accounts[1]]
		if strings.EqualFold(to, depositAddress) == false {
			continue
		}
		from := tx.Transaction.Message.AccountKeys[ins.Accounts[0]]
		bind, ok := getBindAddress(from)
		if !ok {
			continue
		}
		if len(ins.Data) < 1 {
			continue
		}
		if ins.Data[0] != byte(0x2) {
			// Transfer prefix
			continue
		}
		lamports := new(bin.Uint64)
		decoder := bin.NewDecoder(ins.Data[4:])
		err = decoder.Decode(lamports)
		if err != nil {
			continue
		}
		value := big.NewInt(uint64(lamports))
		swapInfo := &tokens.TxSwapInfo{
			PairID:    pairID,
			Hash:      tx.Transaction.Signatures[0].String(),
			Height:    uint64(tx.Slot),
			Timestamp: uint64(BlockTime),
			From:      from.String(),
			TxTo:      to.String(),
			To:        to.String(),
			Bind:      bind,
			Value:     value,
		}
		swapInfos = append(swapInfos, swapInfo)
	}
	return swapInfos, nil
}

func (b *Bridge) verifySwapinTxWithHash(pairID, txid string, allowUnstable bool) (swapInfos []*tokens.TxSwapInfo, errs []error) {
	tx, err := b.GetTransaction(txid)
	if err != nil {
		return nil, []error{err}
	}
	return verifySwapinTx(pairID, tx, allowUnstable)
}

func (b *Bridge) getBindAddress(solanaAddress string) (bindAddress string, ok bool) {
	solanaAddress = strings.ToLower(solanaAddress)
	pkey := SolanaDepositAddressPrefix + solanaAddress
	promise, err := GetSwapinPromise(pkey)
	if err != nil {
		return "", false
	}
	sp, ok := promise.(*SolanaSwapinPromise)
	if !ok {
		return "", false
	}
	bindAddress = sp.ETHBindAddress

	dstBridge := tokens.DstBridge
	if dstBridge.IsValidAddress(bindAddress) {
		return bindAddress, true
	}
	return "", false
}
