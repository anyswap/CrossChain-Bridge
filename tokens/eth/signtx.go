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
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tools/crypto"
	"github.com/anyswap/CrossChain-Bridge/types"
)

const (
	retryGetSignStatusCount    = 70
	retryGetSignStatusInterval = 10 * time.Second
)

var (
	errSenderMismatch = errors.New("sender address mismatch")
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
	msgHash := b.Signer.Hash(tx)
	jsondata, _ := json.Marshal(args)
	msgContext := string(jsondata)

	rootPubkey, err := b.prepareDcrmSign(args)
	if err != nil {
		return nil, "", err
	}

	if params.IsDebugging() {
		log.Warn("DcrmSignTransaction start", "raw", tx.RawStr(), "msgHash", msgHash.String(), "txid", args.SwapID, "pairID", args.PairID)
	}
	rpcAddr, keyID, err := dcrm.DoSignOne(rootPubkey, args.InputCode, msgHash.String(), msgContext)
	if err != nil {
		return nil, "", err
	}
	log.Info(b.ChainConfig.BlockChain+" DcrmSignTransaction start", "keyID", keyID, "msghash", msgHash.String(), "txid", args.SwapID, "pairID", args.PairID)
	time.Sleep(retryGetSignStatusInterval)

	rsv, err := b.getSignStatus(keyID, rpcAddr, args)
	if err != nil {
		return nil, "", err
	}

	signedTx, err := b.signTransactionUseRsv(tx, rsv, keyID, args)
	if err != nil {
		if err != errSenderMismatch {
			return nil, "", err
		}
		if params.IsDebugging() {
			log.Error("retry sign with inverted V value of rsv", "keyID", keyID, "txid", args.SwapID, "pairID", args.PairID)
		}
		// invert v value and retry sign
		invRsv := invertV(rsv)
		signedTx, err = b.signTransactionUseRsv(tx, invRsv, keyID, args)
		if err != nil {
			return nil, "", err
		}
	}

	txHash = signedTx.Hash().String()
	log.Info(b.ChainConfig.BlockChain+" DcrmSignTransaction success", "keyID", keyID, "txhash", txHash, "nonce", signedTx.Nonce(), "txid", args.SwapID, "pairID", args.PairID)
	return signedTx, txHash, err
}

func (b *Bridge) prepareDcrmSign(args *tokens.BuildTxArgs) (rootPubkey string, err error) {
	rootPubkey = b.GetDcrmPublicKey(args.PairID)

	signerAddr := args.From
	if signerAddr == "" {
		token := b.GetTokenConfig(args.PairID)
		signerAddr = token.DcrmAddress
	}

	if args.InputCode != "" {
		childPubkey, err := dcrm.GetBip32ChildKey(rootPubkey, args.InputCode)
		if err != nil {
			return "", err
		}
		signerAddr, err = b.PublicKeyToAddress(childPubkey)
		if err != nil {
			return "", err
		}
	}

	if args.From == "" {
		args.From = signerAddr
	} else if !strings.EqualFold(args.From, signerAddr) {
		log.Error("dcrm sign sender mismath", "inputCode", args.InputCode, "have", args.From, "want", signerAddr)
		return rootPubkey, fmt.Errorf("dcrm sign sender mismath")
	}
	return rootPubkey, nil
}

func invertV(rsv string) string {
	signature := common.FromHex(rsv)
	v := signature[len(signature)-1]
	newV := (v + 1) % 2
	newSignature := append(signature[:len(signature)-2], newV)
	return common.Bytes2Hex(newSignature)
}

func (b *Bridge) signTransactionUseRsv(tx *types.Transaction, rsv, keyID string, args *tokens.BuildTxArgs) (signedTx *types.Transaction, err error) {
	signature := common.FromHex(rsv)

	if len(signature) != crypto.SignatureLength {
		log.Error("DcrmSignTransaction wrong length of signature")
		return nil, errors.New("wrong signature of keyID " + keyID)
	}

	signedTx, err = tx.WithSignature(b.Signer, signature)
	if err != nil {
		return nil, err
	}

	if params.IsDebugging() {
		log.Warn("DcrmSignTransaction finished", "tx", tx.RawStr(), "signedTx", signedTx.RawStr(), "keyID", keyID, "txid", args.SwapID, "pairID", args.PairID)
	}

	sender, err := types.Sender(b.Signer, signedTx)
	if err != nil {
		return nil, err
	}

	if !strings.EqualFold(sender.String(), args.From) {
		log.Error("DcrmSignTransaction verify sender failed", "have", sender.String(), "want", args.From, "keyID", keyID, "txid", args.SwapID, "pairID", args.PairID)
		return nil, errSenderMismatch
	}
	return signedTx, err
}

func (b *Bridge) getSignStatus(keyID, rpcAddr string, args *tokens.BuildTxArgs) (rsv string, err error) {
	var signStatus *dcrm.SignStatus
	i := 0
	for ; i < retryGetSignStatusCount; i++ {
		signStatus, err = dcrm.GetSignStatus(keyID, rpcAddr)
		if err == nil {
			if len(signStatus.Rsv) != 1 {
				return "", fmt.Errorf("get sign status require one rsv but have %v (keyID = %v)", len(signStatus.Rsv), keyID)
			}

			rsv = signStatus.Rsv[0]
			break
		}
		switch err {
		case dcrm.ErrGetSignStatusFailed, dcrm.ErrGetSignStatusTimeout:
			return "", err
		}
		log.Warn("retry get sign status as error", "err", err, "txid", args.SwapID, "keyID", keyID, "bridge", args.Identifier, "swaptype", args.SwapType.String(), "pairID", args.PairID)
		time.Sleep(retryGetSignStatusInterval)
	}
	if i == retryGetSignStatusCount || rsv == "" {
		return "", errors.New("get sign status failed")
	}

	log.Trace(b.ChainConfig.BlockChain+" DcrmSignTransaction get rsv success", "pairID", args.PairID, "txid", args.SwapID, "keyID", keyID, "rsv", rsv)
	return rsv, nil
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
