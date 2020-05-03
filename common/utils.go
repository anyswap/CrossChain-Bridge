package common

import (
	"errors"
	"math/big"
	"strconv"
	"time"

	cmath "github.com/fsn-dev/crossChain-Bridge/common/math"
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
	bi, ok := cmath.ParseBig256(str)
	if !ok {
		return nil, errors.New("invalid 256 bit integer: " + str)
	}
	return bi, nil
}

func GetIntFromStr(str string) (int, error) {
	res, err := cmath.ParseInt(str)
	if err != nil {
		return 0, errors.New("invalid signed integer: " + str)
	}
	return res, nil
}

func GetUint64FromStr(str string) (uint64, error) {
	res, ok := cmath.ParseUint64(str)
	if !ok {
		return 0, errors.New("invalid unsigned 64 bit integer: " + str)
	}
	return res, nil
}

func Now() int64 {
	return time.Now().Unix()
}

func NowStr() string {
	return strconv.FormatInt((time.Now().Unix()), 10)
}

func NowMilli() int64 {
	return time.Now().UnixNano() / 1e6
}

func NowMilliStr() string {
	return strconv.FormatInt((time.Now().UnixNano() / 1e6), 10)
}
