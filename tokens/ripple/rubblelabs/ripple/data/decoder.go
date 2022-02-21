package data

import (
	"errors"
	"fmt"
	"reflect"
)

// ReadWire parses types received via the peer network
func ReadWire(r Reader, typ NodeType, ledgerSequence uint32, nodeId Hash256) (Hashable, error) {
	version, err := readHashPrefix(r)
	if err != nil {
		return nil, err
	}
	switch version {
	case HP_LEAF_NODE:
		return ReadLedgerEntry(r, nodeId)
	case HP_TRANSACTION_NODE:
		return readTransactionWithMetadata(r, ledgerSequence, nodeId)
	case HP_INNER_NODE:
		return readCompressedInnerNode(r, typ, nodeId)
	default:
		return nil, fmt.Errorf("Unknown hash prefix: %s", version.String())
	}
}

// ReadPrefix parses types received from the nodestore
func ReadPrefix(r Reader, nodeId Hash256) (Storer, error) {
	header, err := readHeader(r)
	if err != nil {
		return nil, err
	}
	version, err := readHashPrefix(r)
	if err != nil {
		return nil, err
	}
	switch {
	case version == HP_INNER_NODE:
		return readInnerNode(r, header.NodeType, nodeId)
	case header.NodeType == NT_LEDGER:
		return ReadLedger(r, nodeId)
	case header.NodeType == NT_TRANSACTION_NODE:
		return readTransactionWithMetadata(r, header.LedgerSequence, nodeId)
	case header.NodeType == NT_ACCOUNT_NODE:
		return ReadLedgerEntry(r, nodeId)
	default:
		return nil, fmt.Errorf("Unknown node type")
	}
}

func ReadLedger(r Reader, nodeId Hash256) (*Ledger, error) {
	ledger := new(Ledger)
	if err := read(r, &ledger.LedgerHeader); err != nil {
		return nil, err
	}
	ledger.Hash = nodeId
	return ledger, nil
}

func ReadValidation(r Reader) (*Validation, error) {
	validation := new(Validation)
	v := reflect.ValueOf(validation)
	if err := readObject(r, &v); err != nil {
		return nil, err
	}
	return validation, nil
}

func ReadTransaction(r Reader) (Transaction, error) {
	txType, err := expectType(r, "TransactionType")
	if err != nil {
		return nil, err
	}
	tx := TxFactory[txType]()
	v := reflect.ValueOf(tx)
	if err := readObject(r, &v); err != nil {
		return nil, err
	}
	return tx, nil
}

// ReadTransactionAndMetadata combines the inputs from the two
// readers into a TransactionWithMetaData
func ReadTransactionAndMetadata(tx, meta Reader, hash Hash256, ledger uint32) (*TransactionWithMetaData, error) {
	t, err := ReadTransaction(tx)
	if err != nil {
		return nil, err
	}
	txm := &TransactionWithMetaData{
		Transaction:    t,
		LedgerSequence: ledger,
	}
	m := reflect.ValueOf(&txm.MetaData)
	if err := readObject(meta, &m); err != nil {
		return nil, err
	}
	*txm.GetHash() = hash
	if txm.Id, err = NodeId(txm); err != nil {
		return nil, err
	}
	return txm, nil
}

// For internal use when reading Prefix format
func readTransactionWithMetadata(r Reader, ledger uint32, nodeId Hash256) (*TransactionWithMetaData, error) {
	br, err := NewVariableByteReader(r)
	if err != nil {
		return nil, err
	}
	tx, err := ReadTransaction(br)
	if err != nil {
		return nil, err
	}
	txm := &TransactionWithMetaData{
		Transaction:    tx,
		LedgerSequence: ledger,
		Id:             nodeId,
	}
	br, err = NewVariableByteReader(r)
	if err != nil {
		return nil, err
	}
	meta := reflect.ValueOf(&txm.MetaData)
	if err := readObject(br, &meta); err != nil {
		return nil, err
	}
	hash, err := readHash(r)
	if err != nil {
		return nil, err
	}
	copy(txm.GetHash()[:], hash.Bytes())
	return txm, nil
}

func readInnerNode(r Reader, typ NodeType, nodeId Hash256) (*InnerNode, error) {
	var inner InnerNode
	inner.Type = typ
	for i := range inner.Children {
		if _, err := r.Read(inner.Children[i][:]); err != nil {
			return nil, err
		}
	}
	copy(inner.Id[:], nodeId.Bytes())
	return &inner, nil
}

func readCompressedInnerNode(r Reader, typ NodeType, nodeId Hash256) (*InnerNode, error) {
	var inner InnerNode
	inner.Type = typ
	var entry CompressedNodeEntry
	for read(r, &entry) == nil {
		inner.Children[entry.Pos] = entry.Hash
	}
	copy(inner.Id[:], nodeId.Bytes())
	return &inner, nil
}

func ReadLedgerEntry(r Reader, nodeId Hash256) (LedgerEntry, error) {
	leType, err := expectType(r, "LedgerEntryType")
	if err != nil {
		return nil, err
	}
	le := LedgerEntryFactory[leType]()
	v := reflect.ValueOf(le)
	// LedgerEntries have 32 bytes of index suffixed
	// but don't have a variable bytes indicator
	lr := LimitedByteReader(r, int64(r.Len()-32))
	if err := readObject(lr, &v); err != nil {
		return nil, err
	}
	hash, err := readHash(r)
	if err != nil {
		return nil, err
	}
	copy(le.GetHash()[:], hash.Bytes())
	copy(le.NodeId()[:], nodeId.Bytes())
	return le, nil
}

func readHashPrefix(r Reader) (HashPrefix, error) {
	var version HashPrefix
	return version, read(r, &version)
}

func readHeader(r Reader) (*NodeHeader, error) {
	header := new(NodeHeader)
	return header, read(r, header)
}

func readHash(r Reader) (*Hash256, error) {
	var h Hash256
	n, err := r.Read(h[:])
	switch {
	case err != nil:
		return nil, err
	case n != len(h):
		return nil, fmt.Errorf("Bad hash")
	default:
		return &h, nil
	}
}

func expectType(r Reader, expected string) (uint16, error) {
	enc, err := readEncoding(r)
	if err != nil {
		return 0, err
	}
	name := encodings[*enc]
	if name != expected {
		return 0, fmt.Errorf("Unexpected type: %s expected: %s", name, expected)
	}
	var typ uint16
	return typ, read(r, &typ)
}

var (
	errorEndOfObject = errors.New("EndOfObject")
	errorEndOfArray  = errors.New("EndOfArray")
)

func readObject(r Reader, v *reflect.Value) error {
	var err error
	for enc, err := readEncoding(r); err == nil; enc, err = readEncoding(r) {
		name := encodings[*enc]
		// fmt.Println(name, v, v.IsValid(), enc.typ, enc.field)
		switch enc.typ {
		case ST_ARRAY:
			if name == "EndOfArray" {
				return errorEndOfArray
			}
			array := getField(v, enc)
		loop:
			for {
				child := reflect.New(array.Type().Elem()).Elem()
				err := readObject(r, &child)
				switch err {
				case errorEndOfArray:
					break loop
				case errorEndOfObject:
					array.Set(reflect.Append(*array, child))
				default:
					return err
				}
			}
		case ST_OBJECT:
			switch name {
			case "EndOfObject":
				return errorEndOfObject
			case "PreviousFields", "NewFields", "FinalFields":
				leType := LedgerEntryType(v.Elem().FieldByName("LedgerEntryType").Uint())
				le := LedgerEntryFactory[leType]()
				fields := reflect.ValueOf(le)
				v.Elem().FieldByName(name).Set(fields)
				if err := readObject(r, &fields); err != nil && err != errorEndOfObject {
					return err
				}
				// var fields Fields
				// f := reflect.ValueOf(&fields)
				// v.Elem().FieldByName(name).Set(f)
				// if readObject(r, &f); err != nil && err != errorEndOfObject {
				// 	return err
				// }
			case "ModifiedNode", "DeletedNode", "CreatedNode":
				var node AffectedNode
				n := reflect.ValueOf(&node)
				var effect NodeEffect
				e := reflect.ValueOf(&effect)
				e.Elem().FieldByName(name).Set(n)
				v.Set(e.Elem())
				return readObject(r, &n)
			case "SignerEntry":
				var signerEntry SignerEntry
				s := reflect.ValueOf(&signerEntry)
				err := readObject(r, &s)
				v.Set(s.Elem())
				return err
			case "Majority":
				var majority Majority
				m := reflect.ValueOf(&majority)
				err := readObject(r, &m)
				v.Set(m.Elem())
				return err
			case "Memo":
				var memo Memo
				m := reflect.ValueOf(&memo)
				inner := reflect.ValueOf(&memo.Memo)
				err := readObject(r, &inner)
				v.Set(m.Elem())
				return err
			default:
				return fmt.Errorf("Unexpected object: %s for field: %s", v.Type(), name)
			}
		default:
			if v.Kind() == reflect.Struct {
				return fmt.Errorf("Unexpected struct: %s for field: %s", v.Type(), name)
			}
			field := getField(v, enc)
			if !field.CanAddr() {
				return fmt.Errorf("Missing field: %s %+v", name, enc)
			}
			switch v := field.Addr().Interface().(type) {
			case Wire:
				if err := v.Unmarshal(r); err != nil {
					return err
				}
			default:
				if err := read(r, v); err != nil {
					return err
				}
			}
		}
	}
	return err
}

func getField(v *reflect.Value, e *enc) *reflect.Value {
	name := encodings[*e]
	field := v.Elem().FieldByName(name)
	if field.Kind() == reflect.Ptr {
		field.Set(reflect.New(field.Type().Elem()))
		field = field.Elem()
	}
	return &field
}
