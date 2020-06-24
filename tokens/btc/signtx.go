package btc

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/dcrm"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tools/crypto"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/wallet/txauthor"
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
		sigHash      []byte
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

		sigHash, err = txscript.CalcSignatureHash(sigScript, hashType, authoredTx.Tx, i)
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

	rsvs, err = b.DcrmSignMsgHash(msgHashes, args)
	if err != nil {
		return nil, "", err
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
func (b *Bridge) MakeSignedTransaction(authoredTx *txauthor.AuthoredTx, msgHash, rsv []string, sigScripts [][]byte, args *tokens.BuildTxArgs) (signedTx interface{}, txHash string, err error) {
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
	log.Info(b.TokenConfig.BlockChain+" Bridge MakeSignedTransaction", "msghash", msgHash, "count", len(msgHash))

	cPkData, err := b.getPkDataFronConfig(args)
	if err != nil {
		return nil, "", err
	}

	var sigScript []byte
	for i, txin := range txIn {
		signData, ok := getSigDataFromRSV(rsv[i])
		if !ok {
			return nil, "", errors.New("wrong RSV data")
		}

		if len(cPkData) == 0 {
			cPkData, err = b.getPkDataFronSig(rsv[i], msgHash[i], args.SwapType)
			if err != nil {
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
				err = fmt.Errorf("call MakeSignedTransaction spend p2sh without redeem scripts")
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
	log.Info(b.TokenConfig.BlockChain+" MakeSignedTransaction success", "txhash", txHash)
	return authoredTx, txHash, nil
}

func getSigDataFromRSV(rsv string) ([]byte, bool) {
	rs := rsv[0 : len(rsv)-2]

	r := rs[:64]
	s := rs[64:]

	rr, ok := new(big.Int).SetString(r, 16)
	if !ok {
		return nil, false
	}

	ss, ok := new(big.Int).SetString(s, 16)
	if !ok {
		return nil, false
	}

	sign := &btcec.Signature{
		R: rr,
		S: ss,
	}

	signData := append(sign.Serialize(), byte(hashType))
	return signData, true
}

func (b *Bridge) getPkDataFronConfig(args *tokens.BuildTxArgs) (cPkData []byte, err error) {
	fromPublicKey := tokens.BtcFromPublicKey
	if args != nil && args.Extra != nil {
		extra := args.Extra.BtcExtra
		if extra != nil && extra.FromPublicKey != nil {
			fromPublicKey = *extra.FromPublicKey
		}
	}
	if fromPublicKey == "" {
		return nil, nil
	}
	cPkData = common.FromHex(fromPublicKey)
	err = b.verifyPublickeyData(cPkData, args.SwapType)
	if err != nil {
		return nil, err
	}
	return cPkData, nil
}

func (b *Bridge) getPkDataFronSig(rsv, msgHash string, swapType tokens.SwapType) (cPkData []byte, err error) {
	rsvData := common.FromHex(rsv)
	hashData := common.FromHex(msgHash)
	pkData, err := crypto.Ecrecover(hashData, rsvData)
	if err != nil {
		return nil, err
	}
	pk, err := btcec.ParsePubKey(pkData, btcec.S256())
	if err != nil {
		return nil, err
	}
	cPkData = pk.SerializeCompressed()
	err = b.verifyPublickeyData(cPkData, swapType)
	if err != nil {
		return nil, err
	}
	return cPkData, nil
}

// DcrmSignMsgHash dcrm sign msg hash
func (b *Bridge) DcrmSignMsgHash(msgHash []string, args *tokens.BuildTxArgs) (rsv []string, err error) {
	extra := args.Extra.BtcExtra
	if extra == nil {
		return nil, tokens.ErrWrongExtraArgs
	}
	jsondata, _ := json.Marshal(args)
	msgContext := []string{string(jsondata)}
	keyID, err := dcrm.DoSign(msgHash, msgContext)
	if err != nil {
		return nil, err
	}
	log.Info(b.TokenConfig.BlockChain+" DcrmSignTransaction start", "keyID", keyID, "msghash", msgHash, "txid", args.SwapID)
	time.Sleep(waitInterval)

	i := 0
	for ; i < retryCount; i++ {
		signStatus, err := dcrm.GetSignStatus(keyID)
		if err == nil {
			if len(signStatus.Rsv) != len(msgHash) {
				return nil, fmt.Errorf("get sign status require %v rsv but have %v (keyID = %v)", len(msgHash), len(signStatus.Rsv), keyID)
			}
			rsv = signStatus.Rsv
			break
		}
		switch err {
		case dcrm.ErrGetSignStatusFailed, dcrm.ErrGetSignStatusTimeout:
			return nil, err
		}
		log.Warn("retry get sign status as error", "err", err, "txid", args.SwapID)
		time.Sleep(retryInterval)
	}
	if i == retryCount || len(rsv) == 0 {
		return nil, errors.New("get sign status failed")
	}

	log.Trace(b.TokenConfig.BlockChain+" DcrmSignTransaction get rsv success", "keyID", keyID, "rsv", rsv)
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
