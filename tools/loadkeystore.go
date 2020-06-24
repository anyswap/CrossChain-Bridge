package tools

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/tools/keystore"
)

// LoadKeyStore load keystore from keyfile and passfile
func LoadKeyStore(keyfile, passfile string) (*keystore.Key, error) {
	keyjson, err := ioutil.ReadFile(keyfile)
	if err != nil {
		return nil, fmt.Errorf("read keystore fail %v", err)
	}
	passdata, err := ioutil.ReadFile(passfile)
	if err != nil {
		return nil, fmt.Errorf("read password fail %v", err)
	}
	passwd := strings.TrimSpace(string(passdata))
	key, err := keystore.DecryptKey(keyjson, passwd)
	if err != nil {
		return nil, fmt.Errorf("decrypt key fail %v", err)
	}
	return key, nil
}
