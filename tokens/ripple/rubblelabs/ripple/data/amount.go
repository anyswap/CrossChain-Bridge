package data

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/crypto"
)

type Amount struct {
	*Value
	Currency Currency
	Issuer   Account
}

type ExchangeRate uint64

func newAmount(value *Value, currency Currency, issuer Account) *Amount {
	return &Amount{
		Value:    value,
		Currency: currency,
		Issuer:   issuer,
	}
}

// Requires v to be in computer parsable form
func NewAmount(v interface{}) (*Amount, error) {
	switch n := v.(type) {
	case int64:
		return &Amount{
			Value: newValue(true, n < 0, abs(n), 0),
		}, nil
	case string:
		var err error
		amount := new(Amount)
		parts := strings.Split(strings.TrimSpace(n), "/")
		native := false
		switch {
		case len(parts) == 1:
			native = true
		case len(parts) > 1 && parts[1] == "XRP":
			native = true
			if !strings.Contains(parts[0], ".") {
				parts[0] = parts[0] + "."
			}
		}
		if amount.Value, err = NewValue(parts[0], native); err != nil {
			return nil, err
		}
		if len(parts) > 1 {
			if amount.Currency, err = NewCurrency(parts[1]); err != nil {
				return nil, err
			}
		}
		if len(parts) > 2 {
			if issuer, err := crypto.NewRippleHash(parts[2]); err != nil {
				return nil, err
			} else {
				copy(amount.Issuer[:], issuer.Payload())
			}
		}
		return amount, nil
	default:
		return nil, fmt.Errorf("Bad type: %+v", v)
	}
}

func (a Amount) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	err := a.Marshal(&buf)
	return buf.Bytes(), err
}

func (a *Amount) UnmarshalBinary(b []byte) error {
	buf := bytes.NewBuffer(b)
	return a.Unmarshal(buf)
}

func (a Amount) Equals(b Amount) bool {
	return a.Value.Equals(*b.Value) &&
		a.Currency == b.Currency &&
		a.Issuer == b.Issuer
}

// Returns true if the values are equal, but ignores the currency and issuer
func (a Amount) SameValue(b *Amount) bool {
	return a.Value.Equals(*b.Value)
}

func (a Amount) Clone() *Amount {
	return newAmount(a.Value.Clone(), a.Currency, a.Issuer)
}

// Returns a new Amount with the same currency and issuer, but a zero value
func (a Amount) ZeroClone() *Amount {
	return newAmount(a.Value.ZeroClone(), a.Currency, a.Issuer)
}

func (a Amount) IsPositive() bool {
	return !a.negative
}

func (a Amount) Negate() *Amount {
	clone := a.Clone()
	clone.negative = !clone.negative
	return clone
}

func (a Amount) Abs() *Amount {
	clone := a.Clone()
	clone.negative = false
	return clone
}

func (a Amount) Add(b *Amount) (*Amount, error) {
	sum, err := a.Value.Add(*b.Value)
	if err != nil {
		return nil, err
	}
	return newAmount(sum, a.Currency, a.Issuer), nil
}

func (a Amount) Subtract(b *Amount) (*Amount, error) {
	return a.Add(b.Negate())
}

func (a Amount) multiply(b *Amount) (*Amount, error) {
	product, err := a.Value.Multiply(*b.Value)
	if err != nil {
		return nil, err
	}
	return newAmount(product, a.Currency, a.Issuer), nil
}

func (num Amount) divide(den *Amount) (*Amount, error) {
	quotient, err := num.Value.Divide(*den.Value)
	if err != nil {
		return nil, err
	}
	return newAmount(quotient, num.Currency, num.Issuer), nil
}

func (a Amount) ApplyInterest() (*Amount, error) {
	if a.Currency.Type() != CT_DEMURRAGE {
		return &a, nil
	}
	rate := fmt.Sprintf("%f/%s/%s", a.Currency.Rate(Now().Uint32()), a.Currency.Machine(), a.Issuer.String())
	factor, err := NewAmount(rate)
	if err != nil {
		return nil, err
	}
	return a.multiply(factor)
}

type amountFunc func(Amount, *Amount) (*Amount, error)

func applyInterestPair(a Amount, b *Amount, f amountFunc) (*Amount, error) {
	v1, err := a.ApplyInterest()
	if err != nil {
		return nil, err
	}
	v2, err := b.ApplyInterest()
	if err != nil {
		return nil, err
	}
	return f(*v1, v2)
}

func (a Amount) Divide(b *Amount) (*Amount, error) {
	return applyInterestPair(a, b, Amount.divide)
}

func (a Amount) Multiply(b *Amount) (*Amount, error) {
	return applyInterestPair(a, b, Amount.multiply)
}

// Ratio returns the ratio between a and b.
// Returns a zero value when division is impossible
func (a Amount) Ratio(b Amount) *Value {
	ratio, err := a.Value.Ratio(*b.Value)
	switch {
	case err == nil:
		return ratio
	case a.IsNative():
		return &zeroNative
	default:
		return &zeroNonNative
	}
}

func (a Amount) Bytes() []byte {
	if a.IsNative() {
		return a.Value.Bytes()
	}
	return append(a.Value.Bytes(), append(a.Currency.Bytes(), a.Issuer.Bytes()...)...)
}

// Amount in human parsable form
// with demurrage applied
func (a Amount) String() string {
	factored, err := a.ApplyInterest()
	if err != nil {
		return err.Error()
	}
	switch {
	case a.IsNative():
		return factored.Value.String() + "/XRP"
	case a.Issuer.IsZero():
		return factored.Value.String() + "/" + a.Currency.String()
	default:
		return factored.Value.String() + "/" + a.Currency.String() + "/" + a.Issuer.String()
	}
}

// Amount in computer parsable form
func (a Amount) Machine() string {
	switch {
	case a.IsNative():
		return a.Value.String() + "/XRP"
	case a.Issuer.IsZero():
		return a.Value.String() + "/" + a.Currency.Machine()
	default:
		return a.Value.String() + "/" + a.Currency.Machine() + "/" + a.Issuer.String()
	}
}

func (a Amount) Asset() *Asset {
	switch {
	case a.IsNative():
		return &Asset{
			Currency: "XRP",
		}
	default:
		return &Asset{
			Currency: a.Currency.String(),
			Issuer:   a.Issuer.String(),
		}
	}
}

func NewExchangeRate(a, b *Amount) (ExchangeRate, error) {
	if b.IsZero() {
		return 0, nil
	}
	rate, err := a.Divide(b)
	if err != nil {
		return 0, err
	}
	if rate.IsZero() {
		return 0, nil
	}
	if rate.offset >= -100 || rate.offset <= 155 {
		panic("Impossible Rate")
	}
	return ExchangeRate(uint64(rate.offset+100)<<54 | uint64(rate.num)), nil
}

func (e *ExchangeRate) Bytes() []byte {
	if e == nil {
		return nil
	}
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(*e))
	return b
}
