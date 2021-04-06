package tron

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	proto "github.com/golang/protobuf/proto"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/eth"
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
	tx, ok := rawTx.(*core.Transaction)
	if !ok {
		return errors.New("verify msg hash tx type error")
	}

	if len(msgHash) < 1 {
		return errors.New("no msg hash")
	}
	mh := msgHash[0]

	mh = strings.TrimPrefix(mh, "0x")
	txhash := CalcTxHash(tx)

	if strings.EqualFold(txhash, mh) == false {
		return errors.New("msg hash not match")
	}
	return nil
}

func CalcTxHash(tx *core.Transaction) string {
	inputrawdata := tx.GetRawData()
	rawData, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		return ""
	}

	h256h := sha256.New()
	h256h.Write(rawData)
	hash := h256h.Sum(nil)
	txhash := common.ToHex(hash)

	txhash = strings.TrimPrefix(txhash, "0x")
	return txhash
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

type TransactionExtention struct {
	core.Transaction
	Txid []byte
	BlockNumber uint64
	BlockTime uint64
}

func (b *Bridge) verifySwapinTx(txext *TransactionExtention, allowUnstable bool) (swapInfos []*tokens.TxSwapInfo, errs []error) {
	tx := txext.Transaction

	ret := tx.GetRet()
	if len(ret) != 1 {
		return
	}
	if ret[0].Ret != core.Transaction_Result_SUCESS {
		return
	}

	if err := b.VerifyMsgHash(tx, fmt.Sprintf("%X", txext.Txid)); err != nil {
		addSwapInfoConsiderError(nil, err, &swapInfos, &errs)
		return
	}

	tokenPairsConfig := tokens.GetTokenPairsConfig()

	swapInfo := &tokens.TxSwapInfo{
		Hash: CalcTxHash(tx),
		Height: txext.BlockNumber,
		Timestamp: txext.BlockTime,
	}

	if len(tx.RawData.Contract) != 1 {
		addSwapInfoConsiderError(nil, errors.New("Invalid tron contract"), &swapInfos, &errs)
		return
	}
	contract := tx.RawData.Contract[0]
	switch contract.Type {
	case core.Transaction_Contract_TransferContract:
		// 普通转账
		var c core.TransferContract
		err = ptypes.UnmarshalAny(contract.GetParameter(), &c)
		if err != nil {
			addSwapInfoConsiderError(nil, errors.New("Tx inconsistent"), &swapInfos, &errs)
			return
		}
		toAddress := fmt.Sprintf("%v", tronaddress.Address(c.ToAddress))
		for _, token := range tokenPairsConfig {
			if token.ID == "TRX" {
				depositAddress := token.DepositAddress
				depositAddress = strings.TrimPrefix(depositAddress, "0x")
				if strings.EqualFold(toAddress, depositAddress) {
					swapInfo.PairID = token.ID
				}
			}
		}
		if swapInfo.PairID == "" {
			return
		}
		swapInfo.TxTo = toAddress
		swapInfo.To = toAddress
		swapInfo.From = fmt.Sprintf("%v", tronaddress.Address(OwnerAddress))
		swapInfo.Bind = tronToEth(swapInfo.From)
		swapInfo.Value := big.NewInt(c.Amount)
	case core.Transaction_Contract_TransferAssetContract:
		// TRC10 swapin not supported
		return
	case core.Transaction_Contract_TriggerSmartContract:
		// TRC20
		var c core.TriggerSmartContract
		err = ptypes.UnmarshalAny(contract.GetParameter(), &c)
		if err != nil {
			addSwapInfoConsiderError(nil, errors.New("Tx inconsistent"), &swapInfos, &errs)
			return
		}
		contractAddress := fmt.Sprintf("%v", tronaddress.Address(c.ContractAddress))
		for _, token := range tokenPairsConfig {
			if token.IsTrc20() {
				depositAddress := strings.TrimPrefix(token.DepositAddress)
				tokenContractAddress := strings.TrimPrefix(token.ContractAddress)
				inputData := c.Data
				checkToAddress := tronToEth(contractAddress)
				from := fmt.Sprintf("%v", tronaddress.Address(c.OwnerAddress))
				_, to, value, err := eth.ParseErc20SwapinTxInput(inputData, checkToAddress)
				if err != nil {
					addSwapInfoConsiderError(swapInfo, err, &swapInfos, &errs)
					return
				}
				if strings.EqualFold(tokenContractAddress, contractAddress) && strings.EqualFold(depositAddress, "TRC20 recipient address") {
					swapInfo.PairID = token.ID
				}
			}
			swapInfo.From = from
			swapInfo.TxTo = contractAddress
			swapInfo.Bind = tronToEth(from)
			swapInfo.Value = value
		}
		if swapInfo.PairID == "" {
			return
		}
	default:
		return
	}

	swapInfos = append(swapInfos, swapInfo)
	errs = append(errs, nil)
	return swapInfos, errs
}

func (b *Bridge) verifySwapinTxWithHash(txid string, allowUnstable bool) (swapInfos []*tokens.TxSwapInfo, errs []error) {
	tx, err := b.GetTransaction(txid)
	if err != nil {
		return nil, []error{err}
	}
	txres, ok := tx.(*core.Transaction)
	if !ok {
		return nil, []error{errors.New("Tron transaction type error")}
	}
	txext := TransactionExtention{
		txres,
		Txid: txid,
		BlockNumber: status.BlockNumber,
		BlockTime: status.BlockTime,
	}
	return b.verifySwapinTx(txext, allowUnstable)
}

func (b *Bridge) verifySwapoutTx(txext *TransactionExtention, allowUnstable bool) (swapInfos []*tokens.TxSwapInfo, errs []error) {
	tx := txext.Transaction
	ret := tx.GetRet()
	if len(ret) != 1 {
		return
	}
	if ret[0].Ret != core.Transaction_Result_SUCESS {
		return
	}
	swapInfo := &tokens.TxSwapInfo{
		Hash: CalcTxHash(tx),
		Height: txext.BlockNumber,
		Timestamp: txext.BlockTime,
	}

	tokenPairsConfig := tokens.GetTokenPairsConfig()

	if len(tx.RawData.Contract) != 1 {
		addSwapInfoConsiderError(nil, errors.New("Invalid tron contract"), &swapInfos, &errs)
		return
	}
	contract := tx.RawData.Contract[0]
	switch contract.Type {
	case core.Transaction_Contract_TriggerSmartContract:
		var c core.TriggerSmartContract
		err = ptypes.UnmarshalAny(contract.GetParameter(), &c)
		if err != nil {
			addSwapInfoConsiderError(nil, errors.New("Tx inconsistent"), &swapInfos, &errs)
			return
		}
		contractAddress := fmt.Sprintf("%v", tronaddress.Address(c.ContractAddress))
		if swapInfo.PairID == "" {
			return
		}
		for _, token := range tokenPairsConfig {
			if token.IsTrc20() {
				tokenContractAddress := strings.TrimPrefix(token.ContractAddress)
				if strings.EqualFold(tokenContractAddress, contractAddress) {
					swapInfo.PairID = token.ID
				}
			}
		}
		if swapInfo.PairID == "" {
			return
		}
		inputData := c.Data
		checkToAddress := tronToEth(contractAddress)
		from := fmt.Sprintf("%v", tronaddress.Address(c.OwnerAddress))
		bindAddress, value, err := eth.ParseSwapoutTxInput(inputData, checkToAddress)
		if err != nil {
			addSwapInfoConsiderError(swapInfo, err, &swapInfos, &errs)
			return
		}
		swapInfo.From = from
		swapInfo.TxTo = contractAddress
		swapInfo.Bind = bindAddress
		swapInfo.Value = value
	default:
		return
	}

	return nil, nil
}

func (b *Bridge) verifySwapoutTxWithHash(txid string, allowUnstable bool) (swapInfos []*tokens.TxSwapInfo, errs []error) {
	tx, err := b.GetTransaction(txid)
	if err != nil {
		return nil, []error{err}
	}
	txres, ok := tx.(*core.Transaction)
	if !ok {
		return nil, []error{errors.New("Tron transaction type error")}
	}
	status := GetTransactionStatus(txid)
	txext := TransactionExtention{
		txres,
		Txid: txid,
		BlockNumber: status.BlockNumber,
		BlockTime: status.BlockTime,
	}
	return b.verifySwapoutTx(txext, allowUnstable)
}
