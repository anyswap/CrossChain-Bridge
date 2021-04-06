package tron

import (
	"encoding/hex"

	tronaddress "github.com/fbsobreira/gotron-sdk/pkg/address"
)

// IsValidAddress check address
func (b *Bridge) IsValidAddress(address string) bool {
	_, err := tronaddress.Base58ToAddress(tronAddress)
	return err == nil
}

// PublicKeyToAddress returns cosmos public key address
func (b *Bridge) PublicKeyToAddress(pubKeyHex string) (address string, err error) {
	pubKeyHex = strings.TrimPrefix(pubKeyHex, "0x")
	bz, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return "", err
	}
	address = tronaddress.Address(append([]byte{0x41}, bz[len(bz)-20:]...))
	return
}

func ethToTron(ethAddress string) (string, error) {
	tronaddr := tronaddress.Address(append([]byte{0x41}, common.HexToAddress(ethAddress).Bytes()...))
	return tronaddr.String(), nil
}

func tronToEth(tronAddress string) (string, error) {
	addr, err := tronaddress.Base58ToAddress(tronAddress)
	if err != nil {
		return "", err
	}
	ethaddr := common.BytesToAddress(addr.Bytes())
	return ethaddr.String(), nil
}
