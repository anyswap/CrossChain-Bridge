package common

import (
	"errors"
	"math/big"

	"golang.org/x/crypto/sha3"
)

func Keccak256Hash(data ...[]byte) (h Hash) {
	d := sha3.NewLegacyKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	d.Sum(h[:0])
	return h
}

func GetBigIntFromStr(str string) (*big.Int, error) {
	bi, ok := new(big.Int).SetString(str, 0)
	if !ok {
		return nil, errors.New("GetBigIntFromStr: wrong format")
	}
	return bi, nil
}

func GetIntFromStr(str string) (int, error) {
	bi, ok := new(big.Int).SetString(str, 0)
	if !ok || !bi.IsUint64() || bi.Uint64() > uint64(MaxInt) {
		return 0, errors.New("GetIntFromStr: wrong format")
	}
	return int(bi.Uint64()), nil
}

func GetUint64FromStr(str string) (uint64, error) {
	bi, ok := new(big.Int).SetString(str, 0)
	if !ok || !bi.IsUint64() {
		return 0, errors.New("GetUint64FromStr: wrong format")
	}
	return bi.Uint64(), nil
}

func GetInt64FromStr(str string) (int64, error) {
	bi, ok := new(big.Int).SetString(str, 0)
	if !ok || !bi.IsInt64() {
		return 0, errors.New("GetInt64FromStr: wrong format")
	}
	return bi.Int64(), nil
}
