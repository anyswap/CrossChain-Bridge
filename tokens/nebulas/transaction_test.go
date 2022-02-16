package nebulas

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransactionHash(t *testing.T) {

	rawData := "EhoZV9/fkkcxT4S2Xvm4GfkPES9ZAM+8U65f2xoaGViw5FyyWM2OspproxUg2y0MdkM8gwVs2PIiEAAAAAAAAAAAAAAAAAAAAAAoATCg9LKQBjppCgRjYWxsEmF7ImZ1bmN0aW9uIjoidHJhbnNmZXIiLCJhcmdzIjoiW1wibjFTUW0yM3lMYktuMXVFRHNwV0pFZGhZMndoc2gxc3o1akpcIixcIjk5ODAwMDAwMDAwMDAwMDAwMFwiXSJ9QOkHShAAAAAAAAAAAAAAAAUfTVwAUhAAAAAAAAAAAAAAAAAAAV+QWAFiQYnoTezWO0NK0vHTCLobkiypWrRJGaZ3YqNLJR4bAu8vLdEmrYexv31lF3Z7fS+ysi6N9drcBSvl96I1+g3o7YUA"
	bytes, err := base64.StdEncoding.DecodeString(rawData)
	assert.Nil(t, err)
	tx := &Transaction{}
	err = tx.FromBytes(bytes)
	assert.Nil(t, err)
	calHash, err := tx.HashTransaction()
	assert.Nil(t, err)
	assert.Equal(t, calHash, tx.hash)
}
