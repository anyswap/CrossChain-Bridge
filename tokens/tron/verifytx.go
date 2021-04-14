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
	"github.com/golang/protobuf/ptypes"
	proto "github.com/golang/protobuf/proto"
	tronaddress "github.com/fbsobreira/gotron-sdk/pkg/address"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/eth"
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

func GetTxData(tx *core.Transaction) []byte {
	rawData, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		return []byte{}
	}
	return rawData
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
		log.Warn("No swapInfo", "errors", errs)
	} else {
		swapInfos, errs := b.verifySwapinTxWithHash(txHash, allowUnstable)
		// swapinfos have already aggregated
		for i, swapInfo := range swapInfos {
			if strings.EqualFold(swapInfo.PairID, pairID) {
				return swapInfo, errs[i]
			}
		}
		log.Warn("No swapInfo", "errors", errs)
	}
	return &tokens.TxSwapInfo{}, errors.New("Cannot generate swapinfo")
}

type TransactionExtention struct {
	core.Transaction
	Txid []byte
	BlockNumber uint64
	BlockTime uint64
}

func (b *Bridge) verifySwapinTx(txext *TransactionExtention, allowUnstable bool) (swapInfos []*tokens.TxSwapInfo, errs []error) {
	tx := &txext.Transaction

	ret := tx.GetRet()
	if len(ret) != 1 {
		return []*tokens.TxSwapInfo{&tokens.TxSwapInfo{}}, []error{errors.New("Tron tx return not found")}
	}
	if txret := ret[0].GetRet(); txret != core.Transaction_Result_SUCESS {
		return []*tokens.TxSwapInfo{&tokens.TxSwapInfo{}}, []error{fmt.Errorf("Tron tx not success: %v", txret)}
	}

	if cret := ret[0].GetContractRet(); cret == core.Transaction_Result_SUCCESS {
		return []*tokens.TxSwapInfo{&tokens.TxSwapInfo{}}, []error{fmt.Errorf("Tron contract not success: %v", cret)}
	}

	if err := b.VerifyMsgHash(tx, []string{fmt.Sprintf("%X", txext.Txid)}); err != nil {
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
		addSwapInfoConsiderError(nil, errors.New("Tron contract is not 1"), &swapInfos, &errs)
		return
	}
	contract := tx.RawData.Contract[0]
	switch contract.Type {
	case core.Transaction_Contract_TransferContract:
		// 普通转账
		var c core.TransferContract
		err := ptypes.UnmarshalAny(contract.GetParameter(), &c)
		if err != nil {
			addSwapInfoConsiderError(nil, errors.New("Tx inconsistent"), &swapInfos, &errs)
			return
		}
		toAddress := fmt.Sprintf("%v", tronaddress.Address(c.ToAddress))
		for _, tokenPair := range tokenPairsConfig {
			token := tokenPair.SrcToken
			if tokenPair.PairID == "TRX" {
				depositAddress := token.DepositAddress
				if strings.EqualFold(toAddress, depositAddress) {
					swapInfo.PairID = tokenPair.PairID
				}
			}
		}
		if swapInfo.PairID == "" {
			addSwapInfoConsiderError(nil, errors.New("Invalid TRX swapin"), &swapInfos, &errs)
			return
		}
		swapInfo.TxTo = toAddress
		swapInfo.To = toAddress
		swapInfo.From = fmt.Sprintf("%v", tronaddress.Address(c.OwnerAddress))
		swapInfo.Bind, _ = tronToEth(swapInfo.From)
		swapInfo.Value = big.NewInt(c.Amount)
	case core.Transaction_Contract_TransferAssetContract:
		// TRC10 swapin not supported
		addSwapInfoConsiderError(nil, errors.New("TRC10 not supported"), &swapInfos, &errs)
		return
	case core.Transaction_Contract_TriggerSmartContract:
		// TRC20
		var c core.TriggerSmartContract
		err := ptypes.UnmarshalAny(contract.GetParameter(), &c)
		if err != nil {
			addSwapInfoConsiderError(nil, errors.New("Tx inconsistent"), &swapInfos, &errs)
			return
		}
		contractAddress := fmt.Sprintf("%v", tronaddress.Address(c.ContractAddress))
		for _, tokenPair := range tokenPairsConfig {
			token := tokenPair.SrcToken
			if token.IsTrc20() {
				depositAddress := token.DepositAddress
				tokenContractAddress := token.ContractAddress
				inputData := c.Data
				checkToAddress, _ := tronToEth(depositAddress)
				from := fmt.Sprintf("%v", tronaddress.Address(c.OwnerAddress))
				_, to, value, err := eth.ParseErc20SwapinTxInput(&inputData, checkToAddress)
				if err != nil {
					addSwapInfoConsiderError(swapInfo, err, &swapInfos, &errs)
					break
				}
				transferTo, _ := ethToTron(to)
				if EqualAddress(tokenContractAddress, contractAddress) && EqualAddress(depositAddress, transferTo) {
					txStatus := b.GetTransactionStatus(fmt.Sprintf("%+X", txext.Txid))
					recpt, ok := txStatus.Receipt.(*core.TransactionInfo)
					if !ok {
						addSwapInfoConsiderError(swapInfo, errors.New("Get tron tx receipt error"), &swapInfos, &errs)
						break
					}
					txlogs := recpt.GetLog()
					if len(txlogs) < 1 {
						addSwapInfoConsiderError(swapInfo, errors.New("No contract log"), &swapInfos, &errs)
						break
					}
					logfrom, logto, logamount, err := checkErc20TransferLog(txlogs, contractAddress)
					if err != nil {
						addSwapInfoConsiderError(swapInfo, err, &swapInfos, &errs)
						break
					}
					if EqualAddress(logfrom, from) == false || EqualAddress(logto, depositAddress) == false || logamount.Cmp(value) != 0 {
						addSwapInfoConsiderError(swapInfo, errors.New("Contract log mismatch"), &swapInfos, &errs)
						break
					}

					swapInfo.PairID = tokenPair.PairID
					swapInfo.From = from
					swapInfo.Bind, _ = tronToEth(swapInfo.From) // Use eth format
					swapInfo.TxTo = contractAddress
					swapInfo.Value = value
				}
			}
		}
		if swapInfo.PairID == "" {
			addSwapInfoConsiderError(nil, errors.New("Invalid TRC20 swapin"), &swapInfos, &errs)
			return
		}
	default:
		addSwapInfoConsiderError(nil, errors.New("Unknown error"), &swapInfos, &errs)
		return
	}

	swapInfos = append(swapInfos, swapInfo)
	errs = append(errs, nil)
	return swapInfos, errs
}

func checkErc20TransferLog (txlogs []*core.TransactionInfo_Log, contractAddress string) (logfrom, logto string, logamount *big.Int, err error) {
	logamount = new(big.Int)
	hasAddr := false
	for _, txlog := range txlogs {
		for _, addr := range txlog.GetAddress() {
			if EqualAddress(fmt.Sprintf("%X", addr), contractAddress) {
				hasAddr = true
				topics := txlog.GetTopics()
				if len(topics) < 3 {
					return "", "", nil, errors.New("Log topic number error")
				}
				if bytes.Equal(topics[0], ExtCodeParts["LogTransfer"]) == false {
					return "", "", nil, errors.New("Log topic number error")
				}
				logfrom = fmt.Sprintf("%X", topics[1])
				logto = fmt.Sprintf("%X", topics[2])
				amthex := fmt.Sprintf("%X", txlog.GetData())
				logamount, _ = logamount.SetString(amthex, 16)
				break
			}
		}
	}
	if hasAddr == false {
		return "", "", nil, errors.New("Logs do not contain contract address")
	}
	return
}

func (b *Bridge) verifySwapinTxWithHash(txid string, allowUnstable bool) (swapInfos []*tokens.TxSwapInfo, errs []error) {
	tx, err := b.GetTransaction(txid)
	if err != nil {
		return []*tokens.TxSwapInfo{&tokens.TxSwapInfo{}}, []error{err}
	}
	txres, ok := tx.(*core.Transaction)
	if !ok {
		return []*tokens.TxSwapInfo{&tokens.TxSwapInfo{}}, []error{errors.New("Tron transaction type error")}
	}
	status := b.GetTransactionStatus(txid)
	txext := &TransactionExtention{
		Transaction: *txres,
		BlockNumber: status.BlockHeight,
		BlockTime: status.BlockTime,
	}
	txext.Txid, _ = common.FromHex(txid)
	return b.verifySwapinTx(txext, allowUnstable)
}

func (b *Bridge) verifySwapoutTx(txext *TransactionExtention, allowUnstable bool) (swapInfos []*tokens.TxSwapInfo, errs []error) {
	tx := &txext.Transaction
	ret := tx.GetRet()
	if len(ret) != 1 {
		return nil, []error{errors.New("Tron tx return not found")}
	}
	if txret := ret[0].GetRet(); txret != core.Transaction_Result_SUCESS {
		return nil, []error{fmt.Errorf("Tron tx not success: %+v", txret)}
	}
	if cret := ret[0].GetContractRet(); cret != core.Transaction_Result_SUCCESS {
		return nil, []error{fmt.Errorf("Tron tx contract not success: %+v", cret)}
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
		err := ptypes.UnmarshalAny(contract.GetParameter(), &c)
		if err != nil {
			addSwapInfoConsiderError(nil, errors.New("Tx inconsistent"), &swapInfos, &errs)
			return
		}
		contractAddress := fmt.Sprintf("%v", tronaddress.Address(c.ContractAddress))
		for _, tokenPair := range tokenPairsConfig {
			token := tokenPair.DestToken
				tokenContractAddress := token.ContractAddress
				if strings.EqualFold(tokenContractAddress, contractAddress) {
					swapInfo.PairID = tokenPair.PairID
				}
		}
		if swapInfo.PairID == "" {
			addSwapInfoConsiderError(nil, errors.New("Invalid swapout"), &swapInfos, &errs)
			return
		}
		inputData := c.Data
		from := fmt.Sprintf("%v", tronaddress.Address(c.OwnerAddress))
		bindAddress, value, err := eth.ParseSwapoutTxInput(&inputData)
		if err != nil {
			addSwapInfoConsiderError(swapInfo, err, &swapInfos, &errs)
			return
		}
		swapInfo.From = from
		swapInfo.TxTo = contractAddress
		swapInfo.Bind = bindAddress
		swapInfo.Value = value
	default:
		addSwapInfoConsiderError(nil, errors.New("Unknown error"), &swapInfos, &errs)
		return
	}

	swapInfos = append(swapInfos, swapInfo)
	errs = append(errs, nil)
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
	status := b.GetTransactionStatus(txid)
	txext := &TransactionExtention{
		Transaction: *txres,
		BlockNumber: status.BlockHeight,
		BlockTime: status.BlockTime,
	}
	txext.Txid, _ = common.FromHex(txid)
	return b.verifySwapoutTx(txext, allowUnstable)
}
