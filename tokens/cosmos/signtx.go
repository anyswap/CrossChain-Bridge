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
	retryGetSignStatusCount    = 70
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
	if strings.EqualFold(args.From, msg.FromAddress.String()) == false || strings.EqualFold(msg.FromAddress.String(), tokenCfg.DcrmAddress) == false {
		return errors.New("wrong from address")
	}
	if strings.EqualFold(args.To, msg.ToAddress.String()) == false || b.IsValidAddress(args.To) == false {
		return errors.New("wrong to address")
	}
	if len(msg.Amount) != 1 {
		return errors.New("wrong amount length")
	}
	amount := msg.Amount[0]
	checkPairID, err := b.getPairID(amount)
	if err != nil || checkPairID != args.PairID {
		return errors.New("wrong coin type")
	}
	if amount.Amount.BigInt().Cmp(args.Value) != 0 {
		return errors.New("wrong amount")
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
	rpcAddr, keyID, err := dcrm.DoSignOne(b.GetDcrmPublicKey(args.PairID), msgHash, msgContext)
	if err != nil {
		return nil, "", err
	}
	log.Info(b.ChainConfig.BlockChain+" DcrmSignTransaction start", "keyID", keyID, "msghash", msgHash, "txid", args.SwapID)
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

	rsvb, err := hex.DecodeString(rsv)
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

	if pubkey.VerifyBytes(tx.SignBytes(), signatureBytes) == false {
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
