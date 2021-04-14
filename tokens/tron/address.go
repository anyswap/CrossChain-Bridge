package tron

import (
	"fmt"
	"encoding/hex"
	"strings"

	tronaddress "github.com/fbsobreira/gotron-sdk/pkg/address"
	troncommon "github.com/fbsobreira/gotron-sdk/pkg/common"

	"github.com/anyswap/CrossChain-Bridge/tools/crypto"
	"github.com/anyswap/CrossChain-Bridge/common"
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
	ethaddr1, err1 := tronToEth(addr1)
	if err1 == nil {
		addr1 = ethaddr1
	}
	ethaddr2, err2 := tronToEth(addr2)
	if err2 == nil {
		addr2 = ethaddr2
	}
	bz1, _ := troncommon.FromHex(addr1)
	bz2, _ := troncommon.FromHex(addr2)
	return fmt.Sprintf("%X", bz1) == fmt.Sprintf("%X", bz2)
}

func ethToTron(ethAddress string) (string, error) {
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