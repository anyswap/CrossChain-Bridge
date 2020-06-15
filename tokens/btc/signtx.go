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

// DcrmSignTransaction dcrm sign raw tx
func (b *Bridge) DcrmSignTransaction(rawTx interface{}, args *tokens.BuildTxArgs) (signedTx interface{}, txHash string, err error) {
	authoredTx, ok := rawTx.(*txauthor.AuthoredTx)
	if !ok {
		return nil, "", tokens.ErrWrongRawTx
	}

	var (
		msgHashes    []string
		rsvs         []string
		sigScripts   [][]byte
		hasP2shInput bool
	)

	for i, preScript := range authoredTx.PrevScripts {
		sigScript := preScript
		if txscript.IsPayToScriptHash(preScript) {
			sigScript, err = b.getRedeemScriptByOutputScrpit(preScript)
			if err != nil {
				return nil, "", err
			}
			hasP2shInput = true
		}

		sigHash, err := txscript.CalcSignatureHash(sigScript, hashType, authoredTx.Tx, i)
		if err != nil {
			return nil, "", err
		}
		msgHash := hex.EncodeToString(sigHash)
		msgHashes = append(msgHashes, msgHash)
		sigScripts = append(sigScripts, sigScript)
	}
	if !hasP2shInput {
		sigScripts = nil
	}

	for idx, msgHash := range msgHashes {
		rsv, err := b.DcrmSignMsgHash(msgHash, args, idx)
		if err != nil {
			return nil, "", err
		}
		rsvs = append(rsvs, rsv)
	}

	return b.MakeSignedTransaction(authoredTx, msgHashes, rsvs, sigScripts, args)
}

func (b *Bridge) verifyPublickeyData(pkData []byte, swapType tokens.SwapType) error {
	switch swapType {
	case tokens.SwapinType:
		return tokens.ErrSwapTypeNotSupported
	case tokens.SwapoutType, tokens.SwapRecallType:
		dcrmAddress := b.TokenConfig.DcrmAddress
		address, _ := btcutil.NewAddressPubKeyHash(btcutil.Hash160(pkData), b.GetChainConfig())
		if address.EncodeAddress() != b.TokenConfig.DcrmAddress {
			return fmt.Errorf("sign public key %v is not the configed dcrm address %v", address, dcrmAddress)
		}
	}
	return nil
}

// MakeSignedTransaction make signed tx
func (b *Bridge) MakeSignedTransaction(authoredTx *txauthor.AuthoredTx, msgHash []string, rsv []string,
	sigScripts [][]byte, args *tokens.BuildTxArgs) (signedTx interface{}, txHash string, err error) {

	txIn := authoredTx.Tx.TxIn
	if len(txIn) != len(msgHash) {
		return nil, "", errors.New("mismatch number of msghashes and tx inputs")
	}
	if len(txIn) != len(rsv) {
		return nil, "", errors.New("mismatch number of signatures and tx inputs")
	}
	if sigScripts != nil && len(sigScripts) != len(txIn) {
		return nil, "", errors.New("mismatch number of signatures scripts and tx inputs")
	}
	log.Info("Bridge MakeSignedTransaction", "msghash", msgHash, "count", len(msgHash))

	var (
		cPkData   []byte
		sigScript []byte
	)

	fromPublicKey := tokens.BtcFromPublicKey
	if args != nil && args.Extra != nil {
		extra := args.Extra.BtcExtra
		if extra != nil && extra.FromPublicKey != nil {
			fromPublicKey = *extra.FromPublicKey
		}
	}
	if fromPublicKey != "" {
		cPkData = common.FromHex(fromPublicKey)
		if err := b.verifyPublickeyData(cPkData, args.SwapType); err != nil {
			return nil, "", err
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
				return nil, "", err
			}
			pk, _ := btcec.ParsePubKey(pkData, btcec.S256())
			cPkData = pk.SerializeCompressed()
			if err := b.verifyPublickeyData(cPkData, args.SwapType); err != nil {
				return nil, "", err
			}
		}

		prevScript := authoredTx.PrevScripts[i]
		scriptClass := txscript.GetScriptClass(prevScript)
		switch scriptClass {
		case txscript.PubKeyHashTy:
			sigScript, err = txscript.NewScriptBuilder().AddData(signData).AddData(cPkData).Script()
		case txscript.ScriptHashTy:
			if sigScripts == nil {
				err = fmt.Errorf("MakeSignedTransaction spend p2sh without redeem scripts")
			} else {
				sigScript, err = txscript.NewScriptBuilder().AddData(signData).AddData(cPkData).AddData(sigScripts[i]).Script()
			}
		default:
			err = fmt.Errorf("unsupport to spend '%v' output", scriptClass.String())
		}
		if err != nil {
			return nil, "", err
		}
		txin.SignatureScript = sigScript
	}
	txHash = authoredTx.Tx.TxHash().String()
	return authoredTx, txHash, nil
}

// DcrmSignMsgHash dcrm sign msg hash
func (b *Bridge) DcrmSignMsgHash(msgHash string, args *tokens.BuildTxArgs, idx int) (rsv string, err error) {
	extra := args.Extra.BtcExtra
	if extra == nil {
		return "", tokens.ErrWrongExtraArgs
	}
	extra.SignIndex = &idx
	jsondata, _ := json.Marshal(args)
	msgContext := string(jsondata)
	keyID, err := dcrm.DoSign(msgHash, msgContext)
	if err != nil {
		return "", err
	}
	log.Info("DcrmSignTransaction start", "keyID", keyID, "msghash", msgHash, "txid", args.SwapID)
	time.Sleep(waitInterval)

	i := 0
	for ; i < retryCount; i++ {
		signStatus, err := dcrm.GetSignStatus(keyID)
		if err == nil {
			rsv = signStatus.Rsv
			break
		}
		switch err {
		case dcrm.ErrGetSignStatusFailed, dcrm.ErrGetSignStatusTimeout:
			return "", err
		}
		log.Debug("retry get sign status as error", "err", err, "txid", args.SwapID)
		time.Sleep(retryInterval)
	}
	if i == retryCount || rsv == "" {
		return "", errors.New("get sign status failed")
	}

	log.Trace("DcrmSignTransaction get rsv success", "rsv", rsv)
	return rsv, nil
}

// SignTransaction sign tx with wif
func (b *Bridge) SignTransaction(rawTx interface{}, wif string) (signedTx interface{}, txHash string, err error) {
	authoredTx, ok := rawTx.(*txauthor.AuthoredTx)
	if !ok {
		return nil, "", tokens.ErrWrongRawTx
	}
	pkwif, err := btcutil.DecodeWIF(wif)
	if err != nil {
		return nil, "", err
	}
	privateKey := pkwif.PrivKey

	var (
		msgHashes    []string
		rsvs         []string
		sigScripts   [][]byte
		hasP2shInput bool
	)

	for i, preScript := range authoredTx.PrevScripts {
		sigScript := preScript
		if txscript.IsPayToScriptHash(preScript) {
			sigScript, err = b.getRedeemScriptByOutputScrpit(preScript)
			if err != nil {
				return nil, "", err
			}
			hasP2shInput = true
		}

		sigHash, err := txscript.CalcSignatureHash(sigScript, hashType, authoredTx.Tx, i)
		if err != nil {
			return nil, "", err
		}
		msgHash := hex.EncodeToString(sigHash)
		msgHashes = append(msgHashes, msgHash)
		sigScripts = append(sigScripts, sigScript)
	}
	if !hasP2shInput {
		sigScripts = nil
	}

	for _, msgHash := range msgHashes {
		signature, err := privateKey.Sign(common.FromHex(msgHash))
		if err != nil {
			return nil, "", err
		}
		rr := fmt.Sprintf("%064X", signature.R)
		ss := fmt.Sprintf("%064X", signature.S)
		rsv := fmt.Sprintf("%s%s00", rr, ss)
		rsvs = append(rsvs, rsv)
	}

	pk := (*btcec.PublicKey)(&privateKey.PublicKey).SerializeCompressed()
	pubKey := hex.EncodeToString(pk)
	args := &tokens.BuildTxArgs{
		Extra: &tokens.AllExtras{
			BtcExtra: &tokens.BtcExtraArgs{
				FromPublicKey: &pubKey,
			},
		},
	}
	return b.MakeSignedTransaction(authoredTx, msgHashes, rsvs, sigScripts, args)
}
