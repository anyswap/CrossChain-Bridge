package ltc

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ltcsuite/ltcd/btcec"
	"github.com/ltcsuite/ltcd/chaincfg"
	"github.com/ltcsuite/ltcd/chaincfg/chainhash"
	"github.com/ltcsuite/ltcd/txscript"
	"github.com/ltcsuite/ltcd/wire"
	"github.com/ltcsuite/ltcutil"
)

type ltcAmountType = ltcutil.Amount
type wireTxInType = wire.TxIn
type wireTxOutType = wire.TxOut

func disasmScriptToString(pkScript []byte) (string, error) {
	return txscript.DisasmString(pkScript)
}

func isValidValue(value ltcAmountType) bool {
	return value > 0 && value <= ltcutil.MaxSatoshi
}

func newAmount(value float64) (ltcAmountType, error) {
	return ltcutil.NewAmount(value)
}

// GetChainParams get chain config (net params)
func (b *Bridge) GetChainParams() *chaincfg.Params {
	networkID := strings.ToLower(b.ChainConfig.NetID)
	switch networkID {
	case netMainnet:
		return &chaincfg.MainNetParams
	case netTestnet4:
		return &chaincfg.TestNet4Params
	default:
		return &chaincfg.MainNetParams
	}
}

// ParsePkScript parse pkScript
func (b *Bridge) ParsePkScript(pkScript []byte) (txscript.PkScript, error) {
	return txscript.ParsePkScript(pkScript)
}

// GetPayToAddrScript get pay to address script
func (b *Bridge) GetPayToAddrScript(address string) ([]byte, error) {
	toAddr, err := b.DecodeAddress(address)
	if err != nil {
		return nil, fmt.Errorf("decode ltc address '%v' failed. %w", address, err)
	}
	return txscript.PayToAddrScript(toAddr)
}

// GetP2shRedeemScript get p2sh redeem script
func (b *Bridge) GetP2shRedeemScript(memo, pubKeyHash []byte) (redeemScript []byte, err error) {
	return txscript.NewScriptBuilder().
		AddData(memo).AddOp(txscript.OP_DROP).
		AddOp(txscript.OP_DUP).AddOp(txscript.OP_HASH160).AddData(pubKeyHash).
		AddOp(txscript.OP_EQUALVERIFY).AddOp(txscript.OP_CHECKSIG).
		Script()
}

// NullDataScript encap
func (b *Bridge) NullDataScript(memo string) ([]byte, error) {
	return txscript.NullDataScript([]byte(memo))
}

// IsPayToScriptHash is p2sh
func (b *Bridge) IsPayToScriptHash(sigScript []byte) bool {
	return txscript.IsPayToScriptHash(sigScript)
}

// CalcSignatureHash calc sig hash
func (b *Bridge) CalcSignatureHash(sigScript []byte, tx *wire.MsgTx, i int) (sigHash []byte, err error) {
	return txscript.CalcSignatureHash(sigScript, txscript.SigHashAll, tx, i)
}

// SerializeSignature serialize signature
func (b *Bridge) SerializeSignature(r, s *big.Int) []byte {
	sign := &btcec.Signature{R: r, S: s}
	return append(sign.Serialize(), byte(txscript.SigHashAll))
}

// GetSigScript get script
func (b *Bridge) GetSigScript(sigScripts [][]byte, prevScript, signData, cPkData []byte, i int) (sigScript []byte, err error) {
	scriptClass := txscript.GetScriptClass(prevScript)
	switch scriptClass {
	case txscript.PubKeyHashTy:
		sigScript, err = txscript.NewScriptBuilder().AddData(signData).AddData(cPkData).Script()
	case txscript.ScriptHashTy:
		if sigScripts == nil {
			err = fmt.Errorf("call MakeSignedTransaction spend p2sh without redeem scripts")
		} else {
			redeemScript := sigScripts[i]
			err = b.VerifyRedeemScript(prevScript, redeemScript)
			if err == nil {
				sigScript, err = txscript.NewScriptBuilder().AddData(signData).AddData(cPkData).AddData(redeemScript).Script()
			}
		}
	default:
		err = fmt.Errorf("unsupport to spend '%v' output", scriptClass.String())
	}
	return sigScript, err
}

// SerializePublicKey serialize ecdsa public key
func (b *Bridge) SerializePublicKey(ecPub *ecdsa.PublicKey, compressed bool) []byte {
	if compressed {
		return (*btcec.PublicKey)(ecPub).SerializeCompressed()
	}
	return (*btcec.PublicKey)(ecPub).SerializeUncompressed()
}

// ToCompressedPublicKey convert to compressed public key if not
func (b *Bridge) ToCompressedPublicKey(pkData []byte) ([]byte, error) {
	pubKey, err := btcec.ParsePubKey(pkData, btcec.S256())
	if err != nil {
		return nil, err
	}
	return pubKey.SerializeCompressed(), nil
}

// GetPublicKeyFromECDSA get public key from ecdsa private key
func (b *Bridge) GetPublicKeyFromECDSA(privKey *ecdsa.PrivateKey, compressed bool) []byte {
	if compressed {
		return (*btcec.PublicKey)(&privKey.PublicKey).SerializeCompressed()
	}
	return (*btcec.PublicKey)(&privKey.PublicKey).SerializeUncompressed()
}

// SignWithECDSA sign with ecdsa private key
func (b *Bridge) SignWithECDSA(privKey *ecdsa.PrivateKey, msgHash []byte) (rsv string, err error) {
	signature, err := (*btcec.PrivateKey)(privKey).Sign(msgHash)
	if err != nil {
		return "", err
	}
	rr := fmt.Sprintf("%064X", signature.R)
	ss := fmt.Sprintf("%064X", signature.S)
	rsv = fmt.Sprintf("%s%s00", rr, ss)
	return rsv, nil
}

// NewTxIn new txin
func (b *Bridge) NewTxIn(txid string, vout uint32, pkScript []byte) (*wire.TxIn, error) {
	txHash, err := chainhash.NewHashFromStr(txid)
	if err != nil {
		return nil, err
	}
	prevOutPoint := wire.NewOutPoint(txHash, vout)
	return wire.NewTxIn(prevOutPoint, pkScript, nil), nil
}

// NewTxOut new txout
func (b *Bridge) NewTxOut(amount int64, pkScript []byte) *wire.TxOut {
	return wire.NewTxOut(amount, pkScript)
}

// NewMsgTx new msg tx
func (b *Bridge) NewMsgTx(inputs []*wire.TxIn, outputs []*wire.TxOut, locktime uint32) *wire.MsgTx {
	return &wire.MsgTx{
		Version:  wire.TxVersion,
		TxIn:     inputs,
		TxOut:    outputs,
		LockTime: locktime,
	}
}
