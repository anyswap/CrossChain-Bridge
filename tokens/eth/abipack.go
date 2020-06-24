package eth

import (
	"fmt"
	"math/big"

	"github.com/anyswap/CrossChain-Bridge/common"
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
		case uint64:
			copy(bs[i*32:], packBigInt(new(big.Int).SetUint64(v)))
		case int64:
			copy(bs[i*32:], packBigInt(big.NewInt(v)))
		case int:
			copy(bs[i*32:], packBigInt(big.NewInt(int64(v))))
		default:
			panic(fmt.Sprintf("unsupported to pack %v (%T)", v, v))
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
	return common.LeftPadBytes(bi.Bytes(), 32)
}

func packString(str string) []byte {
	strLen := len(str)
	paddedStrLen := (strLen + 31) / 32 * 32

	bs := make([]byte, 32+paddedStrLen)

	copy(bs[:32], packBigInt(big.NewInt(int64(strLen))))
	copy(bs[32:], str)

	return bs
}
