package btc

import (
	"bytes"
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

const (
	retryGetSignStatusCount    = 70
	retryGetSignStatusInterval = 10 * time.Second

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

	return b.MakeSignedTransaction(authoredTx, msgHashes, rsvs, sigScripts, args.PairID)
}

func checkEqualLength(authoredTx *txauthor.AuthoredTx, msgHash, rsv []string, sigScripts [][]byte) error {
	txIn := authoredTx.Tx.TxIn
	if len(txIn) != len(msgHash) {
		return errors.New("mismatch number of msghashes and tx inputs")
	}
	if len(txIn) != len(rsv) {
		return errors.New("mismatch number of signatures and tx inputs")
	}
	if sigScripts != nil && len(sigScripts) != len(txIn) {
		return errors.New("mismatch number of signatures scripts and tx inputs")
	}
	return nil
}

// MakeSignedTransaction make signed tx
func (b *Bridge) MakeSignedTransaction(authoredTx *txauthor.AuthoredTx, msgHash, rsv []string, sigScripts [][]byte, pairID string) (signedTx interface{}, txHash string, err error) {
	err = checkEqualLength(authoredTx, msgHash, rsv, sigScripts)
	if err != nil {
		return nil, "", err
	}
	log.Info(b.ChainConfig.BlockChain+" Bridge MakeSignedTransaction", "pairID", pairID, "msghash", msgHash, "count", len(msgHash))

	var sigScript, cPkData []byte

	if pairID != "" && tokens.BtcFromPublicKey != "" {
		cPkData, err = b.GetCompressedPublicKey(tokens.BtcFromPublicKey, false)
		if err != nil {
			return nil, "", err
		}
	}

	for i, txin := range authoredTx.Tx.TxIn {
		signData, ok := getSigDataFromRSV(rsv[i])
		if !ok {
			return nil, "", errors.New("wrong RSV data")
		}

		if len(cPkData) == 0 {
			cPkData, err = b.getPkDataFromSig(rsv[i], msgHash[i], true)
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
				redeemScript := sigScripts[i]
				err = b.verifyRedeemScript(prevScript, redeemScript)
				if err == nil {
					sigScript, err = txscript.NewScriptBuilder().AddData(signData).AddData(cPkData).AddData(redeemScript).Script()
				}
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
	log.Info(b.ChainConfig.BlockChain+" MakeSignedTransaction success", "txhash", txHash)
	return authoredTx, txHash, nil
}

func (b *Bridge) verifyRedeemScript(prevScript, redeemScript []byte) error {
	p2shScript, err := b.GetP2shSigScript(redeemScript)
	if err != nil {
		return err
	}
	if !bytes.Equal(p2shScript, prevScript) {
		return fmt.Errorf("redeem script %x mismatch", redeemScript)
	}
	return nil
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

func (b *Bridge) verifyPublickeyData(pkData []byte) error {
	tokenCfg := b.GetTokenConfig(PairID)
	if tokenCfg == nil {
		return nil
	}
	dcrmAddress := tokenCfg.DcrmAddress
	if dcrmAddress == "" {
		return nil
	}
	address, err := btcutil.NewAddressPubKeyHash(btcutil.Hash160(pkData), b.GetChainParams())
	if err != nil {
		return err
	}
	if address.EncodeAddress() != dcrmAddress {
		return fmt.Errorf("public key address %v is not the configed dcrm address %v", address, dcrmAddress)
	}
	return nil
}

// GetCompressedPublicKey get compressed public key
func (b *Bridge) GetCompressedPublicKey(fromPublicKey string, needVerify bool) (cPkData []byte, err error) {
	if fromPublicKey == "" {
		return nil, nil
	}
	pkData := common.FromHex(fromPublicKey)
	pubKey, err := btcec.ParsePubKey(pkData, btcec.S256())
	if err != nil {
		return nil, err
	}
	cPkData = pubKey.SerializeCompressed()
	if needVerify {
		err = b.verifyPublickeyData(cPkData)
		if err != nil {
			return nil, err
		}
	}
	return cPkData, nil
}

func (b *Bridge) getPkDataFromSig(rsv, msgHash string, compressed bool) (pkData []byte, err error) {
	rsvData := common.FromHex(rsv)
	hashData := common.FromHex(msgHash)
	pub, err := crypto.SigToPub(hashData, rsvData)
	if err != nil {
		return nil, err
	}
	if compressed {
		pkData = (*btcec.PublicKey)(pub).SerializeCompressed()
	} else {
		pkData = (*btcec.PublicKey)(pub).SerializeUncompressed()
	}
	return pkData, nil
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
	log.Info(b.ChainConfig.BlockChain+" DcrmSignTransaction start", "keyID", keyID, "msghash", msgHash, "txid", args.SwapID)
	time.Sleep(retryGetSignStatusInterval)

	var signStatus *dcrm.SignStatus
	i := 0
	for ; i < retryGetSignStatusCount; i++ {
		signStatus, err = dcrm.GetSignStatus(keyID)
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
		log.Warn("retry get sign status as error", "err", err, "txid", args.SwapID, "keyID", keyID, "bridge", args.Identifier, "swaptype", args.SwapType.String())
		time.Sleep(retryGetSignStatusInterval)
	}
	if i == retryGetSignStatusCount || len(rsv) == 0 {
		return nil, errors.New("get sign status failed")
	}

	rsv, err = b.adjustRsvOrders(rsv, msgHash, tokens.BtcFromPublicKey)
	if err != nil {
		return nil, err
	}

	log.Trace(b.ChainConfig.BlockChain+" DcrmSignTransaction get rsv success", "keyID", keyID, "rsv", rsv)
	return rsv, nil
}

func (b *Bridge) adjustRsvOrders(rsvs, msgHashes []string, fromPublicKey string) (newRsvs []string, err error) {
	if len(rsvs) <= 1 {
		return rsvs, nil
	}
	fromPubkeyData, err := b.GetCompressedPublicKey(fromPublicKey, false)
	matchedRsvMap := make(map[string]struct{})
	var cPkData []byte
	for _, msgHash := range msgHashes {
		matched := false
		for _, rsv := range rsvs {
			if _, exist := matchedRsvMap[rsv]; exist {
				continue
			}
			cPkData, err = b.getPkDataFromSig(rsv, msgHash, true)
			if err == nil && bytes.Equal(cPkData, fromPubkeyData) {
				matchedRsvMap[rsv] = struct{}{}
				newRsvs = append(newRsvs, rsv)
				matched = true
				break
			}
		}
		if !matched {
			return nil, fmt.Errorf("msgHash %v hash no matched rsv", msgHash)
		}
	}
	return newRsvs, err
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

	return b.MakeSignedTransaction(authoredTx, msgHashes, rsvs, sigScripts, "")
}
