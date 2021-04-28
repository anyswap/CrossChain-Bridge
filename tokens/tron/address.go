package tron

import (
	"encoding/hex"
	"math/big"
	"strings"

	tronaddress "github.com/fbsobreira/gotron-sdk/pkg/address"
	troncommon "github.com/fbsobreira/gotron-sdk/pkg/common"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/tools/crypto"
)

// IsValidAddress check address
func (b *Bridge) IsValidAddress(address string) bool {
	if common.IsHexAddress(address) {
		_, err := ethToTron(address)
		return err == nil
	}
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
	ecdsaPub, err := crypto.UnmarshalPubkey(bz)
	if err != nil {
		return "", err
	}
	ethAddress := crypto.PubkeyToAddress(*ecdsaPub)
	address = tronaddress.Address(append([]byte{0x41}, ethAddress.Bytes()...)).String()
	return
}

func EqualAddress(addr1, addr2 string) bool {
	addr1 = anyToEth(addr1)
	addr2 = anyToEth(addr2)
	return strings.EqualFold(addr1, addr2)
}

func ethToTron(ethAddress string) (string, error) {
	intaddr, ok := new(big.Int).SetString(ethAddress, 16)
	if ok {
		ethAddress = common.BigToAddress(intaddr).String()
	}
	bz, _ := troncommon.FromHex(ethAddress)
	tronaddr := tronaddress.Address(append([]byte{0x41}, bz...))
	return tronaddr.String(), nil
}

func tronToEth(tronAddress string) (string, error) {
	addr, err := tronaddress.Base58ToAddress(tronAddress)
	if err != nil || len(addr) == 0 {
		return "", err
	}
	ethaddr := common.BytesToAddress(addr.Bytes())
	return ethaddr.String(), nil
}

func anyToTron(address string) string {
	addr, err := tronaddress.Base58ToAddress(address)
	if err != nil {
		address, err = ethToTron(address)
		if err != nil {
			return ""
		}
	} else {
		address = addr.String()
	}
	return address
}

func anyToEth(address string) string {
	tronaddr := anyToTron(address)
	address, _ = tronToEth(tronaddr)
	return address
}
