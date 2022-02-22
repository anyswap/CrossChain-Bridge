package websockets

import (
	"encoding/json"

	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/data"
)

// Fields from subscribed ledger stream messages
type LedgerStreamMsg struct {
	FeeBase          uint64          `json:"fee_base"`
	FeeRef           uint64          `json:"fee_ref"`
	LedgerSequence   uint32          `json:"ledger_index"`
	LedgerHash       data.Hash256    `json:"ledger_hash"`
	LedgerTime       data.RippleTime `json:"ledger_time"`
	ReserveBase      uint64          `json:"reserve_base"`
	ReserveIncrement uint64          `json:"reserve_inc"`
	ValidatedLedgers string          `json:"validated_ledgers"`
	TxnCount         uint32          `json:"txn_count"` // Only streamed, not in the subscribe result.
}

// Fields from subscribed transaction stream messages
type TransactionStreamMsg struct {
	Transaction         data.TransactionWithMetaData `json:"transaction"`
	EngineResult        data.TransactionResult       `json:"engine_result"`
	EngineResultCode    int                          `json:"engine_result_code"`
	EngineResultMessage string                       `json:"engine_result_message"`
	LedgerHash          data.Hash256                 `json:"ledger_hash"`
	LedgerSequence      uint32                       `json:"ledger_index"`
	Status              string
	Validated           bool
}

// Fields from subscribed server status stream messages
type ServerStreamMsg struct {
	Status                  string `json:"server_status"`
	BaseFee                 uint64 `json:"base_fee"`
	LoadBase                uint64 `json:"load_base"`
	LoadFactor              uint64 `json:"load_factor"`
	LoadFactorFeeEscalation uint64 `json:"load_factor_fee_escalation"`
	LoadFactorFeeQueue      uint64 `json:"load_factor_fee_queue"`
	LoadFactorFeeReference  uint64 `json:"load_factor_fee_reference"`
	LoadFactorServer        uint64 `json:"load_factor_server"`
	HostID                  string `json:"hostid"`
}

func (s *ServerStreamMsg) TransactionCost() uint64 {
	return (s.BaseFee * s.LoadFactor) / s.LoadBase
}

// Map message types to the appropriate data structure
var streamMessageFactory = map[string]func() interface{}{
	"ledgerClosed": func() interface{} { return &LedgerStreamMsg{} },
	"transaction":  func() interface{} { return &TransactionStreamMsg{} },
	"serverStatus": func() interface{} { return &ServerStreamMsg{} },
	"path_find":    func() interface{} { return &PathFindCreateResult{} },
}

type SubscribeCommand struct {
	*Command
	Streams []string                `json:"streams"`
	Books   []OrderBookSubscription `json:"books,omitempty"`
	Result  *SubscribeResult        `json:"result,omitempty"`
}

type SubscribeResult struct {
	// Contains one or both of these, depending what streams were subscribed
	*LedgerStreamMsg
	*ServerStreamMsg
	// Contains "bids" and "asks" when "both" is true.
	Asks []data.OrderBookOffer
	Bids []data.OrderBookOffer
	// Contains "offers" when "both" is false.
	Offers []data.OrderBookOffer
}

// Wrapper to stop recursive unmarshalling
type txStreamJSON TransactionStreamMsg

func (msg *TransactionStreamMsg) UnmarshalJSON(b []byte) error {
	var extract struct {
		*txStreamJSON
		MetaData *data.MetaData `json:"meta"`
	}
	extract.txStreamJSON = (*txStreamJSON)(msg)
	extract.MetaData = &msg.Transaction.MetaData
	return json.Unmarshal(b, &extract)
}
