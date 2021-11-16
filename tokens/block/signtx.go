package block

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/dcrm"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tools/crypto"
	"github.com/btcsuite/btcwallet/wallet/txauthor"
)

func (b *Bridge) verifyTransactionWithArgs(tx *txauthor.AuthoredTx, args *tokens.BuildTxArgs) error {
	checkReceiver := args.Bind
	if args.Identifier == tokens.AggregateIdentifier {
		checkReceiver = cfgUtxoAggregateToAddress
	}
	payToReceiverScript, err := b.GetPayToAddrScript(checkReceiver)
	if err != nil {
		return err
	}
	isRightReceiver := false
	for _, out := range tx.Tx.TxOut {
		if bytes.Equal(out.PkScript, payToReceiverScript) {
			isRightReceiver = true
			break
		}
	}
	if !isRightReceiver {
		return fmt.Errorf("[sign] verify tx receiver failed")
	}
	return nil
}

// DcrmSignTransaction dcrm sign raw tx
func (b *Bridge) DcrmSignTransaction(rawTx interface{}, args *tokens.BuildTxArgs) (signedTx interface{}, txHash string, err error) {
	authoredTx, ok := rawTx.(*txauthor.AuthoredTx)
	if !ok {
		return nil, "", tokens.ErrWrongRawTx
	}

	err = b.verifyTransactionWithArgs(authoredTx, args)
	if err != nil {
		return nil, "", err
	}

	cPkData, err := b.GetCompressedPublicKey(cfgFromPublicKey, false)
	if err != nil {
		return nil, "", err
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
		if b.IsPayToScriptHash(preScript) {
			sigScript, err = b.getRedeemScriptByOutputScrpit(preScript)
			if err != nil {
				return nil, "", err
			}
			hasP2shInput = true
		}

		sigHash, err = b.CalcSignatureHash(sigScript, authoredTx.Tx, i)
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

	return b.MakeSignedTransaction(authoredTx, msgHashes, rsvs, sigScripts, cPkData)
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
func (b *Bridge) MakeSignedTransaction(authoredTx *txauthor.AuthoredTx, msgHash, rsv []string, sigScripts [][]byte, cPkData []byte) (signedTx interface{}, txHash string, err error) {
	if len(cPkData) == 0 {
		return nil, "", errors.New("empty public key data")
	}
	err = checkEqualLength(authoredTx, msgHash, rsv, sigScripts)
	if err != nil {
		return nil, "", err
	}
	log.Info(b.ChainConfig.BlockChain+" Bridge MakeSignedTransaction", "msghash", msgHash, "count", len(msgHash))

	for i, txin := range authoredTx.Tx.TxIn {
		signData, ok := b.getSigDataFromRSV(rsv[i])
		if !ok {
			return nil, "", errors.New("wrong RSV data")
		}

		sigScript, err := b.GetSigScript(sigScripts, authoredTx.PrevScripts[i], signData, cPkData, i)
		if err != nil {
			return nil, "", err
		}
		txin.SignatureScript = sigScript
	}
	txHash = authoredTx.Tx.TxHash().String()
	log.Info(b.ChainConfig.BlockChain+" MakeSignedTransaction success", "txhash", txHash)
	return authoredTx, txHash, nil
}

// VerifyRedeemScript verify redeem script
func (b *Bridge) VerifyRedeemScript(prevScript, redeemScript []byte) error {
	p2shScript, err := b.GetP2shSigScript(redeemScript)
	if err != nil {
		return err
	}
	if !bytes.Equal(p2shScript, prevScript) {
		return fmt.Errorf("redeem script %x mismatch", redeemScript)
	}
	return nil
}

func (b *Bridge) getSigDataFromRSV(rsv string) ([]byte, bool) {
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

	return b.SerializeSignature(rr, ss), true
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
	address, err := b.NewAddressPubKeyHash(pkData)
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
	cPkData, err = b.ToCompressedPublicKey(pkData)
	if err != nil {
		return nil, err
	}
	if needVerify {
		err = b.verifyPublickeyData(cPkData)
		if err != nil {
			return nil, err
		}
	}
	return cPkData, nil
}

// the rsv must have correct v (recovery id), otherwise will get wrong public key data.
func (b *Bridge) getPkDataFromSig(rsv, msgHash string, compressed bool) (pkData []byte, err error) {
	rsvData := common.FromHex(rsv)
	hashData := common.FromHex(msgHash)
	ecPub, err := crypto.SigToPub(hashData, rsvData)
	if err != nil {
		return nil, err
	}
	return b.SerializePublicKey(ecPub, compressed), nil
}

// DcrmSignMsgHash dcrm sign msg hash
func (b *Bridge) DcrmSignMsgHash(msgHash []string, args *tokens.BuildTxArgs) (rsv []string, err error) {
	extra := args.Extra.BtcExtra
	if extra == nil {
		return nil, tokens.ErrWrongExtraArgs
	}
	jsondata, _ := json.Marshal(args)
	msgContext := []string{string(jsondata)}

	log.Info(b.ChainConfig.BlockChain+" DcrmSignTransaction start", "msgContext", msgContext, "txid", args.SwapID)
	keyID, rsv, err := dcrm.DoSign(cfgFromPublicKey, msgHash, msgContext)
	if err != nil {
		return nil, err
	}
	log.Info(b.ChainConfig.BlockChain+" DcrmSignTransaction finished", "keyID", keyID, "msghash", msgHash, "txid", args.SwapID)

	if len(rsv) != len(msgHash) {
		return nil, fmt.Errorf("get sign status require %v rsv but have %v (keyID = %v)", len(msgHash), len(rsv), keyID)
	}

	rsv, err = b.adjustRsvOrders(rsv, msgHash, cfgFromPublicKey)
	if err != nil {
		return nil, err
	}

	log.Trace(b.ChainConfig.BlockChain+" DcrmSignTransaction get rsv success", "keyID", keyID, "txid", args.SwapID, "rsv", rsv)
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

// SignTransaction sign tx with pairID
func (b *Bridge) SignTransaction(rawTx interface{}, pairID string) (signedTx interface{}, txHash string, err error) {
	privKey := b.GetTokenConfig(pairID).GetDcrmAddressPrivateKey()
	return b.SignTransactionWithPrivateKey(rawTx, privKey)
}

// SignTransactionWithWIF sign tx with WIF
func (b *Bridge) SignTransactionWithWIF(rawTx interface{}, wif string) (signedTx interface{}, txHash string, err error) {
	pkwif, err := DecodeWIF(wif)
	if err != nil {
		return nil, "", err
	}
	return b.SignTransactionWithPrivateKey(rawTx, pkwif.PrivKey.ToECDSA())
}

// SignTransactionWithPrivateKey sign tx with ECDSA private key
func (b *Bridge) SignTransactionWithPrivateKey(rawTx interface{}, privKey *ecdsa.PrivateKey) (signTx interface{}, txHash string, err error) {
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
		if b.IsPayToScriptHash(preScript) {
			sigScript, err = b.getRedeemScriptByOutputScrpit(preScript)
			if err != nil {
				return nil, "", err
			}
			hasP2shInput = true
		}

		sigHash, err := b.CalcSignatureHash(sigScript, authoredTx.Tx, i)
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
		rsv, errf := b.SignWithECDSA(privKey, common.FromHex(msgHash))
		if errf != nil {
			return nil, "", errf
		}
		rsvs = append(rsvs, rsv)
	}

	cPkData := b.GetPublicKeyFromECDSA(privKey, true)
	return b.MakeSignedTransaction(authoredTx, msgHashes, rsvs, sigScripts, cPkData)
}
