package tron

import (
	"encoding/json"
	"errors"
	"crypto/ecdsa"
	"crypto/sha256"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	tronaddress "github.com/fbsobreira/gotron-sdk/pkg/address"
	proto "github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/tools/crypto"
	"github.com/anyswap/CrossChain-Bridge/dcrm"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/eth"
)

const (
	retryGetSignStatusCount    = 70
	retryGetSignStatusInterval = 10 * time.Second
)

// oracle nodes cannot build an identical Tron tx with BuildTxArgs, which they check signing message against
// instead, they accept the raw tx from BuildTxArgs.TronExtra when rebuilding tx and check everything in this function
func (b *Bridge) verifyTransactionWithArgs(tx *core.Transaction, args *tokens.BuildTxArgs) error {
	tokenCfg := b.GetTokenConfig(args.PairID)
	if tokenCfg == nil {
		return fmt.Errorf("[sign] verify tx with unknown pairID '%v'", args.PairID)
	}
	rawdata := tx.GetRawData()
	contracts := rawdata.GetContract()
	if l := len(contracts); l != 1 {
		return fmt.Errorf("[sign] Tron tx contract number is not 1: %v", l)
	}
	if tokenCfg.IsTrc20() {
		// TRC20
		var contract core.TriggerSmartContract
		err := ptypes.UnmarshalAny(contracts[0].GetParameter(), &contract)
		if err != nil {
			return fmt.Errorf("[sign] Decode tron contract error: %v", err)
		}
		txFrom := tronaddress.Address(contract.OwnerAddress).String()
		if EqualAddress(txFrom, args.From) == false {
			return fmt.Errorf("[sign] TRC20 transfer with wrong from address")
		}
		txRecipient := tronaddress.Address(contract.ContractAddress).String()
		if EqualAddress(txRecipient, args.To) == false {
			return fmt.Errorf("[sign] TRC20 transfer with wrong recipient")
		}
		if EqualAddress(txRecipient, tokenCfg.ContractAddress) == false {
			return fmt.Errorf("[sign] TRC20 transfer recipient is not token contract address")
		}
		input := rawdata.GetData()
		bindAddress, value, err := eth.ParseSwapoutTxInput(&input)
		if err != nil {
			return fmt.Errorf("[sign] TRC20 transfer with wrong input data: %v", err)
		}
		if EqualAddress(args.Bind, bindAddress) == false {
			return fmt.Errorf("[sign] TRC20 transfer with wrong bind address")
		}
		if args.Value.Cmp(value) != 0 {
			return fmt.Errorf("[sign] TRC20 transfer with wrong value")
		}
	} else {
		// Not TRC20
		var contract core.TransferContract
		err := ptypes.UnmarshalAny(contracts[0].GetParameter(), &contract)
		if err != nil {
			return fmt.Errorf("[sign] Decode tron contract error: %v", err)
		}
		txFrom := tronaddress.Address(contract.OwnerAddress).String()
		if strings.EqualFold(txFrom, args.From) == false {
			return fmt.Errorf("[sign] TRX transfer with wrong from address")
		}
		txRecipient := tronaddress.Address(contract.ToAddress).String()
		if strings.EqualFold(txRecipient, args.To) == false {
			return fmt.Errorf("[sign] TRX transfer with wrong recipient")
		}
		if strings.EqualFold(txRecipient, tokenCfg.ContractAddress) == false {
			return fmt.Errorf("[sign] TRX transfer recipient is not token contract address")
		}
		if args.Value.Cmp(big.NewInt(contract.Amount)) != 0 {
			return fmt.Errorf("[sign] TRX transfer with wrong value")
		}
	}
	return nil
}

// DcrmSignTransaction dcrm sign raw tx
func (b *Bridge) DcrmSignTransaction(rawTx interface{}, args *tokens.BuildTxArgs) (signedTx interface{}, txHash string, err error) {
	tx, ok := rawTx.(*core.Transaction)
	if !ok {
		return nil, "", errors.New("wrong raw tx param")
	}
	err = b.verifyTransactionWithArgs(tx, args)
	if err != nil {
		return nil, "", err
	}

	txHash = CalcTxHash(tx)
	jsondata, _ := json.Marshal(args)
	msgContext := string(jsondata)
	rpcAddr, keyID, err := dcrm.DoSignOne(b.GetDcrmPublicKey(args.PairID), txHash, msgContext)
	if err != nil {
		return nil, "", err
	}
	log.Info(b.ChainConfig.BlockChain+" DcrmSignTransaction start", "keyID", keyID, "msghash", txHash, "txid", args.SwapID, "data", msgContext)
	time.Sleep(retryGetSignStatusInterval)

	var rsv string
	i := 0
	for ; i < retryGetSignStatusCount; i++ {
		signStatus, err2 := dcrm.GetSignStatus(keyID, rpcAddr)
		if err2 == nil {
			if len(signStatus.Rsv) != 1 {
				return nil, "", fmt.Errorf("get sign status require one rsv but have %v (keyID = %v)", len(signStatus.Rsv), keyID)
			}

			rsv = signStatus.Rsv[0]
			break
		}
		switch err2 {
		case dcrm.ErrGetSignStatusFailed, dcrm.ErrGetSignStatusTimeout:
			return nil, "", err2
		}
		log.Warn("retry get sign status as error", "err", err2, "txid", args.SwapID, "keyID", keyID, "bridge", args.Identifier, "swaptype", args.SwapType.String())
		time.Sleep(retryGetSignStatusInterval)
	}
	if i == retryGetSignStatusCount || rsv == "" {
		return nil, "", errors.New("get sign status failed")
	}

	log.Trace(b.ChainConfig.BlockChain+" DcrmSignTransaction get rsv success", "keyID", keyID, "rsv", rsv)

	signature := common.FromHex(rsv)


	tx.Signature = append(tx.Signature, signature)
	signedTx = tx
	log.Info(b.ChainConfig.BlockChain+" DcrmSignTransaction success", "keyID", keyID, "txhassh", txHash)
	return signedTx, txHash, err
}

// SignTransaction sign tx with pairID
func (b *Bridge) SignTransaction(rawTx interface{}, pairID string) (signedTx interface{}, txHash string, err error) {
	privKey := b.GetTokenConfig(pairID).GetDcrmAddressPrivateKey()
	return b.SignTransactionWithPrivateKey(rawTx, privKey)
}

// SignTransactionWithPrivateKey sign tx with ECDSA private key
func (b *Bridge) SignTransactionWithPrivateKey(rawTx interface{}, privKey *ecdsa.PrivateKey) (signedTx interface{}, txHash string, err error) {
	// rawTx is of type authtypes.StdSignDoc
	tx, ok := rawTx.(*core.Transaction)
	if !ok {
		return nil, "", errors.New("wrong raw tx param")
	}

	rawData, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		return nil, "", err
	}
	h256h := sha256.New()
	h256h.Write(rawData)
	hash := h256h.Sum(nil)
	txhash := fmt.Sprintf("%X", hash)

	signature, err := crypto.Sign(hash, privKey)
	if err != nil {
		return nil, "", err
	}
	tx.Signature = append(tx.Signature, signature)
	signedTx = tx
	return tx, txhash, nil
}
