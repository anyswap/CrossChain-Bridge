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
	if !b.ChainConfig.EnableDynamicFeeTx {
		gasPrice, errt := b.getGasPrice(args)
		if errt == nil && args.Extra.EthExtra.GasPrice.Cmp(gasPrice) < 0 {
			log.Info(b.ChainConfig.BlockChain+" DcrmSignTransaction update gas price", "txid", args.SwapID, "oldGasPrice", args.Extra.EthExtra.GasPrice, "newGasPrice", gasPrice)
			args.Extra.EthExtra.GasPrice = gasPrice
			tx.SetGasPrice(gasPrice)
		}
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

	token := b.GetTokenConfig(args.PairID)
	signedTx, err := b.signTxWithSignature(tx, signature, common.HexToAddress(token.DcrmAddress))
	if err != nil {
		return nil, "", err
	}
	txHash, err = b.CalcTransactionHash(signedTx)
	if err != nil {
		return nil, "", fmt.Errorf("calc signed tx hash failed, %w", err)
	}
	log.Info(b.ChainConfig.BlockChain+" DcrmSignTransaction success", "keyID", keyID, "txid", args.SwapID, "txhash", txHash, "nonce", signedTx.Nonce())
	return signedTx, txHash, nil
}

func (b *Bridge) signTxWithSignature(tx *types.Transaction, signature []byte, signerAddr common.Address) (*types.Transaction, error) {
	signer := b.Signer
	vPos := crypto.SignatureLength - 1
	for i := 0; i < 2; i++ {
		signedTx, err := tx.WithSignature(signer, signature)
		if err != nil {
			return nil, err
		}

		sender, err := types.Sender(signer, signedTx)
		if err != nil {
			return nil, err
		}

		if sender == signerAddr {
			return signedTx, nil
		}

		signature[vPos] ^= 0x1 // v can only be 0 or 1
	}

	return nil, errors.New("wrong sender address")
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
		return nil, "", fmt.Errorf("sign tx failed, %w", err)
	}

	txHash, err = b.CalcTransactionHash(signedTx)
	if err != nil {
		return nil, "", fmt.Errorf("calc signed tx hash failed, %w", err)
	}
	log.Info(b.ChainConfig.BlockChain+" SignTransaction success", "txhash", txHash, "nonce", signedTx.Nonce())
	return signedTx, txHash, err
}

// CalcTransactionHash calc tx hash
func (b *Bridge) CalcTransactionHash(tx *types.Transaction) (txHash string, err error) {
	hash := tx.Hash()
	if hash == common.EmptyHash {
		return hash.Hex(), errors.New("empty tx hash")
	}
	return tx.Hash().Hex(), nil
}

// GetSignedTxHashOfKeyID get signed tx hash by keyID (called by oracle)
func (b *Bridge) GetSignedTxHashOfKeyID(keyID, pairID string, rawTx interface{}) (txHash string, err error) {
	tx, ok := rawTx.(*types.Transaction)
	if !ok {
		return "", errors.New("wrong raw tx of keyID " + keyID)
	}
	rsvs, err := dcrm.GetSignStatusByKeyID(keyID)
	if err != nil {
		return "", err
	}
	if len(rsvs) != 1 {
		return "", errors.New("wrong number of rsvs of keyID " + keyID)
	}

	rsv := rsvs[0]
	signature := common.FromHex(rsv)
	if len(signature) != crypto.SignatureLength {
		return "", errors.New("wrong signature of keyID " + keyID)
	}
	token := b.GetTokenConfig(pairID)
	signedTx, err := b.signTxWithSignature(tx, signature, common.HexToAddress(token.DcrmAddress))
	if err != nil {
		return "", err
	}
	txHash, err = b.CalcTransactionHash(signedTx)
	if err != nil {
		return "", fmt.Errorf("calc signed tx hash failed, %w", err)
	}
	return txHash, nil
}
