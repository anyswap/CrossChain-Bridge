package ltc

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	bchaincfg "github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/ltcsuite/ltcutil"
	"github.com/ltcsuite/ltcutil/base58"
	"github.com/ltcsuite/ltcutil/bech32"
)

// ConvertBTCAddress decode btc address and convert to LTC address
// nolint:gocyclo // keep it
func (b *Bridge) ConvertBTCAddress(addr, btcNet string) (address ltcutil.Address, err error) {
	var bchainConfig *bchaincfg.Params
	switch btcNet {
	case "Main":
		bchainConfig = &bchaincfg.MainNetParams
	case "Test":
		bchainConfig = &bchaincfg.TestNet3Params
	default:
		bchainConfig = &bchaincfg.MainNetParams
	}
	lchainConfig := b.GetChainParams()
	// Bech32 encoded segwit addresses start with a human-readable part
	// (hrp) followed by '1'. For Bitcoin mainnet the hrp is "bc", and for
	// testnet it is "tb". If the address string has a prefix that matches
	// one of the prefixes for the known networks, we try to decode it as
	// a segwit address.
	oneIndex := strings.LastIndexByte(addr, '1')
	if oneIndex > 1 {
		prefix := addr[:oneIndex+1]
		if bchaincfg.IsBech32SegwitPrefix(prefix) {
			witnessVer, witnessProg, errf := decodeBTCSegWitAddress(addr)
			if errf != nil {
				return nil, errf
			}

			// We currently only support P2WPKH and P2WSH, which is
			// witness version 0.
			if witnessVer != 0 {
				return nil, btcutil.UnsupportedWitnessVerError(witnessVer)
			}

			switch len(witnessProg) {
			case 20:
				return ltcutil.NewAddressWitnessPubKeyHash(witnessProg, lchainConfig)
			case 32:
				return ltcutil.NewAddressWitnessScriptHash(witnessProg, lchainConfig)
			default:
				return nil, btcutil.UnsupportedWitnessProgLenError(len(witnessProg))
			}
		}
	}

	// Serialized public keys are either 65 bytes (130 hex chars) if
	// uncompressed/hybrid or 33 bytes (66 hex chars) if compressed.
	if len(addr) == 130 || len(addr) == 66 {
		serializedPubKey, errf := hex.DecodeString(addr)
		if errf != nil {
			return nil, errf
		}
		return ltcutil.NewAddressPubKey(serializedPubKey, lchainConfig)
	}

	// Switch on decoded length to determine the type.
	decoded, netID, err := base58.CheckDecode(addr)
	if err != nil {
		if errors.Is(err, base58.ErrChecksum) {
			return nil, btcutil.ErrChecksumMismatch
		}
		return nil, errors.New("decoded address is of unknown format")
	}
	switch len(decoded) {
	case 20: // P2PKH or P2SH
		isP2PKH := netID == bchainConfig.PubKeyHashAddrID
		isP2SH := netID == bchainConfig.ScriptHashAddrID
		switch hash160 := decoded; {
		case isP2PKH && isP2SH:
			return nil, btcutil.ErrAddressCollision
		case isP2PKH:
			return ltcutil.NewAddressPubKeyHash(hash160, lchainConfig)
		case isP2SH:
			return ltcutil.NewAddressScriptHashFromHash(hash160, lchainConfig)
		default:
			return nil, btcutil.ErrUnknownAddressType
		}

	default:
		return nil, errors.New("decoded address is of unknown size")
	}
}

// decodeSegWitAddress parses a bech32 encoded segwit address string and
// returns the witness version and witness program byte representation.
// nolint:dupl // keep it
func decodeBTCSegWitAddress(address string) (version byte, regrouped []byte, err error) {
	// Decode the bech32 encoded address.
	_, data, err := bech32.Decode(address)
	if err != nil {
		return 0, nil, err
	}

	// The first byte of the decoded address is the witness version, it must
	// exist.
	if len(data) < 1 {
		return 0, nil, fmt.Errorf("no witness version")
	}

	// ...and be <= 16.
	version = data[0]
	if version > 16 {
		return 0, nil, fmt.Errorf("invalid witness version: %v", version)
	}

	// The remaining characters of the address returned are grouped into
	// words of 5 bits. In order to restore the original witness program
	// bytes, we'll need to regroup into 8 bit words.
	regrouped, err = bech32.ConvertBits(data[1:], 5, 8, false)
	if err != nil {
		return 0, nil, err
	}

	// The regrouped data must be between 2 and 40 bytes.
	if len(regrouped) < 2 || len(regrouped) > 40 {
		return 0, nil, fmt.Errorf("invalid data length")
	}

	// For witness version 0, address MUST be exactly 20 or 32 bytes.
	if version == 0 && len(regrouped) != 20 && len(regrouped) != 32 {
		return 0, nil, fmt.Errorf("invalid data length for witness "+
			"version 0: %v", len(regrouped))
	}

	return version, regrouped, nil
}
