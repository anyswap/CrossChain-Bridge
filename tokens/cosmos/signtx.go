package cosmos

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/dcrm"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tools/crypto"

	"github.com/btcsuite/btcd/btcec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

const (
	retryGetSignStatusInterval = 10 * time.Second
)

func (b *Bridge) verifyTransactionWithArgs(tx StdSignContent, args *tokens.BuildTxArgs) error {
	tokenCfg := b.GetTokenConfig(args.PairID)
	if len(tx.Msgs) != 1 {
		return errors.New("wrong msgs length")
	}
	msg, ok := tx.Msgs[0].(MsgSend)
	if !ok {
		return errors.New("msg types error")
	}
	switch {
	case !strings.EqualFold(args.From, msg.FromAddress.String()):
		return fmt.Errorf("[cosmos verify transaction with args] From address not match, args.From: %v, msg.FromAddress: %v", args.From, msg.FromAddress.String())
	case !strings.EqualFold(msg.FromAddress.String(), tokenCfg.DcrmAddress):
		return fmt.Errorf("[cosmos verify transaction with args] From address is not dcrm address, args.From: %v, dcrm address: %v", msg.FromAddress.String(), tokenCfg.DcrmAddress)
	case !b.IsValidAddress(args.Bind):
		return fmt.Errorf("[cosmos verify transaction with args] Invalid to address: %v", args.Bind)
	case !strings.EqualFold(args.Bind, msg.ToAddress.String()):
		return fmt.Errorf("[cosmos verify transaction with args] To address not match, args.To: %v, msg.ToAddress: %v", args.Bind, msg.ToAddress.String())
	default:
	}
	if len(msg.Amount) != 1 {
		return errors.New("wrong amount length")
	}
	amount := msg.Amount[0]
	checkPairID, err := b.getPairID(amount)
	if err != nil || checkPairID != args.PairID {
		return fmt.Errorf("[cosmos verify transaction with args] Token type not match, %v, %v", checkPairID, args.PairID)
	}
	if amount.Amount.BigInt().Cmp(args.OriginValue) >= 0 {
		return fmt.Errorf("[cosmos verify transaction with args] Amount not match, %v, %v", amount.Amount.BigInt(), args.OriginValue)
	}
	return nil
}

// DcrmSignTransaction dcrm sign raw tx
func (b *Bridge) DcrmSignTransaction(rawTx interface{}, args *tokens.BuildTxArgs) (signedTx interface{}, txHash string, err error) {
	tx, ok := rawTx.(StdSignContent)
	if !ok {
		return nil, "", errors.New("wrong raw tx param")
	}
	err = b.verifyTransactionWithArgs(tx, args)
	if err != nil {
		return nil, "", err
	}
	msgHash := tx.Hash()
	jsondata, _ := json.Marshal(args)
	msgContext := string(jsondata)
	keyID, rsv, err := dcrm.DoSignOne(b.GetDcrmPublicKey(args.PairID), msgHash, msgContext)
	if err != nil {
		return nil, "", err
	}
	log.Info(b.ChainConfig.BlockChain+" DcrmSignTransaction start", "keyID", keyID, "msghash", msgHash, "txid", args.SwapID)
	time.Sleep(retryGetSignStatusInterval)

	log.Trace(b.ChainConfig.BlockChain+" DcrmSignTransaction get rsv success", "keyID", keyID, "rsv", rsv)

	signature := common.FromHex(rsv[0])

	if len(signature) != crypto.SignatureLength {
		log.Error("DcrmSignTransaction wrong length of signature")
		return nil, "", errors.New("wrong signature of keyID " + keyID)
	}

	// pub
	pubHex := b.GetDcrmPublicKey(args.PairID)
	pubBytes, err := hex.DecodeString(pubHex)
	if err != nil {
		return nil, "", errors.New("wrong dcrm public key")
	}

	pub, err := btcec.ParsePubKey(pubBytes, btcec.S256())
	if err != nil {
		return nil, "", errors.New("wrong dcrm public key")
	}

	cpub := pub.SerializeCompressed()
	var arr [33]byte
	copy(arr[:], cpub[:33])
	pubkey := secp256k1.PubKeySecp256k1(arr)

	rsvb, err := hex.DecodeString(rsv[0])
	if err != nil {
		return
	}
	var signatureBytes []byte
	if len(rsvb) == 65 {
		signatureBytes = rsvb[:64]
	}
	stdsig := authtypes.StdSignature{
		PubKey:    pubkey,
		Signature: signatureBytes,
	}
	signedTx = HashableStdTx{
		StdSignContent: tx,
		Signatures:     []authtypes.StdSignature{stdsig},
	}

	if !pubkey.VerifyBytes(tx.SignBytes(), signatureBytes) {
		log.Error("Dcrm sign verify error")
		return nil, "", errors.New("wrong signature")
	}

	txHash = msgHash
	log.Info(b.ChainConfig.BlockChain+" DcrmSignTransaction success", "keyID", keyID, "txhash", txHash, "nonce", signedTx.(HashableStdTx).Sequence)
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
	tx, ok := rawTx.(StdSignContent)
	if !ok {
		return nil, "", errors.New("wrong raw tx param")
	}

	signBytes := tx.SignBytes()

	var privBytes [32]byte
	btcecpriv := btcec.PrivateKey(*privKey)
	copy(privBytes[:], btcecpriv.Serialize()[:33])
	priv := secp256k1.PrivKeySecp256k1(privBytes)
	signature, err := priv.Sign(signBytes)
	if err != nil {
		return nil, "", err
	}

	pub := priv.PubKey()

	stdsig := authtypes.StdSignature{
		PubKey:    pub,
		Signature: signature,
	}

	signedTx = HashableStdTx{
		StdSignContent: tx,
		Signatures:     []authtypes.StdSignature{stdsig},
	}

	txHash = signedTx.(HashableStdTx).Hash()

	return
}
