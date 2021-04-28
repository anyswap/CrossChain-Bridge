package solana

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"strings"

	bin "github.com/dfuse-io/binary"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/programs/system"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

func addSwapInfoConsiderError(swapInfo *tokens.TxSwapInfo, err error, swapInfos *[]*tokens.TxSwapInfo, errs *[]error) {
	if !tokens.ShouldRegisterSwapForError(err) {
		return
	}
	*swapInfos = append(*swapInfos, swapInfo)
	*errs = append(*errs, err)
}

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
	err = bin.NewEncoder(buf).Encode(m)
	if err != nil {
		return err
	}
	messageCnt := buf.Bytes()

	if strings.EqualFold(string(messageCnt), mh) == false {
		return errors.New("msg hash not match")
	}
	return nil
}

// VerifyTransaction impl
func (b *Bridge) VerifyTransaction(pairID, txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	if !b.IsSrc {
		swapInfos, errs := b.verifySwapoutTxWithHash(txHash, allowUnstable)
		// swapinfos have already aggregated
		for i, swapInfo := range swapInfos {
			if strings.EqualFold(swapInfo.PairID, pairID) {
				return swapInfo, errs[i]
			}
		}
		log.Warn("No such swapInfo")
	} else {
		swapInfos, errs := b.verifySwapinTxWithHash(txHash, allowUnstable)
		// swapinfos have already aggregated
		for i, swapInfo := range swapInfos {
			if strings.EqualFold(swapInfo.PairID, pairID) {
				return swapInfo, errs[i]
			}
		}
		log.Warn("No such swapInfo")
	}
	return nil, nil
}

func (b *Bridge) verifySwapinTx(tx *GetConfirmedTransactonResult, allowUnstable bool) (swapInfos []*tokens.TxSwapInfo, errs []error) {
	pairIDs := tokens.GetAllPairIDs()
	if len(pairIDs) == 0 {
		addSwapInfoConsiderError(nil, tokens.ErrTxWithWrongReceiver, &swapInfos, &errs)
		return swapInfos, errs
	}

	for _, pairID := range pairIDs {
		tokenCfg := tokens.GetTokenConfig(pairID, b.IsSrc)

		depositAddress := tokenCfg.DepositAddress
		if tx.Meta.Err != nil {
			swapInfos = append(swapInfos, &tokens.TxSwapInfo{PairID: pairID})
			errs = append(errs, fmt.Errorf("Solana tx error: %v", tx.Meta.Err))
			break
		}
		if len(tx.Transaction.Signatures) < 1 {
			swapInfos = append(swapInfos, &tokens.TxSwapInfo{PairID: pairID})
			errs = append(errs, fmt.Errorf("Unexpected, no signature"))
			break
		}
		for _, ins := range tx.Transaction.Message.Instructions {
			if int(ins.ProgramIDIndex) >= len(tx.Transaction.Message.AccountKeys) {
				swapInfos = append(swapInfos, &tokens.TxSwapInfo{PairID: pairID})
				errs = append(errs, fmt.Errorf("Unexpected, wrong program ID index"))
				continue
			}
			if tx.Transaction.Message.AccountKeys[ins.ProgramIDIndex] != system.PROGRAM_ID {
				swapInfos = append(swapInfos, &tokens.TxSwapInfo{PairID: pairID})
				errs = append(errs, fmt.Errorf("Program ID not match"))
				continue
			}
			if len(ins.Accounts) != 2 {
				swapInfos = append(swapInfos, &tokens.TxSwapInfo{PairID: pairID})
				errs = append(errs, fmt.Errorf("Tx has not enough account keys"))
				continue
			}
			to := tx.Transaction.Message.AccountKeys[ins.Accounts[1]]
			if strings.EqualFold(to.String(), depositAddress) == false {
				swapInfos = append(swapInfos, &tokens.TxSwapInfo{PairID: pairID})
				errs = append(errs, fmt.Errorf("Tx recipient not match"))
				continue
			}
			from := tx.Transaction.Message.AccountKeys[ins.Accounts[0]]
			bind, ok := b.getSolana2ETHSwapinBindAddress(from.String())
			if !ok {
				swapInfos = append(swapInfos, &tokens.TxSwapInfo{PairID: pairID})
				errs = append(errs, fmt.Errorf("Bind address not found or invalid"))
				continue
			}
			if len(ins.Data) < 1 {
				swapInfos = append(swapInfos, &tokens.TxSwapInfo{PairID: pairID})
				errs = append(errs, fmt.Errorf("No transfer data"))
				continue
			}
			if ins.Data[0] != byte(0x2) {
				// Transfer prefix
				swapInfos = append(swapInfos, &tokens.TxSwapInfo{PairID: pairID})
				errs = append(errs, fmt.Errorf("Transfer data prefix is not 0x2: %v", ins.Data[0]))
				continue
			}
			lamports := new(bin.Uint64)
			decoder := bin.NewDecoder(ins.Data[4:])
			err := decoder.Decode(lamports)
			if err != nil {
				swapInfos = append(swapInfos, &tokens.TxSwapInfo{PairID: pairID})
				errs = append(errs, fmt.Errorf("Decode transfer data error: %v", err))
				continue
			}
			value := new(big.Int).SetUint64(uint64(*lamports))
			swapInfo := &tokens.TxSwapInfo{
				PairID:    pairID,
				Hash:      tx.Transaction.Signatures[0].String(),
				Height:    uint64(tx.Slot),
				Timestamp: uint64(tx.BlockTime),
				From:      from.String(),
				TxTo:      to.String(),
				To:        to.String(),
				Bind:      bind,
				Value:     value,
			}
			swapInfos = append(swapInfos, swapInfo)
			errs = append(errs, nil)
		}
	}
	return swapInfos, errs
}

func (b *Bridge) verifySwapinTxWithHash(txid string, allowUnstable bool) (swapInfos []*tokens.TxSwapInfo, errs []error) {
	tx, err := b.GetTransaction(txid)
	if err != nil {
		return nil, []error{err}
	}
	txres, ok := tx.(*GetConfirmedTransactonResult)
	if !ok {
		return nil, []error{errors.New("Solana transaction type error")}
	}
	return b.verifySwapinTx(txres, allowUnstable)
}

func (b *Bridge) verifySwapoutTx(tx *GetConfirmedTransactonResult, allowUnstable bool) (swapInfos []*tokens.TxSwapInfo, errs []error) {
	pairIDs := tokens.GetAllPairIDs()
	if len(pairIDs) == 0 {
		addSwapInfoConsiderError(nil, tokens.ErrTxWithWrongReceiver, &swapInfos, &errs)
		return swapInfos, errs
	}

	for _, pairID := range pairIDs {
		tokenCfg := tokens.GetTokenConfig(pairID, b.IsSrc)

		fmt.Printf("PairID: %v\nToken cfg: %+v\n", pairID, tokenCfg)
		// TODO
	}
	return nil, nil
}

func (b *Bridge) verifySwapoutTxWithHash(txid string, allowUnstable bool) (swapInfos []*tokens.TxSwapInfo, errs []error) {
	tx, err := b.GetTransaction(txid)
	if err != nil {
		return nil, []error{err}
	}
	txres, ok := tx.(*GetConfirmedTransactonResult)
	if !ok {
		return nil, []error{errors.New("Solana transaction type error")}
	}
	return b.verifySwapoutTx(txres, allowUnstable)
}

func (b *Bridge) getSolana2ETHSwapinBindAddress(solanaAddress string) (ethAddress string, ok bool) {
	solanaAddress = strings.ToLower(solanaAddress)
	pkey := SolanaAddressPrefix + solanaAddress
	agreement, err := tools.GetSwapAgreement(pkey)
	if err != nil {
		return "", false
	}
	sp, ok := agreement.(*Solana2ETHSwapinAgreement)
	if !ok {
		return "", false
	}
	ethAddress = sp.ETHBindAddress

	dstBridge := tokens.DstBridge
	if dstBridge.IsValidAddress(ethAddress) {
		return ethAddress, true
	}
	return "", false
}

func (b *Bridge) getETH2SolanaSwapinAgreementBindAddress(ethAddress string) (solanaAddress string, ok bool) {
	ethAddress = strings.ToLower(ethAddress)
	pkey := ETHAddressPrefix + ethAddress
	agreement, err := tools.GetSwapAgreement(pkey)
	if err != nil {
		return "", false
	}
	sp, ok := agreement.(*ETH2SolanaSwapinAgreement)
	if !ok {
		return "", false
	}
	solanaAddress = sp.SolanaBindAddress

	dstBridge := tokens.DstBridge
	if dstBridge.IsValidAddress(solanaAddress) {
		return solanaAddress, true
	}
	return "", false
}

func (b *Bridge) getETH2SolanaSwapoutAgreementBindAddress(solanaAddress string) (ethAddress string, ok bool) {
	solanaAddress = strings.ToLower(solanaAddress)
	pkey := SolanaAddressPrefix + solanaAddress
	agreement, err := tools.GetSwapAgreement(pkey)
	if err != nil {
		return "", false
	}
	sp, ok := agreement.(*ETH2SolanaSwapoutAgreement)
	if !ok {
		return "", false
	}
	ethAddress = sp.ETHBindAddress

	srcBridge := tokens.SrcBridge
	if srcBridge.IsValidAddress(ethAddress) {
		return solanaAddress, true
	}
	return "", false
}
