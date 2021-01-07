package data

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
)

type Currency [20]byte
type CurrencyType uint8

const (
	CT_XRP       CurrencyType = 0
	CT_STANDARD  CurrencyType = 1
	CT_DEMURRAGE CurrencyType = 2
	CT_HEX       CurrencyType = 3
	CT_UNKNOWN   CurrencyType = 4
)

var zeroCurrency Currency

// Accepts currency as either a 3 character code
// or a 40 character hex string
func NewCurrency(s string) (Currency, error) {
	if s == "XRP" {
		return zeroCurrency, nil
	}
	var currency Currency
	switch len(s) {
	case 3:
		copy(currency[12:], []byte(s))
		return currency, nil
	case 40:
		c, err := hex.DecodeString(s)
		if err != nil {
			return currency, fmt.Errorf("Bad Currency: %s", s)
		}
		copy(currency[:], c)
		return currency, nil
	default:
		return currency, fmt.Errorf("Bad Currency: %s", s)
	}
}

func (a Currency) Compare(b Currency) int {
	return bytes.Compare(a[:], b[:])
}

func (a Currency) Less(b Currency) bool {
	return a.Compare(b) < 0
}

func (c Currency) Equals(other Currency) bool {
	return c == other
}

func (c Currency) Clone() Currency {
	var n Currency
	copy(n[:], c[:])
	return n
}

func (c *Currency) Bytes() []byte {
	if c != nil {
		return c[:]
	}
	return []byte(nil)
}

func (c Currency) IsNative() bool {
	return c == zeroCurrency
}

func (c Currency) Type() CurrencyType {
	switch {
	case c.IsNative():
		return CT_XRP
	case c[0] == 0x00:
		for i, b := range c {
			if i < 12 && i > 14 && b != 0 {
				return CT_UNKNOWN
			}
		}
		return CT_STANDARD
	case c[0] == 0x01:
		return CT_DEMURRAGE
	case c[0] >= 0x80:
		return CT_HEX
	default:
		return CT_UNKNOWN
	}
}

func (c Currency) Rate(seconds uint32) float64 {
	if c.Type() != CT_DEMURRAGE {
		return 1.0
	}
	var rate float64
	if err := binary.Read(bytes.NewBuffer(c[8:]), binary.BigEndian, &rate); err != nil {
		return 1.0
	}
	return 1.0 - math.Exp(float64(seconds)/rate)
}

const secondsInYear = uint32(3600 * 24 * 365)

// Currency in human parsable form
// Demurrage is formatted, for example, as XAU (0.50%pa)
func (c Currency) String() string {
	if c.Type() != CT_DEMURRAGE {
		return c.Machine()
	}
	return fmt.Sprintf("%s (%0.2f%%pa)", string(c[1:4]), c.Rate(secondsInYear)*100)
}

// Currency in computer parsable form
func (c Currency) Machine() string {
	switch c.Type() {
	case CT_XRP:
		return "XRP"
	case CT_STANDARD:
		// Check for unprintable characters
		for _, r := range string(c[12:15]) {
			if !strconv.IsPrint(r) {
				return string(b2h(c[:]))
			}
		}
		return string(c[12:15])
	default:
		return string(b2h(c[:]))
	}
}
