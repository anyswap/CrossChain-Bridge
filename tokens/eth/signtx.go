package eth

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/dcrm"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tools/crypto"
	"github.com/anyswap/CrossChain-Bridge/types"
)

func (b *Bridge) verifyTransactionWithArgs(rawTx interface{}, args *tokens.BuildTxArgs) (*types.Transaction, error) {
	tx, ok := rawTx.(*types.Transaction)
	if !ok {
		return nil, errors.New("wrong raw tx param")
	}
	if tx.To() == nil || *tx.To() == (common.Address{}) {
		return nil, fmt.Errorf("[sign] verify tx receiver failed")
	}
	tokenCfg := b.GetTokenConfig(args.PairID)
	if tokenCfg == nil {
		return nil, fmt.Errorf("[sign] verify tx with unknown pairID '%v'", args.PairID)
	}
	checkReceiver := tokenCfg.ContractAddress
	if args.SwapType == tokens.SwapoutType && !tokenCfg.IsErc20() {
		checkReceiver = args.Bind
	}
	if !strings.EqualFold(tx.To().String(), checkReceiver) {
		return nil, fmt.Errorf("[sign] verify tx receiver failed")
	}
	return tx, nil
}

// DcrmSignTransaction dcrm sign raw tx
func (b *Bridge) DcrmSignTransaction(rawTx interface{}, args *tokens.BuildTxArgs) (signTx interface{}, txHash string, err error) {
	tx, err := b.verifyTransactionWithArgs(rawTx, args)
	if err != nil {
		return nil, "", err
	}
	gasPrice, err := b.getGasPrice()
	if err == nil && args.Extra.EthExtra.GasPrice.Cmp(gasPrice) < 0 {
		args.Extra.EthExtra.GasPrice = gasPrice
	}
	signer := b.Signer
	msgHash := signer.Hash(tx)
	jsondata, _ := json.Marshal(args)
	msgContext := string(jsondata)

	log.Info(b.ChainConfig.BlockChain+" DcrmSignTransaction start", "msghash", msgHash.String(), "txid", args.SwapID)
	keyID, rsvs, err := dcrm.DoSignOne(b.GetDcrmPublicKey(args.PairID), msgHash.String(), msgContext)
	if err != nil {
		return nil, "", err
	}
	log.Info(b.ChainConfig.BlockChain+" DcrmSignTransaction finished", "keyID", keyID, "msghash", msgHash.String(), "txid", args.SwapID)

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
	if sender != common.HexToAddress(token.DcrmAddress) {
		log.Error("DcrmSignTransaction verify sender failed", "have", sender.String(), "want", token.DcrmAddress)
		return nil, "", errors.New("wrong sender address")
	}
	txHash = signedTx.Hash().String()
	log.Info(b.ChainConfig.BlockChain+" DcrmSignTransaction success", "keyID", keyID, "txid", args.SwapID, "txhash", txHash, "nonce", signedTx.Nonce())
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
