package terra

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"testing"

	"github.com/anyswap/CrossChain-Bridge/tokens/cosmos"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/tendermint/tendermint/crypto/tmhash"

	"github.com/stretchr/testify/assert"
)

var cdc *codec.Codec

func initCdc() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	cdc = codec.New()
	codec.RegisterCrypto(cdc)
	cosmos.RegisterCodec(cdc)
	cdc.RegisterConcrete(authtypes.StdTx{}, "auth/StdTx", nil)
	cdc.RegisterInterface((*sdk.Msg)(nil), nil)
}

func TestTxHash(t *testing.T) {
	initCdc()

	rawtx := "5wHwYl3uCkaoo2GaChQmSIu8hxpJxLcCuIi8fiHN4TMwrRIU/Af1cEG7Rcs/6LjTl7YjRSymJfYaFAoFdWF0b20SCzE0OTk5OTk1MDAwEhMKDQoFdWF0b20SBDUwMDAQwJoMGmoKJuta6YchAwswBShaB1wkZBctLIhYqBC3JrAI28XGzxP+rVEticGEEkAc+khTkKL9CDE47aDvjEHvUNt+izJfT4KVF2v2JkC+bmlH9K08q3PqHeMI9Z5up+XMusnTqlP985KF+SI5J3ZOIhhNYWRlIGJ5IENpcmNsZSB3aXRoIGxvdmU="
	wantHash := "D70952032620CC4E2737EB8AC379806359D8E0B17B0488F627997A0B043ABDED"

	log.Println("rawtx is", rawtx)
	log.Println("want hash is", wantHash)

	txBytes, err := base64.StdEncoding.DecodeString(rawtx)
	assert.Nil(t, err, "base64 decode tx error")

	txHash := fmt.Sprintf("%X", tmhash.Sum(txBytes))
	log.Println("calc hash is", txHash)

	assert.Equal(t, txHash, wantHash)

	tx := authtypes.StdTx{}
	err = cdc.UnmarshalBinaryLengthPrefixed(txBytes, &tx)
	assert.Nil(t, err, "cdc tx decode error")

	js, err := json.Marshal(tx)
	assert.Nil(t, err, "json marshal tx error")
	log.Println("json unmarshal success", string(js))

	txbs, err := cdc.MarshalBinaryLengthPrefixed(tx)
	assert.Nil(t, err, "cdc marshal tx error")

	assert.True(t, bytes.Equal(txBytes, txbs))
}
