// Package config provides a simple way of signing submitting groups of transactions for the same account.
package config

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/data"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/websockets"
)

type Action struct {
	Seed         data.Seed
	Fee          data.Value
	KeyType      data.KeyType
	AccountSets  []data.AccountSet
	TrustSets    []data.TrustSet
	OfferCreates []data.OfferCreate
	Payments     []data.Payment
}

type actionFunc func(seed data.Seed, fee data.Value, keyType data.KeyType, tx data.Transaction, txType data.TransactionType) error

func (a *Action) each(f actionFunc) error {
	for i := range a.AccountSets {
		if err := f(a.Seed, a.Fee, a.KeyType, &a.AccountSets[i], data.ACCOUNT_SET); err != nil {
			return err
		}
	}
	for i := range a.TrustSets {
		if err := f(a.Seed, a.Fee, a.KeyType, &a.TrustSets[i], data.TRUST_SET); err != nil {
			return err
		}
	}
	for i := range a.OfferCreates {
		if err := f(a.Seed, a.Fee, a.KeyType, &a.OfferCreates[i], data.OFFER_CREATE); err != nil {
			return err
		}
	}
	for i := range a.Payments {
		if err := f(a.Seed, a.Fee, a.KeyType, &a.Payments[i], data.PAYMENT); err != nil {
			return err
		}
	}
	return nil
}

type ActionSlice []Action

func Parse(r io.Reader) (ActionSlice, error) {
	var actions []Action
	if err := json.NewDecoder(r).Decode(&actions); err != nil {
		return nil, err
	}
	return actions, nil
}

func (s ActionSlice) each(f actionFunc) error {
	for i := range s {
		if err := s[i].each(f); err != nil {
			return err
		}
	}
	return nil
}

func (s ActionSlice) Prepare() error {
	var prepare = func(seed data.Seed, fee data.Value, keyType data.KeyType, tx data.Transaction, txType data.TransactionType) error {
		var (
			sequence uint32
			key      = seed.Key(keyType)
			base     = tx.GetBase()
		)
		base.TransactionType = txType
		base.Fee = fee
		base.Account = seed.AccountId(keyType, &sequence)
		return data.Sign(tx, key, &sequence)
	}
	return s.each(prepare)
}

func (s ActionSlice) Submit(host string) error {
	remote, err := websockets.NewRemote(host)
	if err != nil {
		return err
	}
	var submit = func(seed data.Seed, fee data.Value, keyType data.KeyType, tx data.Transaction, txType data.TransactionType) error {
		result, err := remote.Submit(tx)
		if err != nil {
			return err
		}
		if !result.EngineResult.Success() {
			return fmt.Errorf("%s\n%s", result.EngineResultMessage, js(tx))
		}
		return nil
	}
	return s.each(submit)
}

func (s ActionSlice) Count() int {
	var count int
	s.each(func(seed data.Seed, fee data.Value, keyType data.KeyType, tx data.Transaction, txType data.TransactionType) error {
		count++
		return nil
	})
	return count
}

func (s ActionSlice) String() string {
	return js(s)
}

func js(v interface{}) string {
	out, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return err.Error()
	}
	return string(out)
}
