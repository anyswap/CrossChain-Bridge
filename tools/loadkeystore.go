package tools

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/fsn-dev/crossChain-Bridge/tools/keystore"
)

// LoadKeyStore load keystore from keyfile and passfile
func LoadKeyStore(keyfile, passfile string) (*keystore.Key, error) {
	keyjson, err := ioutil.ReadFile(keyfile)
	if err != nil {
		return nil, fmt.Errorf("Read keystore fail %v", err)
	}
	passdata, err := ioutil.ReadFile(passfile)
	if err != nil {
		return nil, fmt.Errorf("Read password fail %v", err)
	}
	passwd := strings.TrimSpace(string(passdata))
	key, err := keystore.DecryptKey(keyjson, passwd)
	if err != nil {
		return nil, fmt.Errorf("Decrypt key fail %v", err)
	}
	return key, nil
}
