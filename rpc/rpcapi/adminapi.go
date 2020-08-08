package rpcapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tools/crypto"
)

const (
	blacklistMethod = "blacklist"

	successReuslt = "success"
)

// AdminCallArg call args
type AdminCallArg struct {
	Method    string   `json:"method"`
	Params    []string `json:"params"`
	Timestamp int64    `json:"timestamp"`
	Signature []byte   `json:"-"`
}

func verifySignature(args *AdminCallArg) error {
	data, _ := json.Marshal(args)
	signature := args.Signature
	pubkey, err := crypto.Ecrecover(common.Keccak256Hash(data).Bytes(), signature)
	if err != nil {
		return err
	}
	var signer common.Address
	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])
	if !params.IsAdmin(signer.String()) {
		return fmt.Errorf("signer %v is not admin", signer.String())
	}
	return nil
}

// AdminCall admin call
func (s *RPCAPI) AdminCall(r *http.Request, args *AdminCallArg, result *string) (err error) {
	err = verifySignature(args)
	if err != nil {
		return err
	}
	switch args.Method {
	case blacklistMethod:
		return blacklist(args, result)
	default:
		return fmt.Errorf("unknown admin method '%v'", args.Method)
	}
}

func blacklist(args *AdminCallArg, result *string) (err error) {
	if len(args.Params) != 2 {
		return fmt.Errorf("wrong number of params, have %v want 2", len(args.Params))
	}
	operation := args.Params[0]
	address := args.Params[1]
	isBlacked := false
	isQuery := false
	switch operation {
	case "add":
		err = mongodb.AddToBlacklist(address)
	case "remove":
		err = mongodb.RemoveFromBlacklist(address)
	case "query":
		isQuery = true
		isBlacked, err = mongodb.QueryBlacklist(address)
	default:
		return fmt.Errorf("unknown operation '%v'", operation)
	}
	if err == nil {
		if isQuery {
			if isBlacked {
				*result = "is in blacklist"
			} else {
				*result = "is not in blacklist"
			}
		} else {
			*result = successReuslt
		}
	} else {
		*result = err.Error()
	}
	return err
}
