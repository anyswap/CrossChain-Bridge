package block

import (
	"crypto/ecdsa"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

var bigOne = big.NewInt(1)

// MainNetParams is blocknet mainnet cfg
var MainNetParams = chaincfg.Params{
	Name: "mainnet",
	Net:  wire.MainNet,

	// Chain parameters
	PowLimit:                 new(big.Int).Sub(new(big.Int).Lsh(bigOne, 224), bigOne),
	PowLimitBits:             0x00000fff,
	BIP0034Height:            1,
	BIP0065Height:            1,
	BIP0066Height:            1,
	CoinbaseMaturity:         100,
	SubsidyReductionInterval: 210000,
	TargetTimespan:           time.Minute * 1, // 1 minute
	TargetTimePerBlock:       time.Minute * 1, // 1 minute
	RetargetAdjustmentFactor: 4,               // 25% less, 400% more
	ReduceMinDifficulty:      false,
	MinDiffReductionTime:     0,
	GenerateSupported:        false,

	// Checkpoints ordered from oldest to newest.
	Checkpoints: []chaincfg.Checkpoint{},

	// Consensus rule change deployments.
	//
	// The miner confirmation window is defined as:
	//   target proof of work timespan / target proof of work spacing
	RuleChangeActivationThreshold: 1368, // 95% of MinerConfirmationWindow
	MinerConfirmationWindow:       1440, //
	Deployments: [chaincfg.DefinedDeployments]chaincfg.ConsensusDeployment{
		chaincfg.DeploymentTestDummy: {
			BitNumber:  28,
			StartTime:  1199145601, // January 1, 2008 UTC
			ExpireTime: 1230767999, // December 31, 2008 UTC
		},
		chaincfg.DeploymentCSV: {
			BitNumber:  0,
			StartTime:  0,             // Always vote
			ExpireTime: math.MaxInt64, // No timeout
		},
		chaincfg.DeploymentSegwit: {
			BitNumber:  1,
			StartTime:  1584295200, // March 15, 2020
			ExpireTime: 1589565600, // May 15, 2020
		},
	},

	// Mempool parameters
	RelayNonStdTxs: false,

	// Human-readable part for Bech32 encoded segwit addresses, as defined in
	// BIP 173.
	Bech32HRPSegwit: "block", // always block for mainnet

	// Address encoding magics
	PubKeyHashAddrID:        0x1a, // starts with B
	ScriptHashAddrID:        0x1c, // starts with C
	PrivateKeyID:            0x9a, // starts with 6 (uncompressed) or P (compressed)
	WitnessPubKeyHashAddrID: 0x06, // starts with p2
	WitnessScriptHashAddrID: 0x0A, // starts with 7Xh

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0x88, 0xAD, 0xE4}, // starts with xprv
	HDPublicKeyID:  [4]byte{0x04, 0x88, 0xB2, 0x1E}, // starts with xpub

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 0,
}

type btcAmountType = btcutil.Amount
type wireTxInType = wire.TxIn
type wireTxOutType = wire.TxOut

func isValidValue(value btcAmountType) bool {
	return value > 0 && value <= btcutil.MaxSatoshi
}

func newAmount(value float64) (btcAmountType, error) {
	return btcutil.NewAmount(value)
}

// GetChainParams get chain config (net params)
func (b *Bridge) GetChainParams() *chaincfg.Params {
	var chainParams *chaincfg.Params
	networkID := strings.ToLower(b.ChainConfig.NetID)
	switch networkID {
	case "mainnet":
		chainParams = &MainNetParams
	default:
		chainParams = &MainNetParams
	}
	return chainParams
}

// ParsePkScript parse pkScript
func (b *Bridge) ParsePkScript(pkScript []byte) (txscript.PkScript, error) {
	return txscript.ParsePkScript(pkScript)
}

// GetPayToAddrScript get pay to address script
func (b *Bridge) GetPayToAddrScript(address string) ([]byte, error) {
	toAddr, err := b.DecodeAddress(address)
	if err != nil {
		return nil, fmt.Errorf("decode btc address '%v' failed. %w", address, err)
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
	txin := wire.NewTxIn(prevOutPoint, pkScript, nil)
	return txin, nil
}

// NewTxOut new txout
func (b *Bridge) NewTxOut(amount int64, pkScript []byte) *wire.TxOut {
	txout := wire.NewTxOut(amount, pkScript)
	return txout
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
