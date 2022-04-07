package crypto

import (
	"testing"

	. "github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/testing"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type HashSuite struct{}

var _ = Suite(&HashSuite{})

var testAccounts = map[string]struct {
	Account, Secret string
}{
	"alice":    {"rG1QQv2nh2gr7RCZ1P8YYcBUKCCN633jCn", "alice"},
	"bob":      {"rPMh7Pi9ct699iZUTWaytJUoHcJ7cgyziK", "bob"},
	"carol":    {"rH4KEcG9dEwGwpn6AyoWK9cZPLL4RLSmWW", "carol"},
	"dan":      {"rJ85Mok8YRNxSo7NnxKGrPuk29uAeZQqwZ", "dan"},
	"bitstamp": {"r4jKmc2nQb5yEU6eycefiNKGHTU5NQJASx", "bitstamp"},
	"mtgox":    {"rGihwhaqU8g7ahwAvTq6iX5rvsfcbgZw6v", "mtgox"},
	"amazon":   {"rhheXqX7bDnXePJeMHhubDDvw2uUTtenPd", "amazon"},
	"root":     {"rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "masterpassphrase"},
}

var accountTests = TestSlice{
	{accountCheck("0").Value().String(), Equals, "0", "Parse 0"},
	{accountCheck("0").String(), Equals, ACCOUNT_ZERO, "Parse 0 export"},
	{accountCheck("1").Value().String(), Equals, "1", "Parse 1"},
	{accountCheck("1").String(), Equals, ACCOUNT_ONE, "Parse 1 export"},
	{accountCheck(ACCOUNT_ZERO).String(), Equals, ACCOUNT_ZERO, "Parse rrrrrrrrrrrrrrrrrrrrrhoLvTp export"},
	{accountCheck(ACCOUNT_ONE).String(), Equals, ACCOUNT_ONE, "Parse rrrrrrrrrrrrrrrrrrrrBZbvji export"},
	{accountCheck(testAccounts["mtgox"].Account).String(), Equals, testAccounts["mtgox"].Account, "Parse mtgox export"},
	{accountCheck(ACCOUNT_ZERO), Not(Equals), nil, "IsValid rrrrrrrrrrrrrrrrrrrrrhoLvTp"},
	{ErrorCheck(NewRippleHash("rrrrrrrrrrrrrrrrrrrrrhoLvT")), ErrorMatches, "Bad Base58 checksum:.*", "IsValid rrrrrrrrrrrrrrrrrrrrrhoLvT"},
}

func accountCheck(v interface{}) Hash {
	if a, err := NewRippleHash(v.(string)); err != nil {
		panic(err)
	} else {
		return a
	}
}

func (s *HashSuite) TestHashes(c *C) {
	accountTests.Test(c)
}
