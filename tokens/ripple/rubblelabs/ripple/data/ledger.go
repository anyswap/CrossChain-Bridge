package data

type LedgerHeader struct {
	LedgerSequence  uint32     `json:"ledger_index,string"`
	TotalXRP        uint64     `json:"total_coins,string"`
	PreviousLedger  Hash256    `json:"parent_hash"`
	TransactionHash Hash256    `json:"transaction_hash"`
	StateHash       Hash256    `json:"account_hash"`
	ParentCloseTime RippleTime `json:"parent_close_time"`
	CloseTime       RippleTime `json:"close_time"`
	CloseResolution uint8      `json:"close_time_resolution"`
	CloseFlags      uint8      `json:"close_flags"`
}

type Ledger struct {
	LedgerHeader
	Hash         Hash256          `json:"hash"`
	Closed       bool             `json:"closed"`
	Accepted     bool             `json:"accepted"`
	Transactions TransactionSlice `json:"transactions,omitempty"`
	AccountState LedgerEntrySlice `json:"accountState,omitempty"`
}

func NewEmptyLedger(sequence uint32) *Ledger {
	return &Ledger{
		LedgerHeader: LedgerHeader{
			LedgerSequence: sequence,
		},
	}
}

func (l Ledger) GetType() string    { return "LedgerMaster" }
func (l Ledger) Prefix() HashPrefix { return HP_LEDGER_MASTER }
func (l Ledger) NodeType() NodeType { return NT_LEDGER }
func (l Ledger) Ledger() uint32     { return l.LedgerSequence }
func (l Ledger) NodeId() *Hash256   { return &l.Hash }
func (l Ledger) GetHash() *Hash256  { return &l.Hash }
