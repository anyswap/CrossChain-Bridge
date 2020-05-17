package btc

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/wallet/txauthor"
	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/dcrm"
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	"github.com/fsn-dev/crossChain-Bridge/tools/crypto"
)

var (
	retryCount    = 15
	retryInterval = 10 * time.Second
	waitInterval  = 10 * time.Second

	hashType = txscript.SigHashAll
)

func (b *BtcBridge) DcrmSignTransaction(rawTx interface{}, args *tokens.BuildTxArgs) (signedTx interface{}, err error) {
	authoredTx, ok := rawTx.(*txauthor.AuthoredTx)
	if !ok {
		return nil, tokens.ErrWrongRawTx
	}

	var (
		tx        = authoredTx.Tx
		msgHashes []string
		rsvs      []string
	)

	for i, pkscript := range authoredTx.PrevScripts {
		sigHash, err := txscript.CalcSignatureHash(pkscript, hashType, tx, i)
		if err != nil {
			return nil, err
		}
		msgHash := hex.EncodeToString(sigHash)
		msgHashes = append(msgHashes, msgHash)
	}

	for idx, msgHash := range msgHashes {
		rsv, err := b.DcrmSignMsgHash(msgHash, args, idx)
		if err != nil {
			return nil, err
		}
		rsvs = append(rsvs, rsv)
	}

	return b.MakeSignedTransaction(authoredTx, msgHashes, rsvs, args)
}

func (b *BtcBridge) verifyPublickeyData(pkData []byte, swapType tokens.SwapType) error {
	switch swapType {
	case tokens.Swap_Swapin:
		return tokens.ErrSwapTypeNotSupported
	case tokens.Swap_Swapout, tokens.Swap_Recall:
		dcrmAddress := b.TokenConfig.DcrmAddress
		address, _ := btcutil.NewAddressPubKeyHash(btcutil.Hash160(pkData), b.GetChainConfig())
		if address.EncodeAddress() != b.TokenConfig.DcrmAddress {
			return fmt.Errorf("sign public key %v is not the configed dcrm address %v", address, dcrmAddress)
		}
	}
	return nil
}

func (b *BtcBridge) MakeSignedTransaction(authoredTx *txauthor.AuthoredTx, msgHash []string, rsv []string, args *tokens.BuildTxArgs) (signedTx interface{}, err error) {
	txIn := authoredTx.Tx.TxIn
	if len(txIn) != len(msgHash) {
		return nil, errors.New("mismatch number of msghashes and tx inputs")
	}
	if len(txIn) != len(rsv) {
		return nil, errors.New("mismatch number of signatures and tx inputs")
	}
	log.Info("BtcBridge MakeSignedTransaction", "msghash", msgHash, "count", len(msgHash))

	var cPkData []byte

	if args != nil && args.Extra != nil {
		extra, ok := args.Extra.(*tokens.BtcExtraArgs)
		if !ok {
			return nil, tokens.ErrWrongExtraArgs
		}
		cPkData = common.FromHex(*extra.FromPublicKey)
		if err := b.verifyPublickeyData(cPkData, args.SwapType); err != nil {
			return nil, err
		}
	}

	for i, txin := range txIn {
		l := len(rsv[i]) - 2
		rs := rsv[i][0:l]

		r := rs[:64]
		s := rs[64:]

		rr, _ := new(big.Int).SetString(r, 16)
		ss, _ := new(big.Int).SetString(s, 16)

		sign := &btcec.Signature{
			R: rr,
			S: ss,
		}

		signData := append(sign.Serialize(), byte(hashType))

		if len(cPkData) == 0 {
			rsvData := common.FromHex(rsv[i])
			hashData := common.FromHex(msgHash[i])
			pkData, err := crypto.Ecrecover(hashData, rsvData)
			if err != nil {
				return nil, err
			}
			pk, _ := btcec.ParsePubKey(pkData, btcec.S256())
			cPkData = pk.SerializeCompressed()
			if err := b.verifyPublickeyData(cPkData, args.SwapType); err != nil {
				return nil, err
			}
		}

		sigScript, err := txscript.NewScriptBuilder().AddData(signData).AddData(cPkData).Script()
		if err != nil {
			return nil, err
		}
		txin.SignatureScript = sigScript
	}
	return authoredTx, nil
}

func (b *BtcBridge) DcrmSignMsgHash(msgHash string, args *tokens.BuildTxArgs, idx int) (rsv string, err error) {
	extra, ok := args.Extra.(*tokens.BtcExtraArgs)
	if !ok {
		return "", tokens.ErrWrongExtraArgs
	}
	extra.SignIndex = &idx
	jsondata, _ := json.Marshal(args)
	msgContext := string(jsondata)
	keyID, err := dcrm.DoSign(msgHash, msgContext)
	if err != nil {
		return "", err
	}
	log.Info("DcrmSignTransaction start", "keyID", keyID, "msghash", msgHash)
	time.Sleep(waitInterval)

	i := 0
	for ; i < retryCount; i++ {
		signStatus, err := dcrm.GetSignStatus(keyID)
		if err == nil {
			rsv = signStatus.Rsv
			break
		}
		if err == dcrm.ErrGetSignStatusFailed {
			return "", err
		}
		log.Debug("retry get sign status as error", "err", err)
		time.Sleep(retryInterval)
	}
	if i == retryCount || rsv == "" {
		return "", errors.New("get sign status failed")
	}

	log.Trace("DcrmSignTransaction get rsv success", "rsv", rsv)
	return rsv, nil
}

func (b *BtcBridge) SignTransaction(rawTx interface{}, wif string) (signedTx interface{}, err error) {
	authoredTx, ok := rawTx.(*txauthor.AuthoredTx)
	if !ok {
		return nil, tokens.ErrWrongRawTx
	}
	pkwif, err := btcutil.DecodeWIF(wif)
	if err != nil {
		return nil, err
	}
	privateKey := pkwif.PrivKey

	var (
		tx        = authoredTx.Tx
		msgHashes []string
		rsvs      []string
	)

	for i, pkscript := range authoredTx.PrevScripts {
		sigHash, err := txscript.CalcSignatureHash(pkscript, hashType, tx, i)
		if err != nil {
			return nil, err
		}
		msgHash := hex.EncodeToString(sigHash)
		msgHashes = append(msgHashes, msgHash)
	}

	for _, msgHash := range msgHashes {
		signature, err := privateKey.Sign(common.FromHex(msgHash))
		if err != nil {
			return nil, err
		}
		rr := fmt.Sprintf("%064X", signature.R)
		ss := fmt.Sprintf("%064X", signature.S)
		rsv := fmt.Sprintf("%s%s00", rr, ss)
		rsvs = append(rsvs, rsv)
	}

	pk := (*btcec.PublicKey)(&privateKey.PublicKey).SerializeCompressed()
	pubKey := hex.EncodeToString(pk)
	args := &tokens.BuildTxArgs{
		Extra: &tokens.BtcExtraArgs{
			FromPublicKey: &pubKey,
		},
	}
	return b.MakeSignedTransaction(authoredTx, msgHashes, rsvs, args)
}
