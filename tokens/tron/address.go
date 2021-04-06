package tron

import (
	"encoding/hex"
	"strings"

	tronaddress "github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
)

// IsValidAddress check address
func (b *Bridge) IsValidAddress(address string) bool {
	_, err := tronaddress.Base58ToAddress(address)
	return err == nil
}

// PublicKeyToAddress returns cosmos public key address
func (b *Bridge) PublicKeyToAddress(pubKeyHex string) (address string, err error) {
	pubKeyHex = strings.TrimPrefix(pubKeyHex, "0x")
	bz, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return "", err
	}
	address = tronaddress.Address(append([]byte{0x41}, bz[len(bz)-20:]...)).String()
	return
}

func ethToTron(ethAddress string) (string, error) {
	bz, _ := common.FromHex(ethAddress)
	tronaddr := tronaddress.Address(append([]byte{0x41}, bz...))
	return tronaddr.String(), nil
}

func tronToEth(tronAddress string) (string, error) {
	addr, err := tronaddress.Base58ToAddress(tronAddress)
	if err != nil {
		return "", err
	}
	ethaddr := tronaddress.Address(addr.Bytes())
	return ethaddr.String(), nil
}
