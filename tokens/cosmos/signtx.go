package cosmos

import (
	"crypto/ecdsa"
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
	"github.com/anyswap/CrossChain-Bridge/types"

	"github.com/btcsuite/btcd/btcec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	"github.com/tendermint/tendermint/crypto/tmhash"
)

const (
	retryGetSignStatusCount    = 70
	retryGetSignStatusInterval = 10 * time.Second
)

func (b *Bridge) verifyTransactionWithArgs(tx StdSignContent, args *tokens.BuildTxArgs) error {
	if tx.To() == nil || *tx.To() == (common.Address{}) {
		return fmt.Errorf("[sign] verify tx receiver failed")
	}
	tokenCfg := b.GetTokenConfig(args.PairID)
	if tokenCfg == nil {
		return fmt.Errorf("[sign] verify tx with unknown pairID '%v'", args.PairID)
	}
	checkReceiver := tokenCfg.ContractAddress
	if args.SwapType == tokens.SwapoutType && !tokenCfg.IsErc20() {
		checkReceiver = args.Bind
	}
	if !strings.EqualFold(tx.To().String(), checkReceiver) {
		return fmt.Errorf("[sign] verify tx receiver failed")
	}
	return nil
}

// DcrmSignTransaction dcrm sign raw tx
func (b *Bridge) DcrmSignTransaction(rawTx interface{}, args *tokens.BuildTxArgs) (signTx interface{}, txHash string, err error) {
	tx, ok := rawTx.(StdSignContent)
	if !ok {
		return nil, "", errors.New("wrong raw tx param")
	}
	err = b.verifyTransactionWithArgs(tx, args)
	if err != nil {
		return nil, "", err
	}
	signBytes := authtypes.StdSignBytes(tx.ChainID, tx.AccountNumber, tx.Sequence, fee, msgs, tx.Memo)
	msgHash := fmt.Sprintf("%X", tmhash.Sum(signBytes))
	jsondata, _ := json.Marshal(args)
	msgContext := string(jsondata)
	rpcAddr, keyID, err := dcrm.DoSignOne(b.GetDcrmPublicKey(args.PairID), msgHash.String(), msgContext)
	if err != nil {
		return nil, "", err
	}
	log.Info(b.ChainConfig.BlockChain+" DcrmSignTransaction start", "keyID", keyID, "msghash", msgHash.String(), "txid", args.SwapID)
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

	sender, err := b.PublicKeyToAddress(pubHex)
	if err != nil {
		return nil, "", errors.New("wrong dcrm public key")
	}

	b, err := hex.DecodeString(rsv)
	if err != nil {
		return
	}
	var signature []byte
	if len(b) == 65 {
		sig = b[:64]
	}
	stdsig := authtypes.StdSignature{
		PubKey:    pubkey,
		Signature: signature,
	}
	signedTx = authtypes.StdTx{
		Msgs:       tx.Msgs,
		Fee:        tx.Fee,
		Signatures: []authtypes.StdSignature{stdsig},
		Memo:       tx.Memo,
	}

	pairID := args.PairID
	token := b.GetTokenConfig(pairID)
	if b.EqualAddress(sender.token.DcrmAddress) == false {
		log.Error("DcrmSignTransaction verify sender failed", "have", sender, "want", token.DcrmAddress)
		return nil, "", errors.New("wrong sender address")
	}
	txHash = msgHash
	log.Info(b.ChainConfig.BlockChain+" DcrmSignTransaction success", "keyID", keyID, "txhash", txHash, "nonce", signedTx.Nonce())
	return signedTx, txHash, err
}

// SignTransaction sign tx with pairID
func (b *Bridge) SignTransaction(rawTx interface{}, pairID string) (signTx interface{}, txHash string, err error) {
	privKey := b.GetTokenConfig(pairID).GetDcrmAddressPrivateKey()
	return b.SignTransactionWithPrivateKey(rawTx, privKey)
}

// SignTransactionWithPrivateKey sign tx with ECDSA private key
func (b *Bridge) SignTransactionWithPrivateKey(rawTx interface{}, privKey *ecdsa.PrivateKey) (signTx interface{}, txHash string, err error) {
	// rawTx is of type authtypes.StdSignDoc
	tx, ok := rawTx.(StdSignContent)
	if !ok {
		return nil, "", errors.New("wrong raw tx param")
	}

	msgs := tx.Msgs
	fee = tx.Fee

	signBytes := authtypes.StdSignBytes(tx.ChainID, tx.AccountNumber, tx.Sequence, fee, msgs, tx.Memo)

	priv := secp256k1.PrivKey(btcec.PrivKey(*privKey).Serialize())
	signature, err := priv.Sign(signBytes)
	if err != nil {
		return nil, "", err
	}

	pub := priv.PubKey()

	signedTx, err := types.SignTx(tx, b.Signer, privKey)
	if err != nil {
		return nil, "", fmt.Errorf("sign tx failed, %v", err)
	}

	stdsig := authtypes.StdSignature{
		PubKey:    pub,
		Signature: signature,
	}

	signTx = authtypes.StdTx{
		Msgs:       msgs,
		Fee:        fee,
		Signatures: []authtypes.StdSignature{stdsig},
		Memo:       tx.Memo,
	}

	txHash = fmt.Sprintf("%X", tmhash.Sum(signBytes))

	return
}
