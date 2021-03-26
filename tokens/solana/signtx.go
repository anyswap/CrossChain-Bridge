package solana

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	bin "github.com/dfuse-io/binary"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/programs/system"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/dcrm"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

const (
	retryGetSignStatusCount    = 70
	retryGetSignStatusInterval = 10 * time.Second
)

func (b *Bridge) verifyTransactionWithArgs(tx *solana.Transaction, args *tokens.BuildTxArgs) error {
	tokenCfg := b.GetTokenConfig(args.PairID)
	if len(tx.Message.Instructions) != 1 {
		return errors.New("wrong solana instructions length")
	}
	ins := tx.Message.Instructions[0]
	if len(ins.Accounts) < 2 {
		return errors.New("wrong solana transfer account count")
	}
	txprogram := tx.Message.AccountKeys[ins.ProgramIDIndex]
	typeID := ins.Data[0]
	txfrom := tx.Message.AccountKeys[ins.Accounts[0]].String()
	txto := tx.Message.AccountKeys[ins.Accounts[1]].String()
	lamports := new(bin.Uint64)
	decoder := bin.NewDecoder(ins.Data[4:])
	err := decoder.Decode(lamports)
	if err != nil {
		return errors.New("cannot decode solana transfer data")
	}
	txamount := new(big.Int).SetUint64(uint64(*lamports))
	switch {
	case txprogram != system.PROGRAM_ID:
		return errors.New("wrong solana program id")
	case typeID != byte(0x2):
		return errors.New("wrong solana instruction id")
	case strings.EqualFold(txfrom, args.From) == false:
		return errors.New("wrong solana transfer from address")
	case strings.EqualFold(txfrom, tokenCfg.DcrmAddress) == false:
		return errors.New("solana transfer from address is not dcrm address")
	case strings.EqualFold(txto, args.Bind) == false:
		return errors.New("wrong solana transfer to address")
	case txamount.Cmp(args.OriginValue) >= 0:
		return errors.New("solana transfer amount not match")
	default:
	}
	return nil
}

// DcrmSignTransaction dcrm sign raw tx
func (b *Bridge) DcrmSignTransaction(rawTx interface{}, args *tokens.BuildTxArgs) (signedTx interface{}, txHash string, err error) {
	tx, ok := rawTx.(*solana.Transaction)
	if !ok {
		return nil, "", errors.New("wrong raw tx param")
	}
	err = b.verifyTransactionWithArgs(tx, args)
	if err != nil {
		return nil, "", err
	}

	m := tx.Message
	buf := new(bytes.Buffer)
	err = bin.NewEncoder(buf).Encode(m)
	if err != nil {
		return nil, "", err
	}
	msgHash := buf.Bytes()
	jsondata, _ := json.Marshal(args)
	msgContext := string(jsondata)
	rpcAddr, keyID, err := dcrm.DoSignED25519One(b.GetDcrmPublicKey(args.PairID), fmt.Sprintf("%X", msgHash), msgContext)
	if err != nil {
		return nil, "", err
	}
	log.Info(b.ChainConfig.BlockChain+" DcrmSignTransaction start", "keyID", keyID, "msghash", fmt.Sprintf("%X", msgHash), "txid", args.SwapID)
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

	var solanasignature solana.Signature
	copy(solanasignature[:], signature[:64])
	tx.Signatures = append(tx.Signatures, solanasignature)
	signedTx = tx

	txHash = solanasignature.String()
	log.Info(b.ChainConfig.BlockChain+" DcrmSignTransaction success", "keyID", keyID, "txhash", txHash)
	return signedTx, txHash, err
}

// SignTransaction sign tx with pairID
func (b *Bridge) SignTransaction(rawTx interface{}, pairID string) (signedTx interface{}, txHash string, err error) {
	privKey := b.GetTokenConfig(pairID).GetDcrmAddressED25519PrivateKey()
	return b.SignTransactionWithPrivateKey(rawTx, privKey)
}

// SignTransactionWithPrivateKey sign tx with ECDSA private key
func (b *Bridge) SignTransactionWithPrivateKey(rawTx interface{}, privKey *ed25519.PrivateKey) (signedTx interface{}, txHash string, err error) {
	// rawTx is of type authtypes.StdSignDoc
	tx, ok := rawTx.(*solana.Transaction)
	if !ok {
		return nil, "", errors.New("wrong raw tx param")
	}

	m := tx.Message
	buf := new(bytes.Buffer)
	err = bin.NewEncoder(buf).Encode(m)
	if err != nil {
		return nil, "", err
	}
	messageCnt := buf.Bytes()

	p := solana.PrivateKey(*privKey)
	signature, err := p.Sign(messageCnt)
	err = bin.NewEncoder(buf).Encode(m)
	if err != nil {
		return nil, "", err
	}
	tx.Signatures = append(tx.Signatures, signature)

	return tx, signature.String(), nil
}
