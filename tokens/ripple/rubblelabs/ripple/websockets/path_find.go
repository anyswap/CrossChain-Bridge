package websockets

import (
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/data"
)

// https://ripple.com/build/rippled-apis/#path-find
/*
{
    "id": 8,
    "command": "path_find",
    "subcommand": "create",
    "source_account": "r9cZA1mLK5R5Am25ArfXFmqgNwjZgnfk59",
    "destination_account": "r9cZA1mLK5R5Am25ArfXFmqgNwjZgnfk59",
    "destination_amount": {
        "value": "0.001",
        "currency": "USD",
        "issuer": "rvYAfWj5gh67oV6fW32ZzP3Aw4Eubs59B"
    }
}
*/
type PathFindCreateCommand struct {
	*Command
	Subcommand         string            `json:"subcommand"`
	SourceAccount      data.Account      `json:"source_account"`
	DestinationAccount data.Account      `json:"destination_account"`
	DestinationAmount  data.Amount       `json:"destination_amount"`
	SendMax            *data.Amount      `json:"send_max,omitempty"`
	SourceCurrencies   *[]SourceCurrency `json:"source_currencies,omitempty"`

	// All commands have Result in their struct?
	Result *PathFindCreateResult
}

type SourceCurrency struct {
	Currency string `json:"currency"`
}

func (r *Remote) PathFindCreate(src, dest data.Account, amt data.Amount, sendMax *data.Amount, sourceCurrencies *[]SourceCurrency) (*PathFindCreateResult, error) {
	cmd := &PathFindCreateCommand{
		Command:            newCommand("path_find"),
		Subcommand:         "create",
		SourceAccount:      src,
		DestinationAccount: dest,
		DestinationAmount:  amt,
		SendMax:            sendMax,
		SourceCurrencies:   sourceCurrencies,
	}
	r.outgoing <- cmd
	<-cmd.Ready
	if cmd.CommandError != nil {
		return nil, cmd.CommandError
	}
	return cmd.Result, nil
}

/*

{
  "id": 1,
  "status": "success",
  "type": "response",
  "result": {
    "alternatives": [
      {
        "paths_computed": [
          [
            {
              "currency": "USD",
              "issuer": "rvYAfWj5gh67oV6fW32ZzP3Aw4Eubs59B",
              "type": 48,
              "type_hex": "0000000000000030"
            },
            {
              "account": "rvYAfWj5gh67oV6fW32ZzP3Aw4Eubs59B",
              "type": 1,
              "type_hex": "0000000000000001"
            }
          ],
       <snip>
    ],
    "destination_account": "r9cZA1mLK5R5Am25ArfXFmqgNwjZgnfk59",
    "destination_amount": {
      "currency": "USD",
      "issuer": "rvYAfWj5gh67oV6fW32ZzP3Aw4Eubs59B",
      "value": "0.001"
    },
    "id": 1,
    "source_account": "r9cZA1mLK5R5Am25ArfXFmqgNwjZgnfk59",
    "full_reply": false
  }
}
*/

type PathFindAlternative struct {
	// TODO paths_computed
	SourceAmount data.Amount `json:"source_amount"`
}

type PathFindCreateResult struct {
	SourceAccount      data.Account `json:"source_account"`
	DestinationAccount data.Account `json:"destination_account"`
	DestinationAmount  data.Amount  `json:"destination_amount"`
	// TODO
	Alternatives []PathFindAlternative
}
