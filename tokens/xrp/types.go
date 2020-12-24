package xrp

import (
	"strings"

	"github.com/shawn-cx-li/ripple/data"
)

//
type Account_info_Res struct {
	Account_data Account
}

type Account struct {
	Balance  string
	Sequence uint32
}

type AccountResp struct {
	Result Account_info_Res
}

func parseAccount(s string) *data.Account {
	account, err := data.NewAccountFromAddress(s)
	if err != nil {
		return nil
	}
	return account
}

func parseAmount(s string) *data.Amount {
	amount, err := data.NewAmount(s)
	if err != nil {
		return nil
	}
	return amount
}

func parsePaths(s string) *data.PathSet {
	ps := data.PathSet{}
	for _, pathStr := range strings.Split(s, ",") {
		path, err := data.NewPath(pathStr)
		if err != nil {
			return nil
		}
		ps = append(ps, path)
	}
	return &ps
}
