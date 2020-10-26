package eth

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
)

const (
	retryGetSignStatusCount    = 70
	retryGetSignStatusInterval = 10 * time.Second
)

func (b *Bridge) verifyTransactionWithArgs(tx *types.Transaction, args *tokens.BuildTxArgs) error {
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
	tx, ok := rawTx.(*types.Transaction)
	if !ok {
		return nil, "", errors.New("wrong raw tx param")
	}
	err = b.verifyTransactionWithArgs(tx, args)
	if err != nil {
		return nil, "", err
	}
	signer := b.Signer
	msgHash := signer.Hash(tx)
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

	signedTx, err := tx.WithSignature(signer, signature)
	if err != nil {
		return nil, "", err
	}

	sender, err := types.Sender(signer, signedTx)
	if err != nil {
		return nil, "", err
	}

	pairID := args.PairID
	token := b.GetTokenConfig(pairID)
	if sender.String() != token.DcrmAddress {
		log.Error("DcrmSignTransaction verify sender failed", "have", sender.String(), "want", token.DcrmAddress)
		return nil, "", errors.New("wrong sender address")
	}
	txHash = signedTx.Hash().String()
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
	tx, ok := rawTx.(*types.Transaction)
	if !ok {
		return nil, "", errors.New("wrong raw tx param")
	}

	signedTx, err := types.SignTx(tx, b.Signer, privKey)
	if err != nil {
		return nil, "", fmt.Errorf("sign tx failed, %v", err)
	}

	txHash = signedTx.Hash().String()
	log.Info(b.ChainConfig.BlockChain+" SignTransaction success", "txhash", txHash, "nonce", signedTx.Nonce())
	return signedTx, txHash, err
}
