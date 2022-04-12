package terra

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func convertTo(src, dst interface{}) error {
	jsData, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsData, &dst)
}

// TokenInfo rpc type
type TokenInfo struct {
	Decimals    uint8  `json:"decimals"`
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	TotalSupply string `json:"total_supply"`
}

// GetTokenInfo impl
func (b *Bridge) GetTokenInfo(token string) (*TokenInfo, error) {
	query := map[string]map[string]interface{}{
		"token_info": {},
	}
	res, err := b.QueryContractStore(token, query)
	if err != nil {
		return nil, err
	}
	var tokenInfo TokenInfo
	err = convertTo(res, &tokenInfo)
	if err != nil {
		return nil, err
	}
	return &tokenInfo, nil
}

type TokenBalance struct {
	Balance sdk.Dec `json:"balance"`
}

// GetTokenBalance impl
func (b *Bridge) GetTokenBalance(token, account string) (sdk.Dec, error) {
	query := map[string]map[string]interface{}{
		"balance": {
			"address": account,
		},
	}
	res, err := b.QueryContractStore(token, query)
	if err != nil {
		return zeroDec, err
	}
	var tokenBalance TokenBalance
	err = convertTo(res, &tokenBalance)
	if err != nil {
		return zeroDec, err
	}
	return tokenBalance.Balance, nil
}

// GetTokenTransferExecMsg impl
// `recipient` is user address.
func GetTokenTransferExecMsg(recipient, amount string) (string, error) {
	execMsg := map[string]map[string]interface{}{
		"transfer": {
			"recipient": recipient,
			"amount":    amount,
		},
	}
	return base64EncodedJSON(execMsg)
}

// GetTokenSendExecMsg impl
// `recipient` is contract address.
// `msg` is base64-endcoded JSON string.
func GetTokenSendExecMsg(recipient, amount, msg string) (string, error) {
	execMsg := map[string]map[string]interface{}{
		"send": {
			"contract": recipient,
			"amount":   amount,
			"msg":      msg,
		},
	}
	return base64EncodedJSON(execMsg)
}
