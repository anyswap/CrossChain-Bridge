package ripple

import (
	"strings"

	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/data"
)

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
