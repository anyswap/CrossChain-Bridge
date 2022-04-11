package terra

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/dcrm"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tools/crypto"
	"github.com/btcsuite/btcd/btcec"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/tendermint/tendermint/crypto/tmhash"
)

// DcrmSignTransaction dcrm sign raw tx
func (b *Bridge) DcrmSignTransaction(rawTx interface{}, args *tokens.BuildTxArgs) (signedTx interface{}, txHash string, err error) {
	txw, ok := rawTx.(*wrapper)
	if !ok {
		return nil, "", errors.New("wrong raw tx param")
	}

	pairID := args.PairID
	tokenCfg := b.GetTokenConfig(pairID)
	if tokenCfg == nil {
		return nil, "", fmt.Errorf("swap pair '%v' is not configed", pairID)
	}

	pubKey, err := PubKeyFromStr(tokenCfg.DcrmPubkey)
	if err != nil {
		return nil, "", err
	}

	signBytes, err := txw.GetSignBytes()
	if err != nil {
		return nil, "", err
	}
	msgHash := fmt.Sprintf("%X", tmhash.Sum(signBytes))

	jsondata, _ := json.Marshal(args.GetExtraArgs())
	msgContext := string(jsondata)

	log.Info(b.ChainConfig.BlockChain+" DcrmSignTransaction start", "msghash", msgHash, "txid", args.SwapID)
	keyID, rsvs, err := dcrm.DoSignOne(tokenCfg.DcrmPubkey, msgHash, msgContext)
	if err != nil {
		return nil, "", err
	}
	log.Info(b.ChainConfig.BlockChain+" DcrmSignTransaction finished", "keyID", keyID, "msghash", msgHash, "txid", args.SwapID)

	if len(rsvs) != 1 {
		return nil, "", fmt.Errorf("get sign status require one rsv but have %v (keyID = %v)", len(rsvs), keyID)
	}

	rsv := rsvs[0]
	log.Trace(b.ChainConfig.BlockChain+" DcrmSignTransaction get rsv success", "keyID", keyID, "txid", args.SwapID, "rsv", rsv)
	signature := common.FromHex(rsv)
	if len(signature) != crypto.SignatureLength {
		log.Error("DcrmSignTransaction wrong length of signature")
		return nil, "", errors.New("wrong signature of keyID " + keyID)
	}

	// Construct the SignatureV2 struct
	sigData := signing.SingleSignatureData{
		SignMode:  signMode,
		Signature: signature,
	}
	sig := signing.SignatureV2{
		PubKey:   pubKey,
		Data:     &sigData,
		Sequence: txw.GetSignerData().Sequence,
	}
	txw.SetSignatures(sig)

	txBytes, err := txw.EncodeTx()
	if err != nil {
		return nil, "", err
	}

	txHash = fmt.Sprintf("%X", tmhash.Sum(txBytes))

	return txBytes, txHash, err
}

// SignTransaction sign tx with pairID
func (b *Bridge) SignTransaction(rawTx interface{}, pairID string) (signTx interface{}, txHash string, err error) {
	privKey := b.GetTokenConfig(pairID).GetDcrmAddressPrivateKey()
	ecPrikey, err := crypto.HexToECDSA(*privKey)
	if err != nil {
		return nil, "", err
	}
	return b.SignTransactionWithPrivateKey(rawTx, ecPrikey)
}

// SignTransactionWithPrivateKey sign tx with ECDSA private key
func (b *Bridge) SignTransactionWithPrivateKey(rawTx interface{}, privKey *ecdsa.PrivateKey) (signedTx interface{}, txHash string, err error) {
	txw, ok := rawTx.(*wrapper)
	if !ok {
		return nil, "", errors.New("wrong raw tx param")
	}

	signBytes, err := txw.GetSignBytes()
	if err != nil {
		return nil, "", err
	}
	msgHash := fmt.Sprintf("%X", tmhash.Sum(signBytes))

	ecPriv, ecPub := btcec.PrivKeyFromBytes(btcec.S256(), privKey.D.Bytes())
	if err != nil {
		return nil, "", err
	}
	pubKey, err := PubKeyFromBytes(ecPub.SerializeCompressed())
	if err != nil {
		return nil, "", err
	}

	// Sign those bytes
	signature, err := ecPriv.Sign(common.FromHex(msgHash))
	if err != nil {
		return nil, "", err
	}

	// Construct the SignatureV2 struct
	sigData := signing.SingleSignatureData{
		SignMode:  signMode,
		Signature: signature.Serialize(),
	}
	sig := signing.SignatureV2{
		PubKey:   pubKey,
		Data:     &sigData,
		Sequence: txw.GetSignerData().Sequence,
	}
	txw.SetSignatures(sig)

	txBytes, err := txw.EncodeTx()
	if err != nil {
		return nil, "", err
	}

	txHash = fmt.Sprintf("%X", tmhash.Sum(txBytes))

	return txBytes, txHash, err
}
