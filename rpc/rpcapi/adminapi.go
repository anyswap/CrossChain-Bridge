package rpcapi

import (
	"fmt"
	"net/http"

	"github.com/anyswap/CrossChain-Bridge/admin"
	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

const (
	successReuslt = "Success"
	swapinOp      = "swapin"
	swapoutOp     = "swapout"
)

// AdminCall admin call
func (s *RPCAPI) AdminCall(r *http.Request, rawTx, result *string) (err error) {
	if !params.HasAdmin() {
		return fmt.Errorf("no admin is configed")
	}
	tx, err := admin.DecodeTransaction(*rawTx)
	if err != nil {
		return err
	}
	sender, args, err := admin.VerifyTransaction(tx)
	if err != nil {
		return err
	}
	if !params.IsAdmin(sender.String()) {
		return fmt.Errorf("sender %v is not admin", sender.String())
	}
	return doCall(args, result)
}

func doCall(args *admin.CallArgs, result *string) error {
	switch args.Method {
	case "blacklist":
		return blacklist(args, result)
	case "bigvalue":
		return bigvalue(args, result)
	case "maintain":
		return maintain(args, result)
	case "reverify":
		return reverify(args, result)
	case "reswap":
		return reswap(args, result)
	default:
		return fmt.Errorf("unknown admin method '%v'", args.Method)
	}
}

func blacklist(args *admin.CallArgs, result *string) (err error) {
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
	if err != nil {
		return err
	}
	if isQuery {
		if isBlacked {
			*result = "is in blacklist"
		} else {
			*result = "is not in blacklist"
		}
	} else {
		*result = successReuslt
	}
	return nil
}

func bigvalue(args *admin.CallArgs, result *string) (err error) {
	if len(args.Params) != 2 {
		return fmt.Errorf("wrong number of params, have %v want 2", len(args.Params))
	}
	operation := args.Params[0]
	txid := args.Params[1]
	switch operation {
	case "passswapin":
		err = mongodb.PassSwapinBigValue(txid)
	case "passswapout":
		err = mongodb.PassSwapoutBigValue(txid)
	default:
		return fmt.Errorf("unknown operation '%v'", operation)
	}
	if err != nil {
		return err
	}
	*result = successReuslt
	return nil
}

func maintain(args *admin.CallArgs, result *string) (err error) {
	if len(args.Params) != 2 {
		return fmt.Errorf("wrong number of params, have %v want 2", len(args.Params))
	}
	operation := args.Params[0]
	direction := args.Params[1]

	var newDisableFlag bool
	switch operation {
	case "open":
		newDisableFlag = false
	case "close":
		newDisableFlag = true
	default:
		return fmt.Errorf("unknown operation '%v'", operation)
	}

	isDeposit := false
	isWithdraw := false
	switch direction {
	case "deposit":
		isDeposit = true
	case "withdraw":
		isWithdraw = true
	case "both":
		isDeposit = true
		isWithdraw = true
	default:
		return fmt.Errorf("unknown direction '%v'", direction)
	}

	if isDeposit {
		tokens.GetTokenConfig(true).DisableSwap = newDisableFlag
	}

	if isWithdraw {
		tokens.GetTokenConfig(false).DisableSwap = newDisableFlag
	}

	*result = successReuslt
	return nil
}

func reverify(args *admin.CallArgs, result *string) (err error) {
	if len(args.Params) != 2 {
		return fmt.Errorf("wrong number of params, have %v want 2", len(args.Params))
	}
	operation := args.Params[0]
	txid := args.Params[1]
	switch operation {
	case swapinOp:
		err = mongodb.ReverifySwapin(txid)
	case swapoutOp:
		err = mongodb.ReverifySwapout(txid)
	default:
		return fmt.Errorf("unknown operation '%v'", operation)
	}
	if err != nil {
		return err
	}
	*result = successReuslt
	return nil
}

func reswap(args *admin.CallArgs, result *string) (err error) {
	if len(args.Params) != 2 {
		return fmt.Errorf("wrong number of params, have %v want 2", len(args.Params))
	}
	operation := args.Params[0]
	txid := args.Params[1]
	switch operation {
	case swapinOp:
		err = mongodb.Reswapin(txid)
	case swapoutOp:
		err = mongodb.Reswapout(txid)
	default:
		return fmt.Errorf("unknown operation '%v'", operation)
	}
	if err != nil {
		return err
	}
	*result = successReuslt
	return nil
}
