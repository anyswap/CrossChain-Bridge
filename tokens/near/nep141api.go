package near

import (
	"encoding/json"
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
	return nil, nil
}

type TokenBalance struct {
	Balance uint64 `json:"balance"`
}

// GetTokenBalance impl
func (b *Bridge) GetTokenBalance(token, account string) (uint64, error) {
	return 0, nil
}

// GetTokenTransferExecMsg impl
// `recipient` is user address.
func GetTokenTransferExecMsg(recipient, amount string) ([]byte, error) {
	execMsg := map[string]map[string]interface{}{
		"transfer": {
			"recipient": recipient,
			"amount":    amount,
		},
	}
	return json.Marshal(execMsg)
}

// GetTokenSendExecMsg impl
// `recipient` is contract address.
// `msg` is base64-endcoded JSON string.
func GetTokenSendExecMsg(recipient, amount, msg string) ([]byte, error) {
	execMsg := map[string]map[string]interface{}{
		"send": {
			"contract": recipient,
			"amount":   amount,
			"msg":      msg,
		},
	}
	return json.Marshal(execMsg)
}
