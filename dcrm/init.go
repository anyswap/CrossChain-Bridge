package dcrm

import (
	"math/big"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/tools"
	"github.com/fsn-dev/crossChain-Bridge/tools/keystore"
	"github.com/fsn-dev/crossChain-Bridge/types"
)

const (
	DCRM_TO_ADDR           = "0x00000000000000000000000000000000000000dc"
	DCRM_WALLET_SERVICE_ID = 30400
)

var (
	Signer     = types.MakeSigner("EIP155", big.NewInt(DCRM_WALLET_SERVICE_ID))
	DcrmToAddr = common.HexToAddress(DCRM_TO_ADDR)
	SignGroups []string // sub groups for sign

	keyWrapper     *keystore.Key
	dcrmRpcAddress string

	signPubkey string
	groupID    string
	threshold  string
	mode       string
)

func SetDcrmRpcAddress(url string) {
	dcrmRpcAddress = url
}

func SetSignPubkey(pubkey string) {
	signPubkey = pubkey
}

func SetDcrmGroup(group_ string, threshold_ string, mode_ string) {
	groupID = group_
	threshold = threshold_
	mode = mode_
}

func SetSignGroups(signGroups []string) {
	SignGroups = signGroups
}

func LoadKeyStore(keyfile, passfile string) error {
	key, err := tools.LoadKeyStore(keyfile, passfile)
	if err != nil {
		return err
	}
	keyWrapper = key
	return nil
}
