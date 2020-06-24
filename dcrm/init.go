package dcrm

import (
	"math/big"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/tools"
	"github.com/anyswap/CrossChain-Bridge/tools/keystore"
	"github.com/anyswap/CrossChain-Bridge/types"
)

const (
	// DcrmToAddress used in dcrm sign and accept
	DcrmToAddress = "0x00000000000000000000000000000000000000dc"
	// DcrmWalletServiceID to make dcrm signer
	DcrmWalletServiceID = 30400
)

var (
	dcrmSigner = types.MakeSigner("EIP155", big.NewInt(DcrmWalletServiceID))
	dcrmToAddr = common.HexToAddress(DcrmToAddress)
	signGroups []string // sub groups for sign

	keyWrapper     *keystore.Key
	dcrmUser       common.Address
	dcrmRPCAddress string

	signPubkey string
	groupID    string
	threshold  string
	mode       string

	// ServerDcrmUser dcrm initiator for sign
	ServerDcrmUser common.Address
)

// SetDcrmRPCAddress set dcrm node rpc address
func SetDcrmRPCAddress(url string) {
	dcrmRPCAddress = url
}

// SetSignPubkey set dcrm account public key
func SetSignPubkey(pubkey string) {
	signPubkey = pubkey
}

// SetDcrmGroup set dcrm group
func SetDcrmGroup(group, thresh, mod string) {
	groupID = group
	threshold = thresh
	mode = mod
}

// GetGroupID return dcrm group id
func GetGroupID() string {
	return groupID
}

// SetSignGroups set sign subgroups
func SetSignGroups(groups []string) {
	signGroups = groups
}

// GetSignGroups get sign subgroups
func GetSignGroups() []string {
	return signGroups
}

// LoadKeyStore load keystore
func LoadKeyStore(keyfile, passfile string) error {
	key, err := tools.LoadKeyStore(keyfile, passfile)
	if err != nil {
		return err
	}
	keyWrapper = key
	dcrmUser = keyWrapper.Address
	return nil
}

// GetDcrmUser returns the dcrm user of specified keystore
func GetDcrmUser() common.Address {
	return dcrmUser
}

// IsSwapServer returns if this dcrm user is the swap server
func IsSwapServer() bool {
	return GetDcrmUser() == ServerDcrmUser
}
