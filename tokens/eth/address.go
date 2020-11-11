package eth

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/tools/crypto"
)

// IsValidAddress check address
func (b *Bridge) IsValidAddress(address string) bool {
	if !common.IsHexAddress(address) {
		return false
	}
	unprefixedHex, ok, hasUpperChar := common.GetUnprefixedHex(address)
	if hasUpperChar {
		// valid checksum
		if unprefixedHex != common.HexToAddress(address).String()[2:] {
			return false
		}
	}
	return ok
}

// IsContractAddress is contract address
func (b *Bridge) IsContractAddress(address string) (bool, error) {
	var code []byte
	var err error
	for i := 0; i < retryRPCCount; i++ {
		code, err = b.GetCode(address)
		if err == nil {
			return len(code) != 0, nil
		}
		time.Sleep(retryRPCInterval)
	}
	return false, err
}

// GetBip32InputCode get bip32 input code
func (b *Bridge) GetBip32InputCode(addr string) (string, error) {
	if !b.IsValidAddress(addr) {
		return "", fmt.Errorf("invalid address")
	}
	address := common.HexToAddress(addr)
	index := new(big.Int).SetBytes(address.Bytes())
	index.Add(index, common.BigPow(2, 31))
	return fmt.Sprintf("m/%s", index.String()), nil
}

// PublicKeyToAddress public key to address
func (b *Bridge) PublicKeyToAddress(hexPubkey string) (string, error) {
	pkData := common.FromHex(hexPubkey)
	if len(pkData) != 65 {
		return "", fmt.Errorf("wrong length of public key")
	}
	if pkData[0] != 4 {
		return "", fmt.Errorf("wrong public key, shoule be uncompressed")
	}
	pkData = pkData[1:]
	ecPub := ecdsa.PublicKey{
		Curve: crypto.S256(),
		X:     new(big.Int).SetBytes(pkData[:32]),
		Y:     new(big.Int).SetBytes(pkData[32:]),
	}
	if !ecPub.Curve.IsOnCurve(ecPub.X, ecPub.Y) {
		return "", fmt.Errorf("invalid secp256k1 curve point")
	}
	return crypto.PubkeyToAddress(ecPub).String(), nil
}
