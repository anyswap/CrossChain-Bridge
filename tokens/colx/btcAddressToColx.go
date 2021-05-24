package colx

import (
	"encoding/hex"
	"errors"

	bchaincfg "github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	colxutil "github.com/giangnamnabka/btcutil"
	"github.com/giangnamnabka/btcutil/base58"
)

// ConvertBTCAddress decode btc address and convert to COLX address
func (b *Bridge) ConvertBTCAddress(addr, btcNet string) (address colxutil.Address, err error) {
	var bchainConfig *bchaincfg.Params
	switch btcNet {
	case "Main":
		bchainConfig = &bchaincfg.MainNetParams
	case "Test":
		bchainConfig = &bchaincfg.TestNet3Params
	default:
		bchainConfig = &bchaincfg.MainNetParams
	}
	cchainConfig := b.GetChainParams()
	// Serialized public keys are either 65 bytes (130 hex chars) if
	// uncompressed/hybrid or 33 bytes (66 hex chars) if compressed.
	if len(addr) == 130 || len(addr) == 66 {
		serializedPubKey, errf := hex.DecodeString(addr)
		if errf != nil {
			return nil, errf
		}
		return colxutil.NewAddressPubKey(serializedPubKey, cchainConfig)
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
			return colxutil.NewAddressPubKeyHash(hash160, cchainConfig)
		case isP2SH:
			return colxutil.NewAddressScriptHashFromHash(hash160, cchainConfig)
		default:
			return nil, btcutil.ErrUnknownAddressType
		}

	default:
		return nil, errors.New("decoded address is of unknown size")
	}
}
