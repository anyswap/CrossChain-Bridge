package rpcapi

import (
	"fmt"
	"net/http"

	"github.com/anyswap/CrossChain-Bridge/admin"
	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/worker"
)

const (
	successReuslt = "Success"
	swapinOp      = "swapin"
	swapoutOp     = "swapout"
	passSwapinOp  = "passswapin"
	passSwapoutOp = "passswapout"
	failSwapinOp  = "failswapin"
	failSwapoutOp = "failswapout"
	forceFlag     = "--force"
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
	case "manual":
		return manual(args, result)
	case "setnonce":
		return setnonce(args, result)
	case "addpair":
		return addpair(args, result)
	default:
		return fmt.Errorf("unknown admin method '%v'", args.Method)
	}
}

func blacklist(args *admin.CallArgs, result *string) (err error) {
	if len(args.Params) != 3 {
		return fmt.Errorf("wrong number of params, have %v want 3", len(args.Params))
	}
	operation := args.Params[0]
	address := args.Params[1]
	pairID := args.Params[2]
	isBlacked := false
	isQuery := false
	switch operation {
	case "add":
		err = mongodb.AddToBlacklist(address, pairID)
	case "remove":
		err = mongodb.RemoveFromBlacklist(address, pairID)
	case "query":
		isQuery = true
		isBlacked, err = mongodb.QueryBlacklist(address, pairID)
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
	if len(args.Params) != 4 {
		return fmt.Errorf("wrong number of params, have %v want 4", len(args.Params))
	}
	operation := args.Params[0]
	txid := args.Params[1]
	pairID := args.Params[2]
	bind := args.Params[3]
	switch operation {
	case passSwapinOp:
		err = mongodb.PassSwapinBigValue(txid, pairID, bind)
	case passSwapoutOp:
		err = mongodb.PassSwapoutBigValue(txid, pairID, bind)
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
	if len(args.Params) != 3 {
		return fmt.Errorf("wrong number of params, have %v want 3", len(args.Params))
	}
	operation := args.Params[0]
	direction := args.Params[1]
	pairID := args.Params[2]

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
		tokenCfg := tokens.GetTokenConfig(pairID, true)
		if tokenCfg == nil {
			return fmt.Errorf("pairID %v is not configed", pairID)
		}
		tokenCfg.DisableSwap = newDisableFlag
	}

	if isWithdraw {
		tokenCfg := tokens.GetTokenConfig(pairID, false)
		if tokenCfg == nil {
			return fmt.Errorf("pairID %v is not configed", pairID)
		}
		tokenCfg.DisableSwap = newDisableFlag
	}

	*result = successReuslt
	return nil
}

func getOpTxAndPairID(args *admin.CallArgs) (operation, txid, pairID, bind, forceOpt string, err error) {
	if !(len(args.Params) == 4 || len(args.Params) == 5) {
		err = fmt.Errorf("wrong number of params, have %v want 4 or 5", len(args.Params))
		return
	}
	operation = args.Params[0]
	txid = args.Params[1]
	pairID = args.Params[2]
	bind = args.Params[3]

	if len(args.Params) > 4 {
		forceOpt = args.Params[4]
		if forceOpt != forceFlag {
			err = fmt.Errorf("wrong force flag %v, must be %v", forceOpt, forceFlag)
			return
		}
	}
	return operation, txid, pairID, bind, forceOpt, nil
}

func reverify(args *admin.CallArgs, result *string) (err error) {
	operation, txid, pairID, bind, _, err := getOpTxAndPairID(args)
	if err != nil {
		return err
	}
	switch operation {
	case swapinOp:
		err = mongodb.ReverifySwapin(txid, pairID, bind)
	case swapoutOp:
		err = mongodb.ReverifySwapout(txid, pairID, bind)
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
	operation, txid, pairID, bind, forceOpt, err := getOpTxAndPairID(args)
	if err != nil {
		return err
	}
	switch operation {
	case swapinOp:
		err = mongodb.Reswapin(txid, pairID, bind, forceOpt)
	case swapoutOp:
		err = mongodb.Reswapout(txid, pairID, bind, forceOpt)
	default:
		return fmt.Errorf("unknown operation '%v'", operation)
	}
	if err != nil {
		return err
	}
	*result = successReuslt
	return nil
}

func manual(args *admin.CallArgs, result *string) (err error) {
	if !(len(args.Params) == 4 || len(args.Params) == 5) {
		return fmt.Errorf("wrong number of params, have %v want 4 or 5", len(args.Params))
	}
	operation := args.Params[0]
	txid := args.Params[1]
	pairID := args.Params[2]
	bind := args.Params[3]

	var memo string
	if len(args.Params) > 4 {
		memo = args.Params[4]
	}

	var isSwapin, isPass bool
	switch operation {
	case passSwapinOp:
		isSwapin = true
		isPass = true
	case failSwapinOp:
		isSwapin = true
		isPass = false
	case passSwapoutOp:
		isSwapin = false
		isPass = true
	case failSwapoutOp:
		isSwapin = false
		isPass = false
	default:
		return fmt.Errorf("unknown operation '%v'", operation)
	}
	err = mongodb.ManualManageSwap(txid, pairID, bind, memo, isSwapin, isPass)
	if err != nil {
		return err
	}
	*result = successReuslt
	return nil
}

func setnonce(args *admin.CallArgs, result *string) (err error) {
	if len(args.Params) != 3 {
		return fmt.Errorf("wrong number of params, have %v want 3", len(args.Params))
	}
	operation := args.Params[0]
	nonce, err := common.GetUint64FromStr(args.Params[1])
	if err != nil {
		return fmt.Errorf("wrong nonce value, %v", err)
	}
	pairID := args.Params[2]
	var bridge tokens.CrossChainBridge
	switch operation {
	case swapinOp:
		bridge = tokens.DstBridge
	case swapoutOp:
		bridge = tokens.SrcBridge
	default:
		return fmt.Errorf("unknown operation '%v'", operation)
	}
	nonceSetter, ok := bridge.(tokens.NonceSetter)
	if !ok {
		return fmt.Errorf("nonce setter not supported")
	}
	nonceSetter.SetNonce(pairID, nonce)
	*result = successReuslt
	return nil
}

func addpair(args *admin.CallArgs, result *string) (err error) {
	if len(args.Params) != 1 {
		return fmt.Errorf("wrong number of params, have %v want 1", len(args.Params))
	}
	configFile := args.Params[0]
	pairConfig, err := tokens.AddPairConfig(configFile)
	if err != nil {
		return err
	}
	worker.AddSwapJob(pairConfig)
	*result = successReuslt
	return nil
}
