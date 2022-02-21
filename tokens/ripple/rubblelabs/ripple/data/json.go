package data

// Evil things happen here. Rippled needs a V2 API...

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type ledgerJSON Ledger

// adds all the legacy fields
type ledgerExtraJSON struct {
	ledgerJSON
	HumanCloseTime *rippleHumanTime `json:"close_time_human"`
	LedgerHash     Hash256          `json:"ledger_hash"`
	TotalCoins     uint64           `json:"totalCoins,string"`
	SequenceNumber uint32           `json:"seqNum,string"`
}

func (l Ledger) MarshalJSON() ([]byte, error) {
	return json.Marshal(ledgerExtraJSON{
		ledgerJSON:     ledgerJSON(l),
		HumanCloseTime: l.CloseTime.human(),
		LedgerHash:     l.Hash,
		TotalCoins:     l.TotalXRP,
		SequenceNumber: l.LedgerSequence,
	})
}

func (l *Ledger) UnmarshalJSON(b []byte) error {
	var ledger ledgerExtraJSON
	if err := json.Unmarshal(b, &ledger); err != nil {
		return err
	}
	*l = Ledger(ledger.ledgerJSON)
	return nil
}

// Wrapper types to enable second level of marshalling
// when found in tx API call
type txmNormal TransactionWithMetaData

var (
	txmSplitTypeRegex       = regexp.MustCompile(`"tx":`)
	txmMetaDataRegex        = regexp.MustCompile(`"metaData":`)
	txmTransactionTypeRegex = regexp.MustCompile(`"TransactionType"\s*:\s*"(\w+)"`)
)

// This function is a horrow show, demonstrating the huge
// inconsistencies in the presentation of a transaction
// by the rippled API.  Indeed.
func (txm *TransactionWithMetaData) UnmarshalJSON(b []byte) error {
	if txmSplitTypeRegex.Match(b) {
		// Transaction has the form {"tx":{}, "meta":{}, "validated": true}
		// i.e. returned from `account_tx` command.
		var split struct {
			Tx   json.RawMessage
			Meta json.RawMessage
		}
		if err := json.Unmarshal(b, &split); err != nil {
			return err
		}
		if err := json.Unmarshal(split.Tx, txm); err != nil {
			return err
		}
		return json.Unmarshal(split.Meta, &txm.MetaData)
	}

	// Sniff the transaction type, and allocate the appropriate type
	txTypeMatch := txmTransactionTypeRegex.FindStringSubmatch(string(b))
	if txTypeMatch == nil {
		return fmt.Errorf("Not a valid transaction with metadata: Missing TransactionType")
	}
	txType := txTypeMatch[1]
	txm.Transaction = GetTxFactoryByType(txType)()
	if err := json.Unmarshal(b, txm.Transaction); err != nil {
		return err
	}

	if txmMetaDataRegex.Match(b) {
		// Transaction has the form {...fields..., "metaData":{...}}
		// (no "validated" or ledger sequence or id)
		// i.e. it comes from `ledger` command.
		// Further, "metaData" for payments has "DeliveredAmount" instead of the expected "delivered_amount", so clean that up first.
		b = bytes.Replace(b, []byte("\"DeliveredAmount\":"), []byte("\"delivered_amount\":"), 1)

		// Parse the rest in one shot
		extract := &struct {
			*txmNormal
			MetaData *MetaData `json:"metaData"`
		}{
			txmNormal: (*txmNormal)(txm),
			MetaData:  &txm.MetaData,
		}
		return json.Unmarshal(b, extract)
	}

	// Transaction has the form {...fields..., "metaData":{...}}
	// i.e. it comes from `tx` command.
	extract := &struct {
		*txmNormal
		Date     *RippleTime
		MetaData *MetaData `json:"metaData"`
	}{
		txmNormal: (*txmNormal)(txm),
		Date:      &txm.Date,
		MetaData:  &txm.MetaData,
	}
	return json.Unmarshal(b, extract)
}

func (txm TransactionWithMetaData) marshalJSON() ([]byte, []byte, error) {
	tx, err := json.Marshal(txm.Transaction)
	if err != nil {
		return nil, nil, err
	}
	meta, err := json.Marshal(txm.MetaData)
	if err != nil {
		return nil, nil, err
	}
	return tx, meta, nil
}

const txmFormat = `%s,"hash":"%s","inLedger":%d,"ledger_index":%d,"meta":%s}`

func (txm TransactionWithMetaData) MarshalJSON() ([]byte, error) {
	tx, meta, err := txm.marshalJSON()
	if err != nil {
		return nil, err
	}
	out := fmt.Sprintf(txmFormat, string(tx[:len(tx)-1]), txm.GetHash().String(), txm.LedgerSequence, txm.LedgerSequence, string(meta))
	return []byte(out), nil
}

const txmSliceFormat = `%s,"hash":"%s","metaData":%s}`

func (s TransactionSlice) MarshalJSON() ([]byte, error) {
	raw := make([]json.RawMessage, len(s))
	var err error
	var tx, meta []byte
	for i, txm := range s {
		if tx, meta, err = txm.marshalJSON(); err != nil {
			return nil, err
		}
		extra := fmt.Sprintf(txmSliceFormat, string(tx[:len(tx)-1]), txm.GetHash().String(), meta)
		raw[i] = json.RawMessage(extra)
	}
	return json.Marshal(raw)
}

var (
	leTypeRegex  = regexp.MustCompile(`"LedgerEntryType"\s*:\s*"(\w+)"`)
	leIndexRegex = regexp.MustCompile(`"index"\s*:\s*"(\w+)"`)
)

func (l *LedgerEntrySlice) UnmarshalJSON(b []byte) error {
	var s []json.RawMessage
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	for _, raw := range s {
		leTypeMatch := leTypeRegex.FindStringSubmatch(string(raw))
		indexMatch := leIndexRegex.FindStringSubmatch(string(raw))
		if leTypeMatch == nil {
			return fmt.Errorf("Bad LedgerEntryType")
		}
		if indexMatch == nil {
			return fmt.Errorf("Missing LedgerEntry index")
		}
		le := GetLedgerEntryFactoryByType(leTypeMatch[1])()
		if err := json.Unmarshal(raw, &le); err != nil {
			return err
		}
		*l = append(*l, le)
	}
	return nil
}

// const leSliceFormat = `%s,"LedgerEntryType":"%s"}`

// func (s LedgerEntrySlice) MarshalJSON() ([]byte, error) {
// 	var raw []json.RawMessage
// 	for _, le := range s {
// 		b, err := json.Marshal(le)
// 		if err != nil {
// 			return nil, err
// 		}
// 		extra := fmt.Sprintf(leSliceFormat, string(b[:len(b)-1]), le.GetLedgerEntryType())
// 		raw = append(raw, json.RawMessage(extra))
// 	}
// 	return json.Marshal(raw)
// }

type affectedNodeJSON struct {
	LedgerEntryType   LedgerEntryType
	LedgerIndex       *Hash256
	PreviousTxnID     *Hash256
	PreviousTxnLgrSeq *uint32
	FinalFields       json.RawMessage `json:",omitempty"`
	PreviousFields    json.RawMessage `json:",omitempty"`
	NewFields         json.RawMessage `json:",omitempty"`
}

func (a *AffectedNode) UnmarshalJSON(b []byte) error {
	var affected affectedNodeJSON
	if err := json.Unmarshal(b, &affected); err != nil {
		return err
	}
	*a = AffectedNode{
		LedgerEntryType:   affected.LedgerEntryType,
		LedgerIndex:       affected.LedgerIndex,
		PreviousTxnID:     affected.PreviousTxnID,
		PreviousTxnLgrSeq: affected.PreviousTxnLgrSeq,
	}
	if affected.FinalFields != nil {
		a.FinalFields = LedgerEntryFactory[a.LedgerEntryType]()
		if err := json.Unmarshal(affected.FinalFields, a.FinalFields); err != nil {
			return err
		}
	}
	if affected.PreviousFields != nil {
		a.PreviousFields = LedgerEntryFactory[a.LedgerEntryType]()
		if err := json.Unmarshal(affected.PreviousFields, a.PreviousFields); err != nil {
			return err
		}
	}
	if affected.NewFields != nil {
		a.NewFields = LedgerEntryFactory[a.LedgerEntryType]()
		if err := json.Unmarshal(affected.NewFields, a.NewFields); err != nil {
			return err
		}
	}
	return nil
}

const leTypeFormat = `:{"LedgerEntryType":"%s",`

func (a *AffectedNode) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(struct {
		AffectedNode
	}{*a})
	// I can only apologise
	fixed := strings.Replace(string(b), fmt.Sprintf(leTypeFormat, a.LedgerEntryType), ":{", -1)
	return []byte(fixed), err
}

func (i NodeIndex) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%016X", i)), nil
}

func (i *NodeIndex) UnmarshalText(b []byte) error {
	n, err := strconv.ParseUint(string(b), 16, 64)
	*i = NodeIndex(n)
	return err
}

func (e ExchangeRate) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%016X", e)), nil
}

func (e *ExchangeRate) UnmarshalText(b []byte) error {
	n, err := strconv.ParseUint(string(b), 16, 64)
	*e = ExchangeRate(n)
	return err
}

func (r TransactionResult) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}

func (r *TransactionResult) UnmarshalText(b []byte) error {
	if result, ok := reverseResults[string(b)]; ok {
		*r = result
		return nil
	}
	return fmt.Errorf("Unknown TransactionResult: %s", string(b))
}

func (l LedgerEntryType) MarshalText() ([]byte, error) {
	return []byte(ledgerEntryNames[l]), nil
}

func (l *LedgerEntryType) UnmarshalText(b []byte) error {
	if leType, ok := ledgerEntryTypes[string(b)]; ok {
		*l = leType
		return nil
	}
	// If here, add tx type to TxFactory and TxTypes in factory.go
	return fmt.Errorf("Unknown LedgerEntryType: %s", string(b))
}

func (t TransactionType) MarshalText() ([]byte, error) {
	return []byte(txNames[t]), nil
}

func (t *TransactionType) UnmarshalText(b []byte) error {
	if txType, ok := txTypes[string(b)]; ok {
		*t = txType
		return nil
	}
	// If here, add tx type to TxFactory and TxTypes in factory.go
	return fmt.Errorf("Unknown TransactionType: %s", string(b))
}

func (t RippleTime) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatUint(uint64(t.Uint32()), 10)), nil
}

func (t *RippleTime) UnmarshalJSON(b []byte) error {
	n, err := strconv.ParseUint(string(b), 10, 32)
	if err != nil {
		return err
	}
	t.SetUint32(uint32(n))
	return nil
}

func (t rippleHumanTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.String() + `"`), nil
}

func (t *rippleHumanTime) UnmarshalJSON(b []byte) error {
	return t.SetString(string(b[1 : len(b)-1]))
}

func (v *Value) MarshalText() ([]byte, error) {
	if v.IsNative() {
		num := strconv.FormatUint(v.num, 10)
		if v.IsNegative() {
			num = "-" + num
		}
		return []byte(num), nil
	}
	return []byte(v.String()), nil
}

func (v *Value) UnmarshalText(b []byte) error {
	value, err := NewValue(string(b), true)
	if err != nil {
		return err
	}
	*v = *value
	return nil
}

type NonNativeValue struct {
	Value
}

func (v *NonNativeValue) UnmarshalText(b []byte) error {
	value, err := NewValue(string(b), false)
	if err != nil {
		return err
	}
	v.Value = *value
	return nil
}

type amountJSON struct {
	Value    *NonNativeValue `json:"value"`
	Currency Currency        `json:"currency"`
	Issuer   Account         `json:"issuer"`
}

func (a Amount) MarshalJSON() ([]byte, error) {
	if a.Value == nil {
		return nil, fmt.Errorf("Value has a nil Value")
	}
	if a.IsNative() {
		return []byte(`"` + strconv.FormatUint(a.num, 10) + `"`), nil
	}
	return json.Marshal(amountJSON{&NonNativeValue{*a.Value}, a.Currency, a.Issuer})
}

func (a *Amount) UnmarshalJSON(b []byte) (err error) {
	if b[0] != '{' {
		a.Value = new(Value)
		return json.Unmarshal(b, a.Value)
	}
	var dummy amountJSON
	if err := json.Unmarshal(b, &dummy); err != nil {
		return err
	}
	a.Value, a.Currency, a.Issuer = &dummy.Value.Value, dummy.Currency, dummy.Issuer
	return nil
}

func (c Currency) MarshalText() ([]byte, error) {
	return []byte(c.Machine()), nil
}

func (c *Currency) UnmarshalText(text []byte) error {
	var err error
	*c, err = NewCurrency(string(text))
	return err
}

func (h Hash128) MarshalText() ([]byte, error) {
	return b2h(h[:]), nil
}

func (h *Hash128) UnmarshalText(b []byte) error {
	_, err := hex.Decode(h[:], b)
	return err
}

func (h Hash160) MarshalText() ([]byte, error) {
	return b2h(h[:]), nil
}

func (h *Hash160) UnmarshalText(b []byte) error {
	_, err := hex.Decode(h[:], b)
	return err
}

func (h Hash256) MarshalText() ([]byte, error) {
	return b2h(h[:]), nil
}

func (h *Hash256) UnmarshalText(b []byte) error {
	_, err := hex.Decode(h[:], b)
	return err
}

func (a Account) MarshalText() ([]byte, error) {
	address, err := a.Hash()
	if err != nil {
		return nil, err
	}
	return address.MarshalText()
}

// Expects base58-encoded account id
func (a *Account) UnmarshalText(b []byte) error {
	account, err := NewAccountFromAddress(string(b))
	if err != nil {
		return err
	}
	copy(a[:], account[:])
	return nil
}

func (r RegularKey) MarshalText() ([]byte, error) {
	address, err := r.Hash()
	if err != nil {
		return nil, err
	}
	return address.MarshalText()
}

// Expects base58-encoded account id
func (r *RegularKey) UnmarshalText(b []byte) error {
	account, err := NewRegularKeyFromAddress(string(b))
	if err != nil {
		return err
	}
	copy(r[:], account[:])
	return nil
}

func (s Seed) MarshalText() ([]byte, error) {
	address, err := s.Hash()
	if err != nil {
		return nil, err
	}
	return address.MarshalText()
}

// Expects base58-encoded account id
func (s *Seed) UnmarshalText(b []byte) error {
	account, err := NewSeedFromAddress(string(b))
	if err != nil {
		return err
	}
	copy(s[:], account[:])
	return nil
}

func (v VariableLength) MarshalText() ([]byte, error) {
	return b2h(v), nil
}

// Expects variable length hex
func (v *VariableLength) UnmarshalText(b []byte) error {
	var err error
	*v, err = hex.DecodeString(string(b))
	return err
}

func (p PublicKey) MarshalText() ([]byte, error) {
	if p.IsZero() {
		return []byte{}, nil
	}
	return b2h(p[:]), nil
}

// Expects public key hex
func (p *PublicKey) UnmarshalText(b []byte) error {
	_, err := hex.Decode(p[:], b)
	return err
}

// A uint64 which gets represented as a hex string in json
type Uint64Hex uint64

func (h Uint64Hex) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%0.16X", h)), nil
}

func (h *Uint64Hex) UnmarshalText(b []byte) error {
	_, err := fmt.Sscanf(string(b), "%X", h)
	return err
}

func (keyType KeyType) MarshalText() ([]byte, error) {
	return []byte(keyType.String()), nil
}
