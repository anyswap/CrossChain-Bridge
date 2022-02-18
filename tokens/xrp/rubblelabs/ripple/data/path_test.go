package data

import (
	. "gopkg.in/check.v1"
)

type PathSuite struct{}

var _ = Suite(&PathSuite{})

func (s *PathSuite) TestPathElemOffer(c *C) {
	pe, err := newPathElem("BTC/rNPRNzBB92BVpAhhZr4iXDTveCgV5Pofm9")
	c.Assert(err, IsNil)

	c.Assert(pe.Account, IsNil)
	c.Assert(pe.Currency.String(), Equals, "BTC")
	c.Assert(pe.Issuer.String(), Equals, "rNPRNzBB92BVpAhhZr4iXDTveCgV5Pofm9")
}

func (s *PathSuite) TestPathElemAccount(c *C) {
	pe, err := newPathElem("rNPRNzBB92BVpAhhZr4iXDTveCgV5Pofm9")
	c.Assert(err, IsNil)

	c.Assert(pe.Account.String(), Equals, "rNPRNzBB92BVpAhhZr4iXDTveCgV5Pofm9")
	c.Assert(pe.Currency, IsNil)
	c.Assert(pe.Issuer, IsNil)
}

func (s *PathSuite) TestPathElemError(c *C) {
	_, err := newPathElem("Foo")
	c.Assert(err.Error(), Equals, "Base58 string too short: Foo")
}

func (s *PathSuite) TestPath(c *C) {
	p, err := NewPath("BTC/rNPRNzBB92BVpAhhZr4iXDTveCgV5Pofm9 => r3ADD8kXSUKHd6zTCKfnKT3zV9EZHjzp1S")
	c.Assert(err, IsNil)

	c.Assert(p, HasLen, 2)
	c.Assert(p[0].Currency.String(), Equals, "BTC")
	c.Assert(p[0].Issuer.String(), Equals, "rNPRNzBB92BVpAhhZr4iXDTveCgV5Pofm9")
	c.Assert(p[0].Account, IsNil)
	c.Assert(p[1].Currency, IsNil)
	c.Assert(p[1].Issuer, IsNil)
	c.Assert(p[1].Account.String(), Equals, "r3ADD8kXSUKHd6zTCKfnKT3zV9EZHjzp1S")
}

func (s *PathSuite) TestPathError(c *C) {
	_, err := NewPath("Foo")
	c.Assert(err.Error(), Equals, "Base58 string too short: Foo")
}
