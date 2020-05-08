package eth

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/dcrm"
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	"github.com/fsn-dev/crossChain-Bridge/tools/crypto"
	"github.com/fsn-dev/crossChain-Bridge/types"
)

var (
	retryCount    = 30
	retryInterval = 10 * time.Second
	waitInterval  = 10 * time.Second
)

func (b *EthBridge) DcrmSignTransaction(rawTx interface{}, args *tokens.BuildTxArgs) (interface{}, error) {
	tx, ok := rawTx.(*types.Transaction)
	if !ok {
		return nil, errors.New("wrong raw tx param")
	}
	signer := b.Signer
	msgHash := signer.Hash(tx)
	updateBuildTxArgs(tx, args)
	jsondata, _ := json.Marshal(args)
	msgContext := string(jsondata)
	keyID, err := dcrm.DoSign(msgHash.String(), msgContext)
	if err != nil {
		return nil, err
	}
	log.Info("DcrmSignTransaction start", "keyID", keyID, "msghash", msgHash.String())
	time.Sleep(waitInterval)

	var rsv string
	i := 0
	for ; i < retryCount; i++ {
		signStatus, err := dcrm.GetSignStatus(keyID)
		if err == nil {
			rsv = signStatus.Rsv
			break
		}
		log.Debug("retry get sign status as error", "err", err)
		time.Sleep(retryInterval)
	}
	if i == retryCount {
		return nil, errors.New("get sign status failed")
	}

	log.Trace("DcrmSignTransaction get rsv success", "rsv", rsv)

	signature := common.FromHex(rsv)

	if len(signature) != crypto.SignatureLength {
		log.Error("DcrmSignTransaction wrong length of signature")
		return nil, errors.New("wrong signature of keyID " + keyID)
	}

	signedTx, err := tx.WithSignature(signer, signature)
	if err != nil {
		return nil, err
	}

	sender, err := types.Sender(signer, signedTx)
	if err != nil {
		return nil, err
	}

	token := b.TokenConfig
	if sender.String() != *token.DcrmAddress {
		log.Error("DcrmSignTransaction verify sender failed", "have", sender.String(), "want", *token.DcrmAddress)
		return nil, errors.New("wrong sender address")
	}
	log.Info("DcrmSignTransaction success", "keyID", keyID, "txhash", signedTx.Hash().String())
	return signedTx, err
}

func updateBuildTxArgs(tx *types.Transaction, args *tokens.BuildTxArgs) {
	if args.Gas == nil {
		gas := tx.Gas()
		args.Gas = &gas
	}
	if args.GasPrice == nil {
		gasPrice := tx.GasPrice()
		args.GasPrice = gasPrice
	}
	if args.Nonce == nil {
		nonce := tx.Nonce()
		args.Nonce = &nonce
	}
}
