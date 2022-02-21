package ripple

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/dcrm"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	rcrypto "github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/crypto"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/data"
	"github.com/anyswap/CrossChain-Bridge/tools/crypto"
	"github.com/btcsuite/btcd/btcec"
)

const (
	retryGetSignStatusCount    = 70
	retryGetSignStatusInterval = 10 * time.Second
)

func (b *Bridge) verifyTransactionWithArgs(tx data.Transaction, args *tokens.BuildTxArgs) error {

	if tx.GetTransactionType() != data.PAYMENT {
		return fmt.Errorf("Not a payment transaction")
	}

	payment, ok := tx.(*data.Payment)
	if !ok {
		return fmt.Errorf("Type assertion error, transaction is not a payment")
	}

	to := payment.Destination.String()

	checkReceiver := args.Bind
	if !strings.EqualFold(to, checkReceiver) {
		return fmt.Errorf("[sign] verify tx receiver failed")
	}
	return nil
}

// DcrmSignTransaction dcrm sign raw tx
func (b *Bridge) DcrmSignTransaction(rawTx interface{}, args *tokens.BuildTxArgs) (signedTx interface{}, txHash string, err error) {
	log.Debug("Ripple DcrmSignTransaction")

	payment, ok := rawTx.(*data.Payment)
	if !ok {
		return nil, "", fmt.Errorf("Type assertion error, transaction is not a payment")
	}

	err = b.verifyTransactionWithArgs(payment, args)
	if err != nil {
		log.Warn("Verify transaction failed", "error", err)
		return nil, "", err
	}

	jsondata, _ := json.Marshal(args)
	msgContext := string(jsondata)
	msgHash, _, err := data.SigningHash(payment)
	if err != nil {
		return nil, "", fmt.Errorf("Get transaction signing hash failed: %v", err)
	}

	keyID, rsvs, err := dcrm.DoSignOne(b.GetDcrmPublicKey(args.PairID), msgHash.String(), msgContext)
	if err != nil {
		return nil, "", err
	}
	log.Info(b.ChainConfig.BlockChain+" DcrmSignTransaction finished", "keyID", keyID, "msghash", msgHash.String(), "txid", args.SwapID)

	if len(rsvs) != 1 {
		return nil, "", fmt.Errorf("get sign status require one rsv but have %v (keyID = %v)", len(rsvs), keyID)
	}

	rsv := rsvs[0]
	log.Trace(b.ChainConfig.BlockChain+" DcrmSignTransaction get rsv success", "keyID", keyID, "rsv", rsv)

	signature := common.FromHex(rsv)

	if len(signature) != crypto.SignatureLength {
		log.Error("DcrmSignTransaction wrong length of signature")
		return nil, "", errors.New("wrong signature of keyID " + keyID)
	}

	signedTx, err = b.MakeSignedTransaction([]string{rsv}, rawTx)
	if err != nil {
		return signedTx, "", err
	}

	txhash := signedTx.(data.Transaction).GetHash().String()

	return signedTx, txhash, nil
}

// SignTransaction sign tx with pairID
func (b *Bridge) SignTransaction(rawTx interface{}, pairID string) (signedTx interface{}, txHash string, err error) {
	privKey := b.GetTokenConfig(pairID).GetDcrmAddressPrivateKey()
	ecPrikey, err := crypto.HexToECDSA(*privKey)
	if err != nil {
		return nil, "", err
	}
	return b.SignTransactionWithPrivateKey(rawTx, ecPrikey)
}

// SignTransactionWithPrivateKey sign tx with ECDSA private key
func (b *Bridge) SignTransactionWithPrivateKey(rawTx interface{}, privKey *ecdsa.PrivateKey) (signTx interface{}, txHash string, err error) {
	tx, ok := rawTx.(*data.Payment)
	if !ok {
		return nil, "", fmt.Errorf("sign transaction type assertion error")
	}

	hash1, msg, err := data.SigningHash(tx)
	if err != nil {
		return nil, "", err
	}
	log.Info("Prepare to sign", "signing hash", hash1.String(), "blob", fmt.Sprintf("%X", msg))

	hash := fmt.Sprintf("%v", hash1)

	hashBytes, err := hex.DecodeString(hash)
	if err != nil {
		// Unexpected
		return nil, "", fmt.Errorf("tx hash error, %v", err)
	}

	signature, err := (*btcec.PrivateKey)(privKey).Sign(hashBytes)
	if err != nil {
		// Unexpected
		return nil, "", fmt.Errorf("sign with private key error, %v", err)
	}

	rx := fmt.Sprintf("%X", signature.R)
	rx = make64(rx)
	sx := fmt.Sprintf("%X", signature.S)
	sx = make64(sx)
	rsv := rx + sx + "00"

	stx, err := b.MakeSignedTransaction([]string{rsv}, tx)
	if err != nil {
		return nil, "", err
	}
	return stx, "", nil
}

// SignTransactionWithRippleKey sign tx with ripple key
func (b *Bridge) SignTransactionWithRippleKey(rawTx interface{}, key rcrypto.Key, keyseq *uint32) (signTx interface{}, txHash string, err error) {
	tx, ok := rawTx.(*data.Payment)
	if !ok {
		return nil, "", fmt.Errorf("sign transaction type assertion error")
	}

	hash1, msg, err := data.SigningHash(tx)
	if err != nil {
		return nil, "", err
	}
	log.Info("Prepare to sign", "signing hash", hash1.String(), "blob", fmt.Sprintf("%X", msg))

	hash := fmt.Sprintf("%v", hash1)

	hashBytes, err := hex.DecodeString(hash)
	if err != nil {
		// Unexpected
		return nil, "", fmt.Errorf("tx hash error, %v", err)
	}

	sig, err := rcrypto.Sign(key.Private(keyseq), hashBytes, nil)
	if err != nil {
		return nil, "", fmt.Errorf("sign hash error: %v", err)
	}

	signature, err := btcec.ParseSignature(sig, btcec.S256())

	rx := fmt.Sprintf("%X", signature.R)
	rx = make64(rx)
	sx := fmt.Sprintf("%X", signature.S)
	sx = make64(sx)
	rsv := rx + sx + "00"

	stx, err := b.MakeSignedTransaction([]string{rsv}, tx)
	if err != nil {
		return nil, "", err
	}
	return stx, tx.Hash.String(), nil
}

// MakeSignedTransaction make signed transaction
func (b *Bridge) MakeSignedTransaction(rsv []string, transaction interface{}) (signedTransaction interface{}, err error) {
	sig := rsvToSig(rsv[0])
	tx, ok := transaction.(*data.Payment)
	if !ok {
		return nil, fmt.Errorf("type assertion error, transaction is not a payment")
	}
	signedTransaction = makeSignedTx(tx, sig)
	return
}

func makeSignedTx(tx *data.Payment, sig []byte) data.Transaction {
	*tx.GetSignature() = data.VariableLength(sig)
	hash, _, err := data.Raw(tx)
	if err != nil {
		log.Warn("encode ripple tx error", "error", err)
		return tx
	}
	copy(tx.GetHash().Bytes(), hash.Bytes())
	return tx
}

func rsvToSig(rsv string) []byte {
	b, _ := hex.DecodeString(rsv)
	rx := hex.EncodeToString(b[:32])
	sx := hex.EncodeToString(b[32:64])
	r, _ := new(big.Int).SetString(rx, 16)
	s, _ := new(big.Int).SetString(sx, 16)
	signature := &btcec.Signature{
		R: r,
		S: s,
	}
	return signature.Serialize()
}

func make64(str string) string {
	for l := len(str); l < 64; l++ {
		str = "0" + str
	}
	return str
}
