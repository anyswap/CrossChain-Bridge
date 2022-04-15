package terra

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/dcrm"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tools/crypto"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
)

// DcrmSignTransaction dcrm sign raw tx
func (b *Bridge) DcrmSignTransaction(rawTx interface{}, args *tokens.BuildTxArgs) (signedTx interface{}, txHash string, err error) {
	txb, ok := rawTx.(*TxBuilder)
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

	signBytes, err := txb.GetSignBytes()
	if err != nil {
		return nil, "", err
	}
	msgHash := fmt.Sprintf("%X", common.Sha256Sum(signBytes))

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

	if len(signature) == crypto.SignatureLength {
		signature = signature[:crypto.SignatureLength-1]
	}

	if len(signature) != crypto.SignatureLength-1 {
		log.Error("wrong length of signature", "length", len(signature))
		return nil, "", errors.New("wrong signature length of keyID " + keyID)
	}

	if !pubKey.VerifySignature(signBytes, signature) {
		log.Error("verify signature failed", "signBytes", common.ToHex(signBytes), "signature", signature)
		return nil, "", errors.New("wrong signature")
	}

	// Construct the SignatureV2 struct
	sigData := signing.SingleSignatureData{
		SignMode:  signMode,
		Signature: signature,
	}
	sig := signing.SignatureV2{
		PubKey:   pubKey,
		Data:     &sigData,
		Sequence: txb.GetSignerData().Sequence,
	}
	err = txb.SetSignatures(sig)
	if err != nil {
		return nil, "", err
	}

	err = txb.ValidateBasic()
	if err != nil {
		return nil, "", err
	}
	return txb.GetSignedTx()
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
func (b *Bridge) SignTransactionWithPrivateKey(rawTx interface{}, privKey *ecdsa.PrivateKey) (signTx interface{}, txHash string, err error) {
	txb, ok := rawTx.(*TxBuilder)
	if !ok {
		return nil, "", errors.New("wrong raw tx param")
	}

	// convert private key
	ecPriv := &secp256k1.PrivKey{Key: privKey.D.Bytes()}

	signBytes, err := txb.GetSignBytes()
	if err != nil {
		return nil, "", err
	}

	signature, err := ecPriv.Sign(signBytes)
	if err != nil {
		return nil, "", err
	}

	if len(signature) != crypto.SignatureLength-1 {
		log.Error("wrong length of signature", "length", len(signature))
		return nil, "", errors.New("wrong signature length")
	}

	pubKey := ecPriv.PubKey()

	if params.IsDebugMode() || params.IsTestMode() {
		pubKeyHex := common.ToHex(pubKey.Bytes())
		pubAddr, _ := PublicKeyToAddress(pubKeyHex)
		log.Info("signer info", "pubkey", pubKeyHex, "signer", pubAddr)
	}

	if !pubKey.VerifySignature(signBytes, signature) {
		log.Error("verify signature failed", "signBytes", common.ToHex(signBytes), "signature", signature)
		return nil, "", errors.New("wrong signature")
	}

	// Construct the SignatureV2 struct
	sigData := signing.SingleSignatureData{
		SignMode:  signMode,
		Signature: signature,
	}
	sig := signing.SignatureV2{
		PubKey:   pubKey,
		Data:     &sigData,
		Sequence: txb.GetSignerData().Sequence,
	}
	err = txb.SetSignatures(sig)
	if err != nil {
		return nil, "", err
	}

	err = txb.ValidateBasic()
	if err != nil {
		return nil, "", err
	}
	return txb.GetSignedTx()
}
