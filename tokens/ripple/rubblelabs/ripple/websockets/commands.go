package websockets

import (
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/data"
)

var counter uint64

type Syncer interface {
	Done()
	Fail(message string)
}

type CommandError struct {
	Name    string `json:"error"`
	Code    int    `json:"error_code"`
	Message string `json:"error_message"`
}

type Command struct {
	*CommandError
	Id     uint64        `json:"id"`
	Name   string        `json:"command"`
	Type   string        `json:"type,omitempty"`
	Status string        `json:"status,omitempty"`
	Ready  chan struct{} `json:"-"`
}

func (c *Command) Done() {
	c.Ready <- struct{}{}
}

func (c *Command) Fail(message string) {
	c.CommandError = &CommandError{
		Name:    "Client Error",
		Code:    -1,
		Message: message,
	}
	c.Ready <- struct{}{}
}

func (c *Command) IncrementId() {
	c.Id = atomic.AddUint64(&counter, 1)
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("%s %d %s", e.Name, e.Code, e.Message)
}

func newCommand(command string) *Command {
	return &Command{
		Id:    atomic.AddUint64(&counter, 1),
		Name:  command,
		Ready: make(chan struct{}),
	}
}

type AccountTxCommand struct {
	*Command
	Account   data.Account           `json:"account"`
	MinLedger int64                  `json:"ledger_index_min"`
	MaxLedger int64                  `json:"ledger_index_max"`
	Binary    bool                   `json:"binary,omitempty"`
	Forward   bool                   `json:"forward,omitempty"`
	Limit     int                    `json:"limit,omitempty"`
	Marker    map[string]interface{} `json:"marker,omitempty"`
	Result    *AccountTxResult       `json:"result,omitempty"`
}

type AccountTxResult struct {
	Marker       map[string]interface{} `json:"marker,omitempty"`
	Transactions data.TransactionSlice  `json:"transactions,omitempty"`
}

func newAccountTxCommand(account data.Account, pageSize int, marker map[string]interface{}, minLedger, maxLedger int64) *AccountTxCommand {
	return &AccountTxCommand{
		Command:   newCommand("account_tx"),
		Account:   account,
		MinLedger: minLedger,
		MaxLedger: maxLedger,
		Limit:     pageSize,
		Marker:    marker,
	}
}

func newBinaryLedgerDataCommand(ledger interface{}, marker *data.Hash256) *BinaryLedgerDataCommand {
	return &BinaryLedgerDataCommand{
		Command: newCommand("ledger_data"),
		Ledger:  ledger,
		Binary:  true,
		Marker:  marker,
	}
}

type TxCommand struct {
	*Command
	Transaction data.Hash256 `json:"transaction"`
	Result      *TxResult    `json:"result,omitempty"`
}

type TxResult struct {
	data.TransactionWithMetaData
	Validated bool `json:"validated"`
}

// A shim to populate the Validated field before passing
// control on to TransactionWithMetaData.UnmarshalJSON
func (txr *TxResult) UnmarshalJSON(b []byte) error {
	var extract map[string]interface{}
	if err := json.Unmarshal(b, &extract); err != nil {
		return err
	}
	// "validated" can be absent, when tx result is provisional.
	validated, ok := extract["validated"]
	if !ok {
		txr.Validated = false
	} else {
		txr.Validated = validated.(bool)
	}
	return json.Unmarshal(b, &txr.TransactionWithMetaData)
}

type SubmitCommand struct {
	*Command
	TxBlob string        `json:"tx_blob"`
	Result *SubmitResult `json:"result,omitempty"`
}

type SubmitResult struct {
	EngineResult        data.TransactionResult `json:"engine_result"`
	EngineResultCode    int                    `json:"engine_result_code"`
	EngineResultMessage string                 `json:"engine_result_message"`
	TxBlob              string                 `json:"tx_blob"`
	Tx                  interface{}            `json:"tx_json"`
}

type LedgerCommand struct {
	*Command
	LedgerIndex  interface{}   `json:"ledger_index"`
	Accounts     bool          `json:"accounts"`
	Transactions bool          `json:"transactions"`
	Expand       bool          `json:"expand"`
	Result       *LedgerResult `json:"result,omitempty"`
}

type LedgerResult struct {
	Ledger data.Ledger
}

type LedgerHeaderCommand struct {
	*Command
	Ledger interface{} `json:"ledger"`
	Result *LedgerHeaderResult
}

type LedgerHeaderResult struct {
	Ledger         data.Ledger
	LedgerSequence uint32              `json:"ledger_index"`
	Hash           *data.Hash256       `json:"ledger_hash,omitempty"`
	LedgerData     data.VariableLength `json:"ledger_data"`
}

type LedgerDataCommand struct {
	*Command
	Ledger interface{}       `json:"ledger"`
	Marker *data.Hash256     `json:"marker,omitempty"`
	Result *LedgerDataResult `json:"result,omitempty"`
}

type BinaryLedgerDataCommand struct {
	*Command
	Ledger interface{}             `json:"ledger"`
	Binary bool                    `json:"binary"`
	Marker *data.Hash256           `json:"marker,omitempty"`
	Result *BinaryLedgerDataResult `json:"result,omitempty"`
}

type LedgerDataResult struct {
	LedgerSequence uint32                `json:"ledger_index"`
	Hash           data.Hash256          `json:"ledger_hash"`
	Marker         *data.Hash256         `json:"marker"`
	State          data.LedgerEntrySlice `json:"state"`
}

type BinaryLedgerData struct {
	Data  string `json:"data"`
	Index string `json:"index"`
}

type BinaryLedgerDataResult struct {
	LedgerSequence uint32             `json:"ledger_index"`
	Hash           data.Hash256       `json:"ledger_hash"`
	Marker         *data.Hash256      `json:"marker"`
	State          []BinaryLedgerData `json:"state"`
}

type RipplePathFindCommand struct {
	*Command
	SrcAccount    data.Account          `json:"source_account"`
	SrcCurrencies *[]data.Currency      `json:"source_currencies,omitempty"`
	DestAccount   data.Account          `json:"destination_account"`
	DestAmount    data.Amount           `json:"destination_amount"`
	Result        *RipplePathFindResult `json:"result,omitempty"`
}

type RipplePathFindResult struct {
	Alternatives []struct {
		SrcAmount      data.Amount  `json:"source_amount"`
		PathsComputed  data.PathSet `json:"paths_computed,omitempty"`
		PathsCanonical data.PathSet `json:"paths_canonical,omitempty"`
	}
	DestAccount    data.Account    `json:"destination_account"`
	DestCurrencies []data.Currency `json:"destination_currencies"`
}

type AccountInfoCommand struct {
	*Command
	Account data.Account       `json:"account"`
	Result  *AccountInfoResult `json:"result,omitempty"`
}

type AccountInfoResult struct {
	LedgerSequence uint32           `json:"ledger_current_index"`
	AccountData    data.AccountRoot `json:"account_data"`
}

type AccountLinesCommand struct {
	*Command
	Account     data.Account        `json:"account"`
	Limit       uint32              `json:"limit"`
	LedgerIndex interface{}         `json:"ledger_index,omitempty"`
	Marker      *data.Hash256       `json:"marker,omitempty"`
	Result      *AccountLinesResult `json:"result,omitempty"`
}

type AccountLinesResult struct {
	LedgerSequence *uint32               `json:"ledger_index"`
	Account        data.Account          `json:"account"`
	Marker         *data.Hash256         `json:"marker"`
	Lines          data.AccountLineSlice `json:"lines"`
}

type AccountOffersCommand struct {
	*Command
	Account     data.Account         `json:"account"`
	Limit       uint32               `json:"limit"`
	LedgerIndex interface{}          `json:"ledger_index,omitempty"`
	Marker      *data.Hash256        `json:"marker,omitempty"`
	Result      *AccountOffersResult `json:"result,omitempty"`
}

type AccountOffersResult struct {
	LedgerSequence *uint32                `json:"ledger_index"`
	Account        data.Account           `json:"account"`
	Marker         *data.Hash256          `json:"marker"`
	Offers         data.AccountOfferSlice `json:"offers"`
}

type BookOffersCommand struct {
	*Command
	LedgerIndex interface{}  `json:"ledger_index,omitempty"`
	Taker       data.Account `json:"taker"`
	TakerPays   data.Asset   `json:"taker_pays"`
	TakerGets   data.Asset   `json:"taker_gets"`
	Limit       uint32       `json:"limit"`
	Result      *BookOffersResult
}

type BookOffersResult struct {
	LedgerSequence uint32                `json:"ledger_index"`
	Offers         []data.OrderBookOffer `json:"offers"`
}

type FeeCommand struct {
	*Command
	Result *FeeResult
}

type FeeResult struct {
	CurrentLedgerSize uint32 `json:"current_ledger_size,string"`
	CurrentQueueSize  uint32 `json:"current_queue_size,string"`
	Drops             struct {
		BaseFee       data.Value `json:"base_fee"`
		MedianFee     data.Value `json:"median_fee"`
		MinimumFee    data.Value `json:"minimum_fee"`
		OpenLedgerFee data.Value `json:"open_ledger_fee"`
	} `json:"drops"`
	ExpectedLedgerSize uint32 `json:"expected_ledger_size,string"`
	Levels             struct {
		MedianLevel     data.Value `json:"median_level"`
		MinimumLevel    data.Value `json:"minimum_level"`
		OpenLedgerLevel data.Value `json:"open_ledger_level"`
		ReferenceLevel  data.Value `json:"reference_level"`
	} `json:"levels"`
	MaxQueueSize uint32 `json:"max_queue_size,string"`
	Status       string `json:"status"`
}
