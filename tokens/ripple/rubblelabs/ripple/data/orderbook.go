package data

import (
	"fmt"
	"sort"
	"strings"
)

type Asset struct {
	Currency string `json:"currency"`
	Issuer   string `json:"issuer,omitempty"`
}

func NewAsset(s string) (*Asset, error) {
	if s == "XRP" {
		return &Asset{
			Currency: s,
		}, nil
	}
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("bad asset: %s", s)
	}
	return &Asset{
		Currency: parts[0],
		Issuer:   parts[1],
	}, nil
}

func (a *Asset) IsNative() bool {
	return a.Currency == "XRP"
}

func (a *Asset) Matches(amount *Amount) bool {
	return (a.IsNative() && amount.IsNative()) ||
		(a.Currency == amount.Currency.String() && a.Issuer == amount.Issuer.String())
}

func (a Asset) String() string {
	if a.IsNative() {
		return a.Currency
	}
	return fmt.Sprintf("%s/%s", a.Currency, a.Issuer)
}

type OrderBookOffer struct {
	Offer
	OwnerFunds      Value          `json:"owner_funds"`
	Quality         NonNativeValue `json:"quality"`
	TakerGetsFunded *Amount        `json:"taker_gets_funded"`
	TakerPaysFunded *Amount        `json:"taker_pays_funded"`
}

type AccountOffer struct {
	Flags      LedgerEntryFlag `json:"flags"`
	Quality    NonNativeValue  `json:"quality"`
	Sequence   uint32          `json:"seq"`
	TakerGets  Amount          `json:"taker_gets"`
	TakerPays  Amount          `json:"taker_pays"`
	Expiration *uint32         `json:"expiration"`
}

type AccountOfferSlice []AccountOffer

func (s AccountOfferSlice) Len() int           { return len(s) }
func (s AccountOfferSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s AccountOfferSlice) Less(i, j int) bool { return s[i].Sequence > s[j].Sequence }

func (s AccountOfferSlice) GetSequences(pays, gets *Asset) []uint32 {
	// TODO: improve performance
	var sequences []uint32
	for i := range s {
		if pays.Matches(&s[i].TakerPays) && gets.Matches(&s[i].TakerGets) {
			sequences = append(sequences, s[i].Sequence)
		}
	}
	return sequences
}

func (s AccountOfferSlice) Find(sequence uint32) int {
	return sort.Search(len(s), func(i int) bool {
		return s[i].Sequence <= sequence
	})
}

func (s AccountOfferSlice) Get(sequence uint32) *AccountOffer {
	if i := s.Find(sequence); i < len(s) && s[i].Sequence == sequence {
		return &s[i]
	}
	return nil
}

func defaultUint32(v *uint32) uint32 {
	if v == nil {
		return 0
	}
	return *v
}

func (s *AccountOfferSlice) Add(offer *Offer) bool {
	quality, err := offer.TakerPays.Divide(offer.TakerGets)
	if err != nil {
		panic(fmt.Sprintf("impossible quality: %s %s", offer.TakerPays, offer.TakerGets))
	}
	o := AccountOffer{
		Flags:      LedgerEntryFlag(defaultUint32((*uint32)(offer.Flags))),
		Sequence:   *offer.Sequence,
		TakerGets:  *offer.TakerGets,
		TakerPays:  *offer.TakerPays,
		Quality:    NonNativeValue{*quality.Value},
		Expiration: offer.Expiration,
	}
	i := s.Find(*offer.Sequence)
	switch {
	case i == len(*s):
		*s = append(*s, o)
		return true
	case (*s)[i].Sequence != *offer.Sequence:
		*s = append(*s, AccountOffer{})
		copy((*s)[i+1:], (*s)[i:])
		(*s)[i] = o
		return true
	default:
		return false
	}
}

func (s AccountOfferSlice) Update(offer *Offer) bool {
	if existing := s.Get(*offer.Sequence); existing != nil {
		existing.TakerGets = *offer.TakerGets
		existing.TakerPays = *offer.TakerPays
		return true
	}
	return false
}

func (s *AccountOfferSlice) Delete(offer *Offer) bool {
	if i := s.Find(*offer.Sequence); i < len(*s) && (*s)[i].Sequence == *offer.Sequence {
		*s = append((*s)[:i], (*s)[i+1:]...)
		return true
	}
	return false
}

type AccountLine struct {
	Account      Account        `json:"account"`
	Balance      NonNativeValue `json:"balance"`
	Currency     Currency       `json:"currency"`
	Limit        NonNativeValue `json:"limit"`
	LimitPeer    NonNativeValue `json:"limit_peer"`
	NoRipple     bool           `json:"no_ripple"`
	NoRipplePeer bool           `json:"no_ripple_peer"`
	QualityIn    uint32         `json:"quality_in"`
	QualityOut   uint32         `json:"quality_out"`
}

func (l *AccountLine) Asset() *Asset {
	return &Asset{
		Currency: l.Currency.String(),
		Issuer:   l.Account.String(),
	}
}

func (l *AccountLine) CompareByCurrencyAccount(other *AccountLine) int {
	if cmp := l.Currency.Compare(other.Currency); cmp != 0 {
		return cmp
	}
	return l.Account.Compare(other.Account)
}

func (l *AccountLine) CompareByCurrencyAmount(other *AccountLine) int {
	if cmp := l.Currency.Compare(other.Currency); cmp != 0 {
		return cmp
	}
	return l.Balance.Abs().Compare(*other.Balance.Abs())
}

type AccountLineSlice []AccountLine

type byCurrencyAccount struct {
	AccountLineSlice
}

func (by *byCurrencyAccount) Less(i, j int) bool {
	return by.AccountLineSlice[i].CompareByCurrencyAccount(&by.AccountLineSlice[j]) < 0
}

type byCurrencyAmount struct {
	AccountLineSlice
}

func (by *byCurrencyAmount) Less(i, j int) bool {
	return by.AccountLineSlice[i].CompareByCurrencyAmount(&by.AccountLineSlice[j]) < 0
}

func (s AccountLineSlice) Len() int               { return len(s) }
func (s AccountLineSlice) Swap(i, j int)          { s[i], s[j] = s[j], s[i] }
func (s AccountLineSlice) SortbyCurrencyAccount() { sort.Sort(&byCurrencyAccount{s}) }
func (s AccountLineSlice) SortByCurrencyAmount()  { sort.Sort(&byCurrencyAmount{s}) }

func (s AccountLineSlice) Find(account Account, currency Currency) int {
	return sort.Search(len(s), func(i int) bool {
		switch s[i].Currency.Compare(currency) {
		case 1:
			return true
		case -1:
			return false
		default:
			return s[i].Account.Compare(account) >= 0
		}
	})
}

func (s AccountLineSlice) Get(account Account, currency Currency) *AccountLine {
	if i := s.Find(account, currency); i < len(s) && s[i].Account.Equals(account) && s[i].Currency.Equals(currency) {
		return &s[i]
	}
	return nil
}

type highLowFunc func(balance, limit, limitPeer *Amount, noRipple, noRipplePeer bool, qualityIn, qualityOut uint32) bool

func highLow(account Account, rs *RippleState, f highLowFunc) bool {
	high, low := account.Equals(rs.HighLimit.Issuer), account.Equals(rs.LowLimit.Issuer)
	switch {
	case high == low:
		return false
	case high:
		noRipple, noRipplePeer := *rs.Flags&LsHighNoRipple > 0, *rs.Flags&LsLowNoRipple > 0
		return f(rs.Balance.Negate(), rs.HighLimit, rs.LowLimit, noRipple, noRipplePeer, defaultUint32(rs.HighQualityIn), defaultUint32(rs.HighQualityOut))
	default:
		noRipple, noRipplePeer := *rs.Flags&LsLowNoRipple > 0, *rs.Flags&LsHighNoRipple > 0
		return f(rs.Balance, rs.LowLimit, rs.HighLimit, noRipple, noRipplePeer, defaultUint32(rs.LowQualityIn), defaultUint32(rs.LowQualityOut))
	}
}

func (s *AccountLineSlice) Add(account Account, rs *RippleState) bool {
	var add = func(balance, limit, limitPeer *Amount, noRipple, noRipplePeer bool, qualityIn, qualityOut uint32) bool {
		line := AccountLine{
			Account:      limitPeer.Issuer,
			Balance:      NonNativeValue{*balance.Value},
			Currency:     limit.Currency,
			Limit:        NonNativeValue{*limit.Value},
			LimitPeer:    NonNativeValue{*limitPeer.Value},
			NoRipple:     noRipple,
			NoRipplePeer: noRipplePeer,
			QualityIn:    qualityIn,
			QualityOut:   qualityOut,
		}
		i := s.Find(limitPeer.Issuer, limitPeer.Currency)
		switch {
		case i == len(*s):
			*s = append(*s, line)
			return true
		case (*s)[i].Account.Equals(limitPeer.Issuer) && (*s)[i].Currency.Equals(limitPeer.Currency):
			return false
		default:
			*s = append((*s)[:i], append(AccountLineSlice{line}, (*s)[i:]...)...)
			return true
		}
	}
	return highLow(account, rs, add)
}

func (s AccountLineSlice) Update(account Account, rs *RippleState) bool {
	var update = func(balance, limit, limitPeer *Amount, noripple, noRipplePeer bool, qualityIn, qualityOut uint32) bool {
		i := s.Find(limitPeer.Issuer, limitPeer.Currency)
		if i == len(s) || !s[i].Account.Equals(limitPeer.Issuer) || !s[i].Currency.Equals(limitPeer.Currency) {
			return false
		}
		s[i].Balance = NonNativeValue{*balance.Value}
		s[i].Limit = NonNativeValue{*limit.Value}
		s[i].LimitPeer = NonNativeValue{*limitPeer.Value}
		s[i].NoRipple = noripple
		s[i].NoRipplePeer = noRipplePeer
		return true
	}
	return highLow(account, rs, update)
}

func (s *AccountLineSlice) Delete(account Account, rs *RippleState) bool {
	var del = func(balance, limit, limitPeer *Amount, noripple, noRipplePeer bool, qualityIn, qualityOut uint32) bool {
		i := s.Find(limitPeer.Issuer, limitPeer.Currency)
		if i < len(*s) && (*s)[i].Account.Equals(limitPeer.Issuer) && (*s)[i].Currency.Equals(limitPeer.Currency) {
			*s = append((*s)[:i], (*s)[i+1:]...)
			return true
		}
		return false
	}
	return highLow(account, rs, del)
}
