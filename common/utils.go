package common

import (
	"encoding/json"
	"errors"
	"math/big"
	"strconv"
	"strings"
	"time"

	cmath "github.com/anyswap/CrossChain-Bridge/common/math"
	"golang.org/x/crypto/sha3"
)

// ToJSONString to json string
func ToJSONString(content interface{}, pretty bool) string {
	var data []byte
	if pretty {
		data, _ = json.MarshalIndent(content, "", "  ")
	} else {
		data, _ = json.Marshal(content)
	}
	return string(data)
}

// Keccak256Hash calc keccak hash.
func Keccak256Hash(data ...[]byte) (h Hash) {
	d := sha3.NewLegacyKeccak256()
	for _, b := range data {
		_, _ = d.Write(b)
	}
	d.Sum(h[:0])
	return h
}

// IsEqualIgnoreCase returns if s1 and s2 are equal ignore case.
func IsEqualIgnoreCase(s1, s2 string) bool {
	return strings.EqualFold(s1, s2)
}

// BigFromUint64 new big int from uint64 value.
func BigFromUint64(value uint64) *big.Int {
	return new(big.Int).SetUint64(value)
}

// GetBigIntFromStr new big int from string.
func GetBigIntFromStr(str string) (*big.Int, error) {
	bi, ok := cmath.ParseBig256(str)
	if !ok {
		return nil, errors.New("invalid 256 bit integer: " + str)
	}
	return bi, nil
}

// GetIntFromStr get int from string.
func GetIntFromStr(str string) (int, error) {
	res, err := cmath.ParseInt(str)
	if err != nil {
		return 0, errors.New("invalid signed integer: " + str)
	}
	return res, nil
}

// GetUint64FromStr get uint64 from string.
func GetUint64FromStr(str string) (uint64, error) {
	res, ok := cmath.ParseUint64(str)
	if !ok {
		return 0, errors.New("invalid unsigned 64 bit integer: " + str)
	}
	return res, nil
}

// Now returns timestamp of the point of calling.
func Now() int64 {
	return time.Now().Unix()
}

// NowStr returns now timestamp of string format.
func NowStr() string {
	return strconv.FormatInt((time.Now().Unix()), 10)
}

// NowMilli returns now timestamp in miliseconds
func NowMilli() int64 {
	return time.Now().UnixNano() / 1e6
}

// NowMilliStr returns now timestamp in miliseconds of string format.
func NowMilliStr() string {
	return strconv.FormatInt((time.Now().UnixNano() / 1e6), 10)
}

// MinUint64 get minimum value of x and y
func MinUint64(x, y uint64) uint64 {
	if x <= y {
		return x
	}
	return y
}

// MaxUint64 get maximum calue of x and y.
func MaxUint64(x, y uint64) uint64 {
	if x < y {
		return y
	}
	return x
}

// GetData get data[start:start+size] (won't out of index range),
// and right padding the bytes to size long
func GetData(data []byte, start, size uint64) []byte {
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

// BigUint64 big to uint64 and an overflow flag.
func BigUint64(v *big.Int) (uint64, bool) {
	return v.Uint64(), !v.IsUint64()
}

// GetBigInt get big int from data[start:start+size]
func GetBigInt(data []byte, start, size uint64) *big.Int {
	length := uint64(len(data))
	if length <= start || size == 0 {
		return big.NewInt(0)
	}
	end := start + size
	if end > length {
		end = length
	}
	return new(big.Int).SetBytes(data[start:end])
}

// GetUint64 get uint64 from data[start:start+size]
func GetUint64(data []byte, start, size uint64) (uint64, bool) {
	return BigUint64(GetBigInt(data, start, size))
}
