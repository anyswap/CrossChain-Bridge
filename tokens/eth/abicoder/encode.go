// Package abicoder is simple tool to pack datas like solidity abi.
package abicoder

import (
	"math/big"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/common/hexutil"
	"github.com/anyswap/CrossChain-Bridge/log"
)

// PackDataWithFuncHash pack data with func hash
func PackDataWithFuncHash(funcHash []byte, args ...interface{}) []byte {
	packData := PackData(args...)

	bs := make([]byte, 4+len(packData))

	copy(bs[:4], funcHash)
	copy(bs[4:], packData)

	return bs
}

// PackData pack data
// nolint:gocyclo,makezero // allow big switch
func PackData(args ...interface{}) []byte {
	lenArgs := len(args)
	bs := make([]byte, lenArgs*32)
	for i, arg := range args {
		switch v := arg.(type) {
		case common.Hash:
			copy(bs[i*32:], packHash(v))
		case common.Address:
			copy(bs[i*32:], packAddress(v))
		case *big.Int:
			copy(bs[i*32:(i+1)*32], packBigInt(v))
		case string:
			offset := big.NewInt(int64(len(bs)))
			copy(bs[i*32:], packBigInt(offset))
			bs = append(bs, packString(v)...)
		case []byte:
			offset := big.NewInt(int64(len(bs)))
			copy(bs[i*32:], packBigInt(offset))
			bs = append(bs, packBytes(v)...)
		case hexutil.Bytes:
			offset := big.NewInt(int64(len(bs)))
			copy(bs[i*32:], packBigInt(offset))
			bs = append(bs, packBytes(v)...)
		case uint64:
			copy(bs[i*32:], packBigInt(new(big.Int).SetUint64(v)))
		case int64:
			copy(bs[i*32:], packBigInt(big.NewInt(v)))
		case int:
			copy(bs[i*32:], packBigInt(big.NewInt(int64(v))))
		case uint8:
			copy(bs[i*32:], packBigInt(big.NewInt(int64(v))))
		case []common.Address:
			offset := big.NewInt(int64(len(bs)))
			copy(bs[i*32:], packBigInt(offset))
			bs = append(bs, packAddressSlice(v)...)
		case []*big.Int:
			offset := big.NewInt(int64(len(bs)))
			copy(bs[i*32:], packBigInt(offset))
			bs = append(bs, packBigIntSlice(v)...)
		case []string:
			offset := big.NewInt(int64(len(bs)))
			copy(bs[i*32:], packBigInt(offset))
			bs = append(bs, packStringSlice(v)...)
		case []hexutil.Bytes:
			offset := big.NewInt(int64(len(bs)))
			copy(bs[i*32:], packBigInt(offset))
			bs = append(bs, packHexBytesSlice(v)...)
		case [][]byte:
			offset := big.NewInt(int64(len(bs)))
			copy(bs[i*32:], packBigInt(offset))
			bs = append(bs, packBytesSlice(v)...)
		default:
			log.Fatalf("unsupported to pack %v (%T)", v, v)
		}
	}
	return bs
}

func packHash(hash common.Hash) []byte {
	return hash.Bytes()
}

func packAddress(address common.Address) []byte {
	return address.Hash().Bytes()
}

func packBigInt(bi *big.Int) []byte {
	var bs []byte
	if bi != nil {
		bs = bi.Bytes()
	}
	return common.LeftPadBytes(bs, 32)
}

func packString(str string) []byte {
	strLen := len(str)
	paddedStrLen := (strLen + 31) / 32 * 32

	bs := make([]byte, 32+paddedStrLen)

	copy(bs[:32], packBigInt(big.NewInt(int64(strLen))))
	copy(bs[32:], str)

	return bs
}

func packBytes(data []byte) []byte {
	bsLen := len(data)
	paddedLen := (bsLen + 31) / 32 * 32

	bs := make([]byte, 32+paddedLen)

	copy(bs[:32], packBigInt(big.NewInt(int64(bsLen))))
	copy(bs[32:], data)

	return bs
}

func packAddressSlice(addrs []common.Address) []byte {
	length := len(addrs)
	bs := make([]byte, (1+length)*32)
	copy(bs[:32], packBigInt(big.NewInt(int64(length))))
	for i, addr := range addrs {
		copy(bs[(i+1)*32:], addr.Hash().Bytes())
	}
	return bs
}

// nolint:makezero // keep it
func packStringSlice(strSlice []string) []byte {
	length := len(strSlice)
	bsLen := packBigInt(big.NewInt(int64(length)))
	bsInner := make([]byte, length*32)
	for i, str := range strSlice {
		copy(bsInner[i*32:], packBigInt(big.NewInt(int64(len(bsInner)))))
		bsInner = append(bsInner, packString(str)...)
	}
	return append(bsLen, bsInner...)
}

func packBigIntSlice(biSlice []*big.Int) []byte {
	length := len(biSlice)
	bs := make([]byte, (1+length)*32)
	copy(bs[:32], packBigInt(big.NewInt(int64(length))))
	for i, bi := range biSlice {
		copy(bs[(i+1)*32:], packBigInt(bi))
	}
	return bs
}

// nolint:makezero // keep it
func packHexBytesSlice(bsSlice []hexutil.Bytes) []byte {
	length := len(bsSlice)
	bsLen := packBigInt(big.NewInt(int64(length)))
	bsInner := make([]byte, length*32)
	for i, bs := range bsSlice {
		copy(bsInner[i*32:], packBigInt(big.NewInt(int64(len(bsInner)))))
		bsInner = append(bsInner, packBytes(bs)...)
	}
	return append(bsLen, bsInner...)
}

// nolint:makezero // keep it
func packBytesSlice(bsSlice [][]byte) []byte {
	length := len(bsSlice)
	bsLen := packBigInt(big.NewInt(int64(length)))
	bsInner := make([]byte, length*32)
	for i, bs := range bsSlice {
		copy(bsInner[i*32:], packBigInt(big.NewInt(int64(len(bsInner)))))
		bsInner = append(bsInner, packBytes(bs)...)
	}
	return append(bsLen, bsInner...)
}
