package terra

import (
	"encoding/base64"
	"fmt"

	"github.com/gogo/protobuf/proto"

	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/tendermint/tendermint/crypto/tmhash"
)

// wrapper is a wrapper around the tx.Tx proto.Message which retain the raw
// body and auth_info bytes.
type wrapper struct {
	tx *tx.Tx

	// bodyBz represents the protobuf encoding of TxBody. This should be encoding
	// from the client using TxRaw if the tx was decoded from the wire
	bodyBz []byte

	// authInfoBz represents the protobuf encoding of TxBody. This should be encoding
	// from the client using TxRaw if the tx was decoded from the wire
	authInfoBz []byte

	// signerData is the specific information needed to sign a transaction that generally
	// isn't included in the transaction body itself
	signerData *authsigning.SignerData
}

var (
	_ authsigning.Tx             = &wrapper{}
	_ client.TxBuilder           = &wrapper{}
	_ ante.HasExtensionOptionsTx = &wrapper{}
	_ ExtensionOptionsTxBuilder  = &wrapper{}
)

// ExtensionOptionsTxBuilder defines a TxBuilder that can also set extensions.
type ExtensionOptionsTxBuilder interface {
	client.TxBuilder

	SetExtensionOptions(...*codectypes.Any)
	SetNonCriticalExtensionOptions(...*codectypes.Any)
}

func newBuilder() *wrapper {
	return &wrapper{
		tx: &tx.Tx{
			Body: &tx.TxBody{},
			AuthInfo: &tx.AuthInfo{
				Fee: &tx.Fee{},
			},
		},
	}
}

// GetMsgs impl
func (w *wrapper) GetMsgs() []sdk.Msg {
	return w.tx.GetMsgs()
}

// ValidateBasic impl
func (w *wrapper) ValidateBasic() error {
	return w.tx.ValidateBasic()
}

func (w *wrapper) getBodyBytes() []byte {
	if len(w.bodyBz) == 0 {
		// if bodyBz is empty, then marshal the body. bodyBz will generally
		// be set to nil whenever SetBody is called so the result of calling
		// this method should always return the correct bytes. Note that after
		// decoding bodyBz is derived from TxRaw so that it matches what was
		// transmitted over the wire
		var err error
		w.bodyBz, err = proto.Marshal(w.tx.Body)
		if err != nil {
			panic(err)
		}
	}
	return w.bodyBz
}

func (w *wrapper) getAuthInfoBytes() []byte {
	if len(w.authInfoBz) == 0 {
		// if authInfoBz is empty, then marshal the body. authInfoBz will generally
		// be set to nil whenever SetAuthInfo is called so the result of calling
		// this method should always return the correct bytes. Note that after
		// decoding authInfoBz is derived from TxRaw so that it matches what was
		// transmitted over the wire
		var err error
		w.authInfoBz, err = proto.Marshal(w.tx.AuthInfo)
		if err != nil {
			panic(err)
		}
	}
	return w.authInfoBz
}

// GetSigners impl
func (w *wrapper) GetSigners() []sdk.AccAddress {
	return w.tx.GetSigners()
}

// GetPubkeys returns the pubkeys of signers if the pubkey is included in the signature
// If pubkey is not included in the signature, then nil is in the slice instead
func (w *wrapper) GetPubKeys() ([]cryptotypes.PubKey, error) {
	signerInfos := w.tx.AuthInfo.SignerInfos
	pks := make([]cryptotypes.PubKey, len(signerInfos))

	for i, si := range signerInfos {
		// NOTE: it is okay to leave this nil if there is no PubKey in the SignerInfo.
		// PubKey's can be left unset in SignerInfo.
		if si.PublicKey == nil {
			continue
		}

		pkAny := si.PublicKey.GetCachedValue()
		pk, ok := pkAny.(cryptotypes.PubKey)
		if ok {
			pks[i] = pk
		} else {
			return nil, sdkerrors.Wrapf(sdkerrors.ErrLogic, "Expecting PubKey, got: %T", pkAny)
		}
	}

	return pks, nil
}

// GetGas get gas
func (w *wrapper) GetGas() uint64 {
	return w.tx.AuthInfo.Fee.GasLimit
}

// GetFee get fee
func (w *wrapper) GetFee() sdk.Coins {
	return w.tx.AuthInfo.Fee.Amount
}

// FeePayer get fee payer
func (w *wrapper) FeePayer() sdk.AccAddress {
	feePayer := w.tx.AuthInfo.Fee.Payer
	if feePayer != "" {
		payerAddr, err := sdk.AccAddressFromBech32(feePayer)
		if err != nil {
			panic(err)
		}
		return payerAddr
	}
	// use first signer as default if no payer specified
	return w.GetSigners()[0]
}

// FeeGranter get fee granter
func (w *wrapper) FeeGranter() sdk.AccAddress {
	feePayer := w.tx.AuthInfo.Fee.Granter
	if feePayer != "" {
		granterAddr, err := sdk.AccAddressFromBech32(feePayer)
		if err != nil {
			panic(err)
		}
		return granterAddr
	}
	return nil
}

// GetMemo get memo
func (w *wrapper) GetMemo() string {
	return w.tx.Body.Memo
}

// GetTimeoutHeight returns the transaction's timeout height (if set).
func (w *wrapper) GetTimeoutHeight() uint64 {
	return w.tx.Body.TimeoutHeight
}

// GetSignaturesV2 get signature v2
func (w *wrapper) GetSignaturesV2() ([]signing.SignatureV2, error) {
	signerInfos := w.tx.AuthInfo.SignerInfos
	sigs := w.tx.Signatures
	pubKeys, err := w.GetPubKeys()
	if err != nil {
		return nil, err
	}
	n := len(signerInfos)
	res := make([]signing.SignatureV2, n)

	for i, si := range signerInfos {
		// handle nil signatures (in case of simulation)
		if si.ModeInfo == nil {
			res[i] = signing.SignatureV2{
				PubKey: pubKeys[i],
			}
		} else {
			var err error
			sigData, err := authtx.ModeInfoAndSigToSignatureData(si.ModeInfo, sigs[i])
			if err != nil {
				return nil, err
			}
			res[i] = signing.SignatureV2{
				PubKey:   pubKeys[i],
				Data:     sigData,
				Sequence: si.GetSequence(),
			}
		}
	}

	return res, nil
}

// SetMsgs set msgs
func (w *wrapper) SetMsgs(msgs ...sdk.Msg) error {
	anys := make([]*codectypes.Any, len(msgs))

	for i, msg := range msgs {
		var err error
		anys[i], err = codectypes.NewAnyWithValue(msg)
		if err != nil {
			return err
		}
	}

	w.tx.Body.Messages = anys

	// set bodyBz to nil because the cached bodyBz no longer matches tx.Body
	w.bodyBz = nil

	return nil
}

// SetTimeoutHeight sets the transaction's height timeout.
func (w *wrapper) SetTimeoutHeight(height uint64) {
	w.tx.Body.TimeoutHeight = height

	// set bodyBz to nil because the cached bodyBz no longer matches tx.Body
	w.bodyBz = nil
}

// SetMemo set memo
func (w *wrapper) SetMemo(memo string) {
	w.tx.Body.Memo = memo

	// set bodyBz to nil because the cached bodyBz no longer matches tx.Body
	w.bodyBz = nil
}

// SetGasLimit set gas limit
func (w *wrapper) SetGasLimit(limit uint64) {
	if w.tx.AuthInfo.Fee == nil {
		w.tx.AuthInfo.Fee = &tx.Fee{}
	}

	w.tx.AuthInfo.Fee.GasLimit = limit

	// set authInfoBz to nil because the cached authInfoBz no longer matches tx.AuthInfo
	w.authInfoBz = nil
}

// SetFeeAmount set fee amount
func (w *wrapper) SetFeeAmount(coins sdk.Coins) {
	if w.tx.AuthInfo.Fee == nil {
		w.tx.AuthInfo.Fee = &tx.Fee{}
	}

	w.tx.AuthInfo.Fee.Amount = coins

	// set authInfoBz to nil because the cached authInfoBz no longer matches tx.AuthInfo
	w.authInfoBz = nil
}

// SetFeePayer set fee payer
func (w *wrapper) SetFeePayer(feePayer sdk.AccAddress) {
	if w.tx.AuthInfo.Fee == nil {
		w.tx.AuthInfo.Fee = &tx.Fee{}
	}

	w.tx.AuthInfo.Fee.Payer = feePayer.String()

	// set authInfoBz to nil because the cached authInfoBz no longer matches tx.AuthInfo
	w.authInfoBz = nil
}

// SetFeeGranter set fee granter
func (w *wrapper) SetFeeGranter(feeGranter sdk.AccAddress) {
	if w.tx.AuthInfo.Fee == nil {
		w.tx.AuthInfo.Fee = &tx.Fee{}
	}

	w.tx.AuthInfo.Fee.Granter = feeGranter.String()

	// set authInfoBz to nil because the cached authInfoBz no longer matches tx.AuthInfo
	w.authInfoBz = nil
}

// SetSignatures set signatures
func (w *wrapper) SetSignatures(signatures ...signing.SignatureV2) error {
	n := len(signatures)
	signerInfos := make([]*tx.SignerInfo, n)
	rawSigs := make([][]byte, n)

	for i, sig := range signatures {
		var modeInfo *tx.ModeInfo
		modeInfo, rawSigs[i] = authtx.SignatureDataToModeInfoAndSig(sig.Data)
		any, err := codectypes.NewAnyWithValue(sig.PubKey)
		if err != nil {
			return err
		}
		signerInfos[i] = &tx.SignerInfo{
			PublicKey: any,
			ModeInfo:  modeInfo,
			Sequence:  sig.Sequence,
		}
	}

	w.setSignerInfos(signerInfos)
	w.setSignatures(rawSigs)

	return nil
}

func (w *wrapper) setSignerInfos(infos []*tx.SignerInfo) {
	w.tx.AuthInfo.SignerInfos = infos
	// set authInfoBz to nil because the cached authInfoBz no longer matches tx.AuthInfo
	w.authInfoBz = nil
}

func (w *wrapper) setSignatures(sigs [][]byte) {
	w.tx.Signatures = sigs
}

// GetTx get tx
func (w *wrapper) GetTx() authsigning.Tx {
	return w
}

// GetProtoTx get proto tx
func (w *wrapper) GetProtoTx() *tx.Tx {
	return w.tx
}

// WrapTx creates a TxBuilder wrapper around a tx.Tx proto message.
func WrapTx(protoTx *tx.Tx) client.TxBuilder {
	return &wrapper{
		tx: protoTx,
	}
}

// GetExtensionOptions get extension options
func (w *wrapper) GetExtensionOptions() []*codectypes.Any {
	return w.tx.Body.ExtensionOptions
}

// GetNonCriticalExtensionOptions get non critical extension options
func (w *wrapper) GetNonCriticalExtensionOptions() []*codectypes.Any {
	return w.tx.Body.NonCriticalExtensionOptions
}

// SetExtensionOptions set extension options
func (w *wrapper) SetExtensionOptions(extOpts ...*codectypes.Any) {
	w.tx.Body.ExtensionOptions = extOpts
	w.bodyBz = nil
}

// SetNonCriticalExtensionOptions set non critical extension options
func (w *wrapper) SetNonCriticalExtensionOptions(extOpts ...*codectypes.Any) {
	w.tx.Body.NonCriticalExtensionOptions = extOpts
	w.bodyBz = nil
}

// GetSignerData get signer data
func (w *wrapper) GetSignerData() *authsigning.SignerData {
	return w.signerData
}

// SetSignerData set signer data
func (w *wrapper) SetSignerData(chainID string, accountNumber, sequence uint64) {
	w.signerData = &authsigning.SignerData{
		ChainID:       chainID,
		AccountNumber: accountNumber,
		Sequence:      sequence,
	}
}

// GetSignBytes get sign bytes
func (w *wrapper) GetSignBytes() ([]byte, error) {
	return authtx.DirectSignBytes(
		w.getBodyBytes(),
		w.getAuthInfoBytes(),
		w.signerData.ChainID,
		w.signerData.AccountNumber)
}

// GetSignedTx get signed raw tx to broadcast
func (w *wrapper) GetSignedTx() (signedTx []byte, txHash string, err error) {
	txBytes, err := w.GetProtoTx().Marshal()
	if err != nil {
		return nil, "", err
	}
	signedTx = []byte(base64.StdEncoding.EncodeToString(txBytes))
	txHash = fmt.Sprintf("%X", tmhash.Sum(txBytes))
	return signedTx, txHash, nil
}

// GetTxBytes get tx marshal bytes
func (w *wrapper) GetTxBytes() ([]byte, error) {
	return w.GetProtoTx().Marshal()
}
