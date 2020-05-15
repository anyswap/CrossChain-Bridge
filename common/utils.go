package common

import (
	"errors"
	"math/big"
	"strconv"
	"strings"
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

func IsEqualIgnoreCase(s1, s2 string) bool {
	return strings.ToLower(s1) == strings.ToLower(s2)
}

func BigFromUint64(value uint64) *big.Int {
	return new(big.Int).SetUint64(value)
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

func MinUint64(x, y uint64) uint64 {
	if x <= y {
		return x
	}
	return y
}

func MaxUint64(x, y uint64) uint64 {
	if x < y {
		return y
	}
	return x
}

func GetData(data []byte, start uint64, size uint64) []byte {
	length := uint64(len(data))
	if start > length {
		start = length
	}
	end := start + size
	if end > length {
		end = length
	}
	return RightPadBytes(data[start:end], int(size))
}

func BigUint64(v *big.Int) (uint64, bool) {
	return v.Uint64(), !v.IsUint64()
}

func GetBigInt(data []byte, start uint64, size uint64) *big.Int {
	return new(big.Int).SetBytes(GetData(data, start, size))
}

func GetUint64(data []byte, start uint64, size uint64) (uint64, bool) {
	return BigUint64(GetBigInt(data, start, size))
}
