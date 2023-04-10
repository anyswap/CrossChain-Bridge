package eth

import (
	"crypto/ecdsa"
	"encoding/hex"
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
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/zksync-sdk/zksync2-go"
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
	if b.IsZKSync() {
		return b.DcrmSignZkSyncTransaction(rawTx, args)
	}
	tx, err := b.verifyTransactionWithArgs(rawTx, args)
	if err != nil {
		return nil, "", err
	}
	gasPrice, err := b.getGasPrice(args)
	if err == nil && args.Extra.EthExtra.GasPrice.Cmp(gasPrice) < 0 {
		log.Info(b.ChainConfig.BlockChain+" DcrmSignTransaction update gas price", "txid", args.SwapID, "oldGasPrice", args.Extra.EthExtra.GasPrice, "newGasPrice", gasPrice)
		args.Extra.EthExtra.GasPrice = gasPrice
		tx.SetGasPrice(gasPrice)
	}
	signer := b.Signer
	msgHash := signer.Hash(tx)
	jsondata, _ := json.Marshal(args)
	msgContext := string(jsondata)

	log.Info(b.ChainConfig.BlockChain+" DcrmSignTransaction start", "msghash", msgHash.String(), "txid", args.SwapID)
	keyID, rsvs, err := dcrm.DoSignOne(b.GetDcrmPublicKey(args.PairID), msgHash.String(), msgContext)
	if err != nil {
		log.Info(b.ChainConfig.BlockChain+" DcrmSignTransaction failed", "keyID", keyID, "msghash", msgHash.String(), "txid", args.SwapID, "err", err)
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
	txHash = signedTx.Hash().String()
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
		return nil, "", fmt.Errorf("sign tx failed, %v", err)
	}

	txHash = signedTx.Hash().String()
	log.Info(b.ChainConfig.BlockChain+" SignTransaction success", "txhash", txHash, "nonce", signedTx.Nonce())
	return signedTx, txHash, err
}

func (b *Bridge) verifyZkSyncTransactionReceiver(rawTx interface{}, args *tokens.BuildTxArgs) (*zksync2.Transaction712, error) {
	tx, ok := rawTx.(*zksync2.Transaction712)
	if !ok {
		return nil, errors.New("[sign] wrong raw tx param")
	}
	if tx.To == nil || *tx.To == (ethcommon.Address{}) {
		return nil, errors.New("[sign] tx receiver is empty")
	}
	tokenCfg := b.GetTokenConfig(args.PairID)
	if tokenCfg == nil {
		return nil, fmt.Errorf("[sign] verify tx with unknown pairID '%v'", args.PairID)
	}
	checkReceiver := tokenCfg.ContractAddress
	if args.SwapType == tokens.SwapoutType && !tokenCfg.IsErc20() {
		checkReceiver = args.Bind
	}
	if !strings.EqualFold(tx.To.String(), checkReceiver) {
		return nil, fmt.Errorf("[sign] verify tx receiver failed")
	}
	return tx, nil
}

func HashTypedData(data apitypes.TypedData) ([]byte, error) {
	domain, err := data.HashStruct("EIP712Domain", data.Domain.Map())
	if err != nil {
		return nil, fmt.Errorf("failed to get hash of typed data domain: %w", err)
	}
	dataHash, err := data.HashStruct(data.PrimaryType, data.Message)
	if err != nil {
		return nil, fmt.Errorf("failed to get hash of typed message: %w", err)
	}
	prefixedData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domain), string(dataHash)))
	prefixedDataHash := crypto.Keccak256(prefixedData)
	return prefixedDataHash, nil
}

type SignedZKSyncTx struct {
	Raw []byte
}

func (b *Bridge) DcrmSignZkSyncTransaction(rawTx interface{}, args *tokens.BuildTxArgs) (signTx interface{}, txHash string, err error) {
	tx, err := b.verifyZkSyncTransactionReceiver(rawTx, args)
	if err != nil {
		return nil, "", err
	}

	domain := zksync2.DefaultEip712Domain(b.SignerChainID.Int64())
	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			tx.GetEIP712Type():     tx.GetEIP712Types(),
			domain.GetEIP712Type(): domain.GetEIP712Types(),
		},
		PrimaryType: tx.GetEIP712Type(),
		Domain:      domain.GetEIP712Domain(),
		Message:     tx.GetEIP712Message(),
	}
	msgHash, err := HashTypedData(typedData)
	if err != nil {
		return nil, "", err
	}

	jsondata, _ := json.Marshal(args.GetExtraArgs())
	msgContext := string(jsondata)

	txid := args.SwapID
	logPrefix := b.ChainConfig.BlockChain + " DcrmSignTransaction "
	log.Info(logPrefix+"start", "txid", txid, "msghash", fmt.Sprintf("%x", msgHash))

	keyID, rsvs, err := dcrm.DoSignOne(b.GetDcrmPublicKey(args.PairID), fmt.Sprintf("%x", msgHash), msgContext)
	if err != nil {
		log.Info(logPrefix+"failed", "keyID", keyID, "txid", txid, "err", err)
		return nil, "", err
	}
	log.Info(logPrefix+"finished", "keyID", keyID, "txid", txid, "msghash", fmt.Sprintf("%x", msgHash))

	if len(rsvs) != 1 {
		log.Warn("get sign status require one rsv but return many",
			"rsvs", len(rsvs), "keyID", keyID, "txid", txid)
		return nil, "", errors.New("get sign status require one rsv but return many")
	}

	rsv := rsvs[0]
	log.Trace(logPrefix+"get rsv signature success", "keyID", keyID, "txid", txid, "rsv", rsv)
	signature := common.FromHex(rsv)
	if len(signature) != crypto.SignatureLength {
		log.Error("wrong signature length", "keyID", keyID, "txid", txid, "have", len(signature), "want", crypto.SignatureLength)
		return nil, "", errors.New("wrong signature length")
	}

	sig, _ := hex.DecodeString(rsv)
	if sig[64] < 27 {
		sig[64] += 27
	}

	signedRawTx, err := tx.RLPValues(sig)
	if err != nil {
		return nil, "", err
	}

	signedTx := &SignedZKSyncTx{
		Raw: signedRawTx,
	}

	digest := []byte{}
	digest = append(digest, msgHash...)
	digest = append(digest, crypto.Keccak256(sig)...)
	txHash = fmt.Sprintf("0x%x", crypto.Keccak256(digest))

	log.Info(logPrefix+"success", "keyID", keyID, "txid", txid, "txhash", txHash, "nonce", tx.Nonce)
	return signedTx, txHash, nil
}
