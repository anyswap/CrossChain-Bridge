package data

import (
	"fmt"
	"sort"
)

type Trade struct {
	LedgerSequence   uint32
	TransactionIndex uint32
	TransactionType  string
	Op               string
	Paid             *Amount
	Got              *Amount
	Giver            Account
	Taker            Account
}

func newTrade(txm *TransactionWithMetaData, i int) (*Trade, error) {
	_, final, previous, action := txm.MetaData.AffectedNodes[i].AffectedNode()
	v, ok := final.(*Offer)
	if !ok || action == Created {
		return nil, nil
	}
	p := previous.(*Offer)
	if p != nil && p.TakerGets == nil || p.TakerPays == nil {
		// Some "micro" offer consumptions don't change both balances!
		return nil, nil
	}
	paid, err := p.TakerPays.Subtract(v.TakerPays)
	if err != nil {
		return nil, err
	}
	got, err := p.TakerGets.Subtract(v.TakerGets)
	if err != nil {
		return nil, err
	}
	trade := &Trade{
		LedgerSequence:   txm.LedgerSequence,
		TransactionIndex: txm.MetaData.TransactionIndex,
		TransactionType:  txm.GetTransactionType().String(),
		Op:               "Modify",
		Paid:             paid,
		Got:              got,
		Giver:            *v.Account,
		Taker:            txm.Transaction.GetBase().Account,
	}
	if action == Deleted {
		trade.Op = "Delete"
	}
	return trade, nil
}

func (t *Trade) Rate() float64 {
	return t.Got.Ratio(*t.Paid).Float()
}

func (t Trade) String() string {
	return fmt.Sprintf("%8d %3d %22.8f %22.8f %-38s %22.8f %-38s %34s %34s %11s %s", t.LedgerSequence, t.TransactionIndex, t.Rate(), t.Paid.Float(), t.Paid.Asset(), t.Got.Float(), t.Got.Asset(), t.Taker, t.Giver, t.TransactionType, t.Op)

}

type TradeSlice []Trade

func NewTradeSlice(txm *TransactionWithMetaData) (TradeSlice, error) {
	var trades TradeSlice
	for i := range txm.MetaData.AffectedNodes {
		trade, err := newTrade(txm, i)
		if err != nil {
			return nil, err
		}
		if trade != nil {
			trades = append(trades, *trade)
		}
	}
	trades.Sort()
	return trades, nil
}

func (s TradeSlice) Filter(account Account) TradeSlice {
	var trades TradeSlice
	for i := range s {
		if s[i].Taker.Equals(account) || s[i].Giver.Equals(account) {
			trades = append(trades, s[i])
		}
	}
	return trades
}

func (s TradeSlice) Sort()         { sort.Sort(s) }
func (s TradeSlice) Len() int      { return len(s) }
func (s TradeSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s TradeSlice) Less(i, j int) bool {
	if s[i].LedgerSequence == s[j].LedgerSequence {
		if s[i].TransactionIndex == s[j].TransactionIndex {
			if s[i].Got.Currency.Equals(s[j].Got.Currency) {
				if s[i].Got.Issuer.Equals(s[j].Got.Issuer) {
					if s[i].Paid.Currency.Equals(s[j].Paid.Currency) {
						if s[i].Paid.Issuer.Equals(s[j].Paid.Issuer) {
							return s[i].Rate() > s[j].Rate()
						}
						return s[i].Paid.Issuer.Less(s[j].Paid.Issuer)
					}
					return s[i].Paid.Currency.Less(s[j].Paid.Currency)
				}
				return s[i].Got.Issuer.Less(s[j].Got.Issuer)
			}
			return s[i].Got.Currency.Less(s[j].Got.Currency)
		}
		return s[i].TransactionIndex < s[j].TransactionIndex
	}
	return s[i].LedgerSequence < s[j].LedgerSequence
}
