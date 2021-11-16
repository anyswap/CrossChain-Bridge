package tools

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/tools/keystore"
)

var errUnsafeFilePermissions = errors.New("unsafe file permissions, want 0400")

// SafeReadFile check permissions is '0400' and read file
func SafeReadFile(file string) ([]byte, error) {
	fi, err := os.Stat(file)
	if err != nil {
		return nil, err
	}
	if fi.Mode() != 0400 {
		return nil, errUnsafeFilePermissions
	}
	return ioutil.ReadFile(file) // nolint:gosec // ok
}

// LoadKeyStore load keystore from keyfile and passfile
func LoadKeyStore(keyfile, passfile string) (*keystore.Key, error) {
	keyjson, err := SafeReadFile(keyfile)
	if err != nil {
		return nil, fmt.Errorf("read keystore fail %w", err)
	}
	passdata, err := SafeReadFile(passfile)
	if err != nil {
		return nil, fmt.Errorf("read password fail %w", err)
	}
	passwd := strings.TrimSpace(string(passdata))
	key, err := keystore.DecryptKey(keyjson, passwd)
	if err != nil {
		return nil, fmt.Errorf("decrypt key fail %w", err)
	}
	return key, nil
}
