package data

import (
	. "gopkg.in/check.v1"
)

type CurrencySuite struct{}

var _ = Suite(&CurrencySuite{})

func (s *CurrencySuite) TestCurrencyTypes(c *C) {
	xrp, err := NewCurrency("XRP")
	c.Assert(err, IsNil)
	c.Assert(xrp.Machine(), Equals, "XRP")
	c.Assert(xrp.String(), Equals, "XRP")
	c.Assert(xrp.Type(), Equals, CT_XRP)
	c.Assert(xrp.IsNative(), Equals, true)

	usd, err := NewCurrency("USD")
	c.Assert(err, IsNil)
	c.Assert(usd.Machine(), Equals, "USD")
	c.Assert(usd.String(), Equals, "USD")
	c.Assert(usd.Type(), Equals, CT_STANDARD)

	demurrage, err := NewCurrency("015841551A748AD2C1F76FF6ECB0CCCD00000000")
	c.Assert(err, IsNil)
	c.Assert(demurrage.Machine(), Equals, "015841551A748AD2C1F76FF6ECB0CCCD00000000")
	c.Assert(demurrage.String(), Equals, "XAU (0.50%pa)")
	c.Assert(demurrage.Type(), Equals, CT_DEMURRAGE)

	demurrage2, err := NewCurrency("0158415500000000C1F76FF6ECB0BAC600000000")
	c.Assert(err, IsNil)
	c.Assert(demurrage2.Machine(), Equals, "0158415500000000C1F76FF6ECB0BAC600000000")
	c.Assert(demurrage2.String(), Equals, "XAU (0.50%pa)")
	c.Assert(demurrage2.Type(), Equals, CT_DEMURRAGE)

	hex, err := NewCurrency("815841551A748AD2C1F76FF6ECB0CCCD00000000")
	c.Assert(err, IsNil)
	c.Assert(hex.Machine(), Equals, "815841551A748AD2C1F76FF6ECB0CCCD00000000")
	c.Assert(hex.String(), Equals, "815841551A748AD2C1F76FF6ECB0CCCD00000000")
	c.Assert(hex.Type(), Equals, CT_HEX)

	// Unprintable
	wtf, err := NewCurrency("0000000000000000000000007F80010000000000")
	c.Assert(err, IsNil)
	c.Assert(wtf.Machine(), Equals, "0000000000000000000000007F80010000000000")
	c.Assert(wtf.String(), Equals, "0000000000000000000000007F80010000000000")
	c.Assert(wtf.Type(), Equals, CT_STANDARD)
}
