package data

import (
	"fmt"
	"testing"

	. "github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/testing"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type AmountSuite struct{}

var _ = Suite(&AmountSuite{})

var amountTests = TestSlice{
	// {amountCheck("0").Add(amountCheck("-1")).ToHuman(), Equals, "-1", "Negatives"},
	{amountCheck("1").IsPositive(), Equals, true, "Positives"},
	// {amountCheck(int64(1)).String(), Equals, "1/1/rrrrrrrrrrrrrrrrrrrrBZbvji", "FromNumber"}, //WHY?
	{amountCheck(int64(1)).String(), Equals, "0.000001/XRP", "int64(1) String"},
	{amountCheck("1/XRP").String(), Equals, "1/XRP", "Parse 1/XRP"},
	{amountCheck("1/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL").String(), Equals, "1/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL", "Parse 1/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL"},
	{amountCheck("10/015841551A748AD2C1F76FF6ECB0CCCD00000000/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Not(Equals), "10/XAU/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Demurrage"},
	{amountCheck("0").String(), Equals, "0/XRP", "Parse native 0"},
	{amountCheck("0.0").String(), Equals, "0/XRP", "Parse native 0.0"},
	{amountCheck("-0").String(), Equals, "0/XRP", "Parse native -0"},
	{amountCheck("-0.0").String(), Equals, "0/XRP", "Parse native -0.0"},
	{amountCheck("1000").String(), Equals, "0.001/XRP", "Parse native 1000"},
	{amountCheck("1234").String(), Equals, "0.001234/XRP", "Parse native 1234"},
	{amountCheck("12.3").String(), Equals, "12.3/XRP", "Parse native 12.3"},
	{amountCheck("-12.3").String(), Equals, "-12.3/XRP", "Parse native -12.3"},
	{amountCheck("123./USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "123/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Parse 123./USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh"},
	{amountCheck("12300/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "12300/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Parse 12300/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh"},
	{amountCheck("12.3/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "12.3/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Parse 12.3/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh"},
	{amountCheck("1.2300/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "1.23/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Parse 1.2300/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh"},
	{amountCheck("-0/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "0/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Parse -0/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh"},
	{amountCheck("-0.0/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "0/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Parse -0.0/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh"},
	{amountCheck("123").Negate().String(), Equals, "-0.000123/XRP", "Negate native 123"},
	{amountCheck("-123").Negate().String(), Equals, "0.000123/XRP", "Negate native -123"},
	{amountCheck("123/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").Negate().String(), Equals, "-123/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Negate 123/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh"},
	{amountCheck("-123/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").Negate().String(), Equals, "123/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Negate -123/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh"},
	{amountCheck("-123/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").Clone().String(), Equals, "-123/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Clone -123/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh"},
	{addCheck("150", "50").String(), Equals, "0.0002/XRP", "Add XRP to XRP"},
	{addCheck("150.02/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "50.5/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "200.52/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Add USD to USD"},
	{addCheck("0/USD", "1/USD").String(), Equals, "1/USD", "Add 0 USD to 1 USD"},
	{ErrorCheck(amountCheck("1/XRP").Add(amountCheck("1/USD"))), ErrorMatches, "Cannot add.*", "Add 1 XRP to 1 USD"},
	{subCheck("150.02/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "50.5/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "99.52/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Subtract USD from USD"},
	{mulCheck("0", "0").String(), Equals, "0/XRP", "Multiply 0 XRP with 0 XRP"},
	{mulCheck("0/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "0").String(), Equals, "0/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Multiply 0 USD with 0 XRP"},
	{mulCheck("0", "0/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "0/XRP", "Multiply 0 XRP with 0 USD"},
	{mulCheck("1", "0").String(), Equals, "0/XRP", "Multiply 1 XRP with 0 XRP"},
	{mulCheck("1/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "0").String(), Equals, "0/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Multiply 1 USD with 0 XRP"},
	{mulCheck("1", "0/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "0/XRP", "Multiply 1 XRP with 0 USD"},
	{mulCheck("0", "1").String(), Equals, "0/XRP", "Multiply 0 XRP with 1 XRP"},
	{mulCheck("0/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "1").String(), Equals, "0/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Multiply 0 USD with 1 XRP"},
	{mulCheck("0", "1/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "0/XRP", "Multiply 0 XRP with 1 USD"},
	{mulCheck("200", "10/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "0.002/XRP", "Multiply XRP with USD"},
	{mulCheck("20000", "10/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "0.2/XRP", "Multiply XRP with USD"},
	{mulCheck("2000000", "10/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "20/XRP", "Multiply XRP with USD"},
	{mulCheck("200", "-10/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "-0.002/XRP", "Multiply XRP with USD, neg"},
	{mulCheck("-6000", "37/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "-0.222/XRP", "Multiply XRP with USD, neg, frac"},
	{mulCheck("2000/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "10/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "20000/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Multiply USD with USD"},
	{mulCheck("2000000/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "100000/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "2e11/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Multiply USD with USD"},
	{mulCheck("100/EUR/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "1000/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "100000/EUR/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Multiply EUR with USD, result < 1"},
	{mulCheck("-24000/EUR/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "2000/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "-48000000/EUR/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Multiply EUR with USD, neg"},
	{mulCheck("0.1/EUR/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "-1000/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "-100/EUR/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Multiply EUR with USD, neg, <1"},
	{mulCheck("0.05/EUR/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "2000").String(), Equals, "100/EUR/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Multiply EUR with XRP, factor < 1"},
	{mulCheck("-100/EUR/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "5").String(), Equals, "-500/EUR/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Multiply EUR with XRP, neg"},
	{mulCheck("-0.05/EUR/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "2000").String(), Equals, "-100/EUR/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Multiply EUR with XRP, neg, <1"},
	{mulCheck("10", "10").String(), Equals, "0.0001/XRP", "Multiply XRP with XRP"},
	{mulCheck("2000/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "10/015841551A748AD2C1F76FF6ECB0CCCD00000000/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Not(Equals), "20000/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Multiply USD with XAU (demurred)"},
	{divCheck("200", "10/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "0.00002/XRP", "Divide XRP by USD"},
	{divCheck("20000", "10/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "0.002/XRP", "Divide XRP by USD"},
	{divCheck("2000000", "10/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "0.2/XRP", "Divide XRP by USD"},
	{divCheck("200", "-10/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "-0.00002/XRP", "Divide XRP by USD, neg"},
	{divCheck("-6000", "37/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "-0.000162/XRP", "Divide XRP by USD, neg, frac"},
	{divCheck("2000/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "10/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "200/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Divide USD by USD"},
	{divCheck("2000000/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "35/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "57142.85714285714/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Divide USD by USD, fractional"},
	{divCheck("2000000/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "100000/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "20/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Divide USD by USD"},
	{divCheck("100/EUR/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "1000/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "0.1/EUR/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Divide EUR by USD, factor < 1"},
	{divCheck("-24000/EUR/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "2000/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "-12/EUR/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Divide EUR by USD, neg"},
	{divCheck("100/EUR/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "-1000/USD/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh").String(), Equals, "-0.1/EUR/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Divide EUR by USD, neg, <1"},
	{divCheck("100/EUR/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "2000").String(), Equals, "0.05/EUR/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Divide EUR by XRP, result < 1"},
	{divCheck("-100/EUR/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "5").String(), Equals, "-20/EUR/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Divide EUR by XRP, neg"},
	{divCheck("-100/EUR/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "2000").String(), Equals, "-0.05/EUR/rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", "Divide EUR by XRP, neg, <1"},
	{equalCheck("0/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL", "0/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL"), Equals, true, "0 USD == 0 USD"},
	{equalCheck("0/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL", "0/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL"), Equals, true, "0 USD == 0 USD"},
	{equalCheck("0/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL", "-0/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL"), Equals, true, "0 USD == -0 USD"},
	{equalCheck("0", "0.0"), Equals, true, "0 XRP == 0 XRP"},
	{equalCheck("0", "-0"), Equals, true, "0 XRP == -0 XRP"},
	{equalCheck("10/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL", "10/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL"), Equals, true, "10 USD == 10 USD"},
	{equalCheck("123.4567/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL", "123.4567/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL"), Equals, true, "123.4567 USD == 123.4567 USD"},
	{equalCheck("10", "10"), Equals, true, "10 XRP == 10 XRP"},
	// {equalCheck("1.1", "11.0").ratio_human(10,false),Equals,true, "1.1 XRP == 1.1 XRP"},
	{amountCheck("0/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL").SameValue(amountCheck("0/USD/rH5aWQJ4R7v4Mpyf4kDBUvDFT5cbpFq3XP")), Equals, true, "0 USD == 0 USD (ignore issuer)"},
	{amountCheck("1.1/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL").SameValue(amountCheck("1.10/USD/rH5aWQJ4R7v4Mpyf4kDBUvDFT5cbpFq3XP")), Equals, true, "1.1 USD == 1.10 USD (ignore issuer)"},
	{equalCheck("10/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL", "100/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL"), Equals, false, "10 USD != 100 USD"},
	{equalCheck("10", "100"), Equals, false, "10 XRP != 100 XRP"},
	{equalCheck("1/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL", "2/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL"), Equals, false, "1 USD != 2 USD"},
	{equalCheck("1", "2"), Equals, false, "1 XRP != 2 XRP"},
	{equalCheck("0.1/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL", "0.2/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL"), Equals, false, "0.1 USD != 0.2 USD"},
	{equalCheck("1/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL", "-1/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL"), Equals, false, "1 USD != -1 USD"},
	{equalCheck("1", "-1"), Equals, false, "1 XRP != -1 XRP"},
	{equalCheck("1/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL", "1/USD/rH5aWQJ4R7v4Mpyf4kDBUvDFT5cbpFq3XP"), Equals, false, "1 USD != 1 USD (issuer mismatch)"},
	{equalCheck("1/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL", "1/EUR/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL"), Equals, false, "1 USD != 1 EUR"},
	{equalCheck("1/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL", "1"), Equals, false, "1 USD != 1 XRP"},
	{equalCheck("1", "1/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL"), Equals, false, "1 XRP != 1 USD"},
	{ErrorCheck(amountCheck("1").Divide(amountCheck("0"))), ErrorMatches, "Division by zero", "Divide one by zero"},
	{amountCheck("-1/XRP").Abs().String(), Equals, "1/XRP", "Abs -1"},
	// {ErrorCheck(NewAmount("xx")), ErrorMatches, "Bad amount:.*", "IsValid xx"},
	{ErrorCheck(NewAmount(nil)), ErrorMatches, "Bad type:.*", "IsValid nil"},
	{ErrorCheck(NewAmount(int(1))), ErrorMatches, "Bad type:.*", "IsValid int(0)"},

	{checkBinaryMarshal(amountCheck("0/XRP")).String(), Equals, "0/XRP", "Binary Marshal 0/XRP"},
	{checkBinaryMarshal(amountCheck("0.1/XRP")).String(), Equals, "0.1/XRP", "Binary Marshal 0.1/XRP"},
	{checkBinaryMarshal(amountCheck("-0.1/XRP")).String(), Equals, "-0.1/XRP", "Binary Marshal -0.1/XRP"},
	{checkBinaryMarshal(amountCheck("0/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL")).String(), Equals, "0/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL", "Binary Marshal 0/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL"},
	{checkBinaryMarshal(amountCheck("0.1/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL")).String(), Equals, "0.1/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL", "Binary Marshal 0.1/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL"},
	{checkBinaryMarshal(amountCheck("-0.1/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL")).String(), Equals, "-0.1/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL", "Binary Marshal -0.1/USD/rNDKeo9RrCiRdfsMG8AdoZvNZxHASGzbZL"},
}

func subCheck(a, b string) *Amount {
	if sum, err := amountCheck(a).Subtract(amountCheck(b)); err != nil {
		panic(err)
	} else {
		return sum
	}
}

func addCheck(a, b string) *Amount {
	if sum, err := amountCheck(a).Add(amountCheck(b)); err != nil {
		panic(err)
	} else {
		return sum
	}
}

func mulCheck(a, b string) *Amount {
	if product, err := amountCheck(a).Multiply(amountCheck(b)); err != nil {
		panic(err)
	} else {
		return product
	}
}

func divCheck(a, b string) *Amount {
	if quotient, err := amountCheck(a).Divide(amountCheck(b)); err != nil {
		panic(err)
	} else {
		return quotient
	}
}

func amountCheck(v interface{}) *Amount {
	if a, err := NewAmount(v); err != nil {
		panic(err)
	} else {
		return a
	}
}

func equalCheck(a, b string) bool {
	return amountCheck(a).Equals(*amountCheck(b))
}

func (s *AmountSuite) TestAmount(c *C) {
	amountTests.Test(c)
}

func ExampleValue_Add() {
	v1, _ := NewValue("100", false)
	v2, _ := NewValue("200.199", false)
	sum, _ := v1.Add(*v2)
	fmt.Println(v1.String())
	fmt.Println(v2.String())
	fmt.Println(sum.String())
	// Output:
	// 100
	// 200.199
	// 300.199
}

func checkBinaryMarshal(v1 *Amount) *Amount {
	var b []byte
	var err error

	if b, err = v1.MarshalBinary(); err != nil {
		panic(err)
	}

	v2 := &Amount{}
	if err = v2.UnmarshalBinary(b); err != nil {
		panic(err)
	}

	return v2
}
