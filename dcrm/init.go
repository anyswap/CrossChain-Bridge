package dcrm

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"strings"

	"github.com/fsn-dev/crossChain-Bridge/common"
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
	keyjson, err := ioutil.ReadFile(keyfile)
	if err != nil {
		return fmt.Errorf("Read keystore fail %v", err)
	}
	passdata, err := ioutil.ReadFile(passfile)
	if err != nil {
		return fmt.Errorf("Read password fail %v", err)
	}
	passwd := strings.TrimSpace(string(passdata))
	key, err := keystore.DecryptKey(keyjson, passwd)
	if err != nil {
		return fmt.Errorf("Decrypt key fail %v", err)
	}
	keyWrapper = key
	return nil
}
