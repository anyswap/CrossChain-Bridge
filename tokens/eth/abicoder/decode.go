package abicoder

import (
	"errors"
	"math/big"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/common/hexutil"
)

// parse errors
var (
	ErrParseDataError = errors.New("parse data error")
)

func parseSliceInData(data []byte, pos uint64) (offset, length uint64, err error) {
	offset, overflow := common.GetUint64(data, pos, 32)
	if overflow {
		return 0, 0, ErrParseDataError
	}
	length, overflow = common.GetUint64(data, offset, 32)
	if overflow {
		return 0, 0, ErrParseDataError
	}
	offset += 32
	if uint64(len(data)) < offset+length*32 {
		return 0, 0, ErrParseDataError
	}
	return offset, length, nil
}

// ParseAddressSliceInData parse
func ParseAddressSliceInData(data []byte, pos uint64) ([]string, error) {
	offset, length, err := parseSliceInData(data, pos)
	if err != nil {
		return nil, err
	}
	path := make([]string, length)
	for i := uint64(0); i < length; i++ {
		path[i] = common.BytesToAddress(common.GetData(data, offset, 32)).LowerHex()
		offset += 32
	}
	return path, nil
}

// ParseAddressSliceAsAddressesInData parse
func ParseAddressSliceAsAddressesInData(data []byte, pos uint64) ([]common.Address, error) {
	offset, length, err := parseSliceInData(data, pos)
	if err != nil {
		return nil, err
	}
	path := make([]common.Address, length)
	for i := uint64(0); i < length; i++ {
		path[i] = common.BytesToAddress(common.GetData(data, offset, 32))
		offset += 32
	}
	return path, nil
}

// ParseNumberSliceInData parse
func ParseNumberSliceInData(data []byte, pos uint64) ([]string, error) {
	offset, length, err := parseSliceInData(data, pos)
	if err != nil {
		return nil, err
	}
	results := make([]string, length)
	for i := uint64(0); i < length; i++ {
		results[i] = common.GetBigInt(data, offset, 32).String()
		offset += 32
	}
	return results, nil
}

// ParseNumberSliceAsBigIntsInData parse
func ParseNumberSliceAsBigIntsInData(data []byte, pos uint64) ([]*big.Int, error) {
	offset, length, err := parseSliceInData(data, pos)
	if err != nil {
		return nil, err
	}
	results := make([]*big.Int, length)
	for i := uint64(0); i < length; i++ {
		results[i] = common.GetBigInt(data, offset, 32)
		offset += 32
	}
	return results, nil
}

// ParseStringSliceInData parse
func ParseStringSliceInData(data []byte, pos uint64) ([]string, error) {
	offset, length, err := parseSliceInData(data, pos)
	if err != nil {
		return nil, err
	}
	// new data for inner array
	data = data[offset:]
	offset = 0
	results := make([]string, length)
	for i := uint64(0); i < length; i++ {
		str, err := ParseStringInData(data, offset)
		if err != nil {
			return nil, err
		}
		results[i] = str
		offset += 32
	}
	return results, nil
}

// ParseStringInData parse
func ParseStringInData(data []byte, pos uint64) (string, error) {
	offset, overflow := common.GetUint64(data, pos, 32)
	if overflow {
		return "", ErrParseDataError
	}
	length, overflow := common.GetUint64(data, offset, 32)
	if overflow {
		return "", ErrParseDataError
	}
	if uint64(len(data)) < offset+32+length {
		return "", ErrParseDataError
	}
	return string(common.GetData(data, offset+32, length)), nil
}

// ParseBytesSliceInData parse
func ParseBytesSliceInData(data []byte, pos uint64) ([]hexutil.Bytes, error) {
	offset, length, err := parseSliceInData(data, pos)
	if err != nil {
		return nil, err
	}
	// new data for inner array
	data = data[offset:]
	offset = 0
	results := make([]hexutil.Bytes, length)
	for i := uint64(0); i < length; i++ {
		bs, err := ParseBytesInData(data, offset)
		if err != nil {
			return nil, err
		}
		results[i] = bs
		offset += 32
	}
	return results, nil
}

// ParseBytesInData parse
func ParseBytesInData(data []byte, pos uint64) (hexutil.Bytes, error) {
	offset, overflow := common.GetUint64(data, pos, 32)
	if overflow {
		return nil, ErrParseDataError
	}
	length, overflow := common.GetUint64(data, offset, 32)
	if overflow {
		return nil, ErrParseDataError
	}
	if uint64(len(data)) < offset+32+length {
		return nil, ErrParseDataError
	}
	return common.GetData(data, offset+32, length), nil
}
