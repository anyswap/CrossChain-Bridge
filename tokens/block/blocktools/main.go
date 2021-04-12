package main

import (
	"fmt"
	"log"
	"math"
	"math/big"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/anyswap/CrossChain-Bridge/common"
)

func main() {
	TestNewAddressPubKeyHash()
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func TestNewAddressPubKeyHash() {
	fmt.Println("TestNewAddressPubKeyHash")

	/*wif := "PnPaj8UeZCziJx9EBTDsZfzuYDZVaGwvrtettNeTqnhH5Z3d6B41"
	privWif, err := btcutil.DecodeWIF(wif)
	checkError(t, err)

	pkdata := privWif.SerializePubKey()*/

	pkData := common.FromHex("04d38309dfdfd9adf129287b68cf2e1f1124e0cbc40cc98f94e5f2d23c26712fa3b33d63280dd1448319a6a4f4111722d6b3a730ebe07652ed2b3770947b3de2e2")
	cPkData, err := ToCompressedPublicKey(pkData)
	checkError(err)

	addr, err := NewAddressPubKeyHash(cPkData)
	checkError(err)
	fmt.Printf("NewAddressPubKeyHash: %s\n", addr)

	/*realaddr := "BXcz95EZfLBREpQrMDsKFnMJSaUYNRyhHU"

	if addr.String() != realaddr {
		err := fmt.Errorf("Block address error, got %s, should be: %s", addr.String(), realaddr)
		checkError(err)
	}
	*/
}

func NewAddressPubKeyHash(pkData []byte) (*btcutil.AddressPubKeyHash, error) {
	return btcutil.NewAddressPubKeyHash(btcutil.Hash160(pkData), &MainNetParams)
}

var bigOne = big.NewInt(1)

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

func ToCompressedPublicKey(pkData []byte) ([]byte, error) {
	pubKey, err := btcec.ParsePubKey(pkData, btcec.S256())
	if err != nil {
		return nil, err
	}
	return pubKey.SerializeCompressed(), nil
}