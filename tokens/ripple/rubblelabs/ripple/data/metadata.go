package data

import (
	"fmt"
	"sort"
)

type LedgerEntryState uint8

const (
	Created LedgerEntryState = iota
	Modified
	Deleted
)

type AffectedNode struct {
	FinalFields       LedgerEntry `json:",omitempty"`
	LedgerEntryType   LedgerEntryType
	LedgerIndex       *Hash256    `json:",omitempty"`
	PreviousFields    LedgerEntry `json:",omitempty"`
	NewFields         LedgerEntry `json:",omitempty"`
	PreviousTxnID     *Hash256    `json:",omitempty"`
	PreviousTxnLgrSeq *uint32     `json:",omitempty"`
}

type NodeEffect struct {
	ModifiedNode *AffectedNode `json:",omitempty"`
	CreatedNode  *AffectedNode `json:",omitempty"`
	DeletedNode  *AffectedNode `json:",omitempty"`
}

type NodeEffects []NodeEffect

type MetaData struct {
	AffectedNodes     NodeEffects
	TransactionIndex  uint32
	TransactionResult TransactionResult
	DeliveredAmount   *Amount `json:"delivered_amount,omitempty"`
}

type TransactionSlice []*TransactionWithMetaData

func (s TransactionSlice) Len() int      { return len(s) }
func (s TransactionSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s TransactionSlice) Less(i, j int) bool {
	if s[i].LedgerSequence == s[j].LedgerSequence {
		return s[i].MetaData.TransactionIndex < s[j].MetaData.TransactionIndex
	}
	return s[i].LedgerSequence < s[j].LedgerSequence
}

func (s TransactionSlice) Sort() { sort.Sort(s) }

type TransactionWithMetaData struct {
	Transaction
	MetaData       MetaData   `json:"meta"`
	Date           RippleTime `json:"date"`
	LedgerSequence uint32     `json:"ledger_index"`
	Id             Hash256    `json:"-"`
}

func (t *TransactionWithMetaData) GetType() string    { return t.Transaction.GetType() }
func (t *TransactionWithMetaData) Prefix() HashPrefix { return HP_TRANSACTION_NODE }
func (t *TransactionWithMetaData) NodeType() NodeType { return NT_TRANSACTION_NODE }
func (t *TransactionWithMetaData) Ledger() uint32     { return t.LedgerSequence }
func (t *TransactionWithMetaData) NodeId() *Hash256   { return &t.Id }

func (t *TransactionWithMetaData) Affects(account Account) bool {
	for _, effect := range t.MetaData.AffectedNodes {
		if _, final, _, _ := effect.AffectedNode(); final.Affects(account) {
			return true
		}
	}
	return false
}

func NewTransactionWithMetadata(typ TransactionType) *TransactionWithMetaData {
	return &TransactionWithMetaData{Transaction: TxFactory[typ]()}
}

// AffectedNode returns the AffectedNode, the current LedgerEntry,
// the previous LedgerEntry (which might be nil) and the LedgerEntryState
func (effect *NodeEffect) AffectedNode() (*AffectedNode, LedgerEntry, LedgerEntry, LedgerEntryState) {
	var (
		node            *AffectedNode
		final, previous LedgerEntry
		state           LedgerEntryState
	)
	switch {
	case effect.CreatedNode != nil && effect.CreatedNode.NewFields != nil:
		node, final, state = effect.CreatedNode, effect.CreatedNode.NewFields, Created
	case effect.DeletedNode != nil && effect.DeletedNode.FinalFields != nil:
		node, final, state = effect.DeletedNode, effect.DeletedNode.FinalFields, Deleted
	case effect.ModifiedNode != nil && effect.ModifiedNode.FinalFields != nil:
		node, final, state = effect.ModifiedNode, effect.ModifiedNode.FinalFields, Modified
	case effect.ModifiedNode != nil && effect.ModifiedNode.FinalFields == nil:
		node, final, state = effect.ModifiedNode, LedgerEntryFactory[effect.ModifiedNode.LedgerEntryType](), Modified
	default:
		panic(fmt.Sprintf("Unknown LedgerEntryState: %+v", effect))
	}
	previous = node.PreviousFields
	if previous == nil {
		previous = LedgerEntryFactory[final.GetLedgerEntryType()]()
	}
	return node, final, previous, state
}
