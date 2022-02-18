package data

import (
	"bytes"
	"crypto/sha512"
	"fmt"
	"math"
)

type NodeIndex uint64

func (i *NodeIndex) Previous() *NodeIndex {
	if i == nil || *i == 0 {
		return nil
	}
	prev := *i - 1
	return &prev
}

func (i *NodeIndex) Next() *NodeIndex {
	if i == nil || *i == math.MaxUint64 {
		return nil
	}
	next := *i + 1
	return &next
}

func LedgerIndex(le LedgerEntry) (*Hash256, error) {
	switch v := le.(type) {
	case *AccountRoot:
		return GetAccountRootIndex(*v.Account)
	case *RippleState:
		return GetRippleStateIndex(v.LowLimit.Issuer, v.HighLimit.Issuer, v.Balance.Currency)
	case *Offer:
		return GetOfferIndex(*v.Account, *v.Sequence)
	case *LedgerHashes:
		return GetLedgerHashIndex()
	case *Directory:
		return GetDirectoryNodeIndex(*v.RootIndex, v.IndexPrevious.Next())
	case *FeeSettings:
		return buildIndex([]interface{}{NS_FEE})
	case *Amendments:
		return buildIndex([]interface{}{NS_AMENDMENT})
	default:
		return nil, fmt.Errorf("Unknown LedgerEntry")
	}
}

func GetAccountRootIndex(account Account) (*Hash256, error) {
	return buildIndex([]interface{}{NS_ACCOUNT, account.Bytes()})
}

func GetOfferIndex(account Account, sequence uint32) (*Hash256, error) {
	return buildIndex([]interface{}{NS_OFFER, account.Bytes(), sequence})
}

func GetRippleStateIndex(a, b Account, c Currency) (*Hash256, error) {
	if bytes.Compare(a.Bytes(), b.Bytes()) < 0 {
		return buildIndex([]interface{}{NS_RIPPLE_STATE, a.Bytes(), b.Bytes(), c.Bytes()})
	}
	return buildIndex([]interface{}{NS_RIPPLE_STATE, b.Bytes(), a.Bytes(), c.Bytes()})
}

func GetDirectoryNodeIndex(root Hash256, index *NodeIndex) (*Hash256, error) {
	if index == nil {
		return &root, nil
	}
	return buildIndex([]interface{}{NS_DIRECTORY_NODE, root, *index})
}

func GetOwnerDirectoryIndex(account Account) (*Hash256, error) {
	return buildIndex([]interface{}{NS_OWNER_DIRECTORY, account.Bytes()})
}

func GetBookIndex(paysCurrency, getsCurrency Hash160, paysIssuer, getsIssuer Hash160) (*Hash256, error) {
	//TODO: change types to Currency and Account
	index, err := buildIndex([]interface{}{NS_BOOK_DIRECTORY, paysCurrency.Bytes(), getsCurrency.Bytes(), paysCurrency.Bytes(), getsCurrency.Bytes()})
	if err != nil {
		return nil, err
	}
	var zero [8]byte
	copy(index[24:], zero[:])
	return index, nil
}

func GetFeeIndex() (*Hash256, error) {
	return buildIndex([]interface{}{NS_FEE})
}

func GetAmendmentsIndex() (*Hash256, error) {
	return buildIndex([]interface{}{NS_AMENDMENT})
}

func GetLedgerHashIndex() (*Hash256, error) {
	return buildIndex([]interface{}{NS_SKIP_LIST})
}

func GetPreviousLedgerHashIndex(sequence uint32) (*Hash256, error) {
	return buildIndex([]interface{}{NS_SKIP_LIST, sequence >> 16})
}

func buildIndex(items []interface{}) (*Hash256, error) {
	index := sha512.New()
	for _, item := range items {
		if err := write(index, item); err != nil {
			return nil, err
		}
	}
	return NewHash256(index.Sum(nil)[:32])
}
