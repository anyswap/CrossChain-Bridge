package data

import (
	. "github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/testing"
	. "gopkg.in/check.v1"
)

type ValueSuite struct{}

var _ = Suite(&ValueSuite{})

var valueTests = TestSlice{
	// Zero
	{valueCheckCanonical(false, false, 0, -15).String(), Equals, "0", "String 0, -15"},
	{valueCheckCanonical(false, false, 0, -25).String(), Equals, "0", "String 0, -25"},
	{valueCheckCanonical(false, false, 0, -26).String(), Equals, "0", "String 0, -26"},
	{valueCheckCanonical(false, false, 0, -5).String(), Equals, "0", "String 0, -5"},
	{valueCheckCanonical(false, false, 0, -4).String(), Equals, "0", "String 0, -4"},
	{valueCheckCanonical(false, true, 0, -15).String(), Equals, "0", "String -0, -15"},
	{valueCheckCanonical(false, true, 0, -25).String(), Equals, "0", "String -0, -25"},
	{valueCheckCanonical(false, true, 0, -26).String(), Equals, "0", "String -0, -26"},
	{valueCheckCanonical(false, true, 0, -5).String(), Equals, "0", "String -0, -5"},
	{valueCheckCanonical(false, true, 0, -4).String(), Equals, "0", "String -0, -4"},
	{valueCheckCanonical(true, false, 0, 0).String(), Equals, "0", "String n0, 0"},
	{valueCheckCanonical(true, false, 0, 6).String(), Equals, "0", "String n0, 0"},
	{valueCheckCanonical(true, false, 0, -6).String(), Equals, "0", "String n0, 0"},

	// Canonicalized values
	{valueCheckCanonical(false, false, 1230000000000000, -15).String(), Equals, "1.23", "String 1230000000000000, -15"},
	{valueCheckCanonical(false, false, 1230000000000000, -25).String(), Equals, "0.000000000123", "String 1230000000000000, -25"},
	{valueCheckCanonical(false, false, 1230000000000000, -26).String(), Equals, "123e-13", "String 1230000000000000, -26"},
	{valueCheckCanonical(false, false, 1230000000000000, -5).String(), Equals, "12300000000", "String 1230000000000000, -5"},
	{valueCheckCanonical(false, false, 1230000000000000, -4).String(), Equals, "123e9", "String 1230000000000000, -4"},
	{valueCheckCanonical(false, false, 9999999999999999, 80).String(), Equals, "9999999999999999e80", "String 9999999999999999, 80"},
	{valueCheckCanonical(false, false, 1000000000000000, -96).String(), Equals, "1e-81", "String 1000000000000000, -96"},
	{valueCheckCanonical(false, true, 1230000000000000, -15).String(), Equals, "-1.23", "String -1230000000000000, -15"},
	{valueCheckCanonical(false, true, 1230000000000000, -25).String(), Equals, "-0.000000000123", "String -1230000000000000, -25"},
	{valueCheckCanonical(false, true, 1230000000000000, -26).String(), Equals, "-123e-13", "String -1230000000000000, -26"},
	{valueCheckCanonical(false, true, 1230000000000000, -5).String(), Equals, "-12300000000", "String -1230000000000000, -5"},
	{valueCheckCanonical(false, true, 1230000000000000, -4).String(), Equals, "-123e9", "String -1230000000000000, -4"},
	{valueCheckCanonical(false, true, 9999999999999999, 80).String(), Equals, "-9999999999999999e80", "String -9999999999999999, 80"},
	{valueCheckCanonical(false, true, 1000000000000000, -96).String(), Equals, "-1e-81", "String -1000000000000000, -96"},

	// Native is stored as drips, but String()ed as XRP
	{valueCheckCanonical(true, false, 1, 0).String(), Equals, "0.000001", "String n1, 1"},
	{valueCheckCanonical(true, false, 1, 10).String(), Equals, "10000", "String n1, 10"},
	{valueCheckCanonical(true, false, 1, 5).String(), Equals, "0.1", "String n1, 5"},
	{valueCheckCanonical(true, false, 400, 5).String(), Equals, "40", "String n400, 5"},
	{valueCheckCanonical(true, true, 1, 0).String(), Equals, "-0.000001", "String n-1, 1"},
	{valueCheckCanonical(true, true, 1, 10).String(), Equals, "-10000", "String n-1, 10"},
	{valueCheckCanonical(true, true, 1, 5).String(), Equals, "-0.1", "String n-1, 5"},
	{valueCheckCanonical(true, true, 400, 5).String(), Equals, "-40", "String n-400, 5"},

	{valueCheckCanonical(false, false, 0, 0).Rat().FloatString(3), Equals, "0.000", "Rat String 0, 0"},
	{valueCheckCanonical(false, false, 1230000000000000, -15).Rat().FloatString(3), Equals, "1.230", "Rat String 1230000000000000, -15"},
	{valueCheckCanonical(true, false, 1, 0).Rat().FloatString(2), Equals, "1.00", "Rat String n1, 0"},
	{valueCheckCanonical(true, false, 4000000, 0).Rat().FloatString(2), Equals, "4000000.00", "Rat String n4000000, 0"},

	{valueCheck("0"), DeepEquals, valueCheckCanonical(false, false, 0, -100), "Parse 0"},
	{valueCheck("1"), DeepEquals, valueCheckCanonical(false, false, 1000000000000000, -15), "Parse 1"},
	{valueCheck("0.01"), DeepEquals, valueCheckCanonical(false, false, 1000000000000000, -17), "Parse 0.01"},
	{valueCheck("-0"), DeepEquals, valueCheckCanonical(false, false, 0, -100), "Parse -0"},
	{valueCheck("-1"), DeepEquals, valueCheckCanonical(false, true, 1000000000000000, -15), "Parse -1"},
	{valueCheck("-0.01"), DeepEquals, valueCheckCanonical(false, true, 1000000000000000, -17), "Parse -0.01"},
	{valueCheck("9999999999999999e80"), DeepEquals, valueCheckCanonical(false, false, 9999999999999999, 80), "Parse 9999999999999999e80"},
	{valueCheck("1e-81"), DeepEquals, valueCheckCanonical(false, false, 1000000000000000, -96), "Parse 1e-81"},

	{valueCheck("n0"), DeepEquals, valueCheckCanonical(true, false, 0, 0), "Parse n0"},
	{valueCheck("n0.0"), DeepEquals, valueCheckCanonical(true, false, 0, 0), "Parse n0.0"},
	{valueCheck("n9000000"), DeepEquals, valueCheckCanonical(true, false, 9000000, 0), "Parse n9000000"},
	{valueCheck("n-9000000"), DeepEquals, valueCheckCanonical(true, true, 9000000, 0), "Parse n-9000000"},
	{valueCheck("n9000000000000."), DeepEquals, valueCheckCanonical(true, false, 9000000000000000000, 0), "Parse n9000000000000"},
	{valueCheck("n-9000000000000."), DeepEquals, valueCheckCanonical(true, true, 9000000000000000000, 0), "Parse n-9000000000000"},

	{valueCheck("1e-82").IsZero(), Equals, true, "Parse 1e-82 (silent underflow)"},
	{ErrorCheck(NewValue("1e96", false)), ErrorMatches, "Value overflow: .*", "Parse 1e96 (overflow)"},
	{ErrorCheck(NewValue("foo", false)), ErrorMatches, "Invalid Number: .*", "Parse foo (invalid)"},
	{valueCheck("n0.0000001").IsZero(), Equals, true, "Parse n0.0000001 (silent underflow)"},
	{ErrorCheck(NewValue("9000000000000.000001", true)), ErrorMatches, "Native amount out of range: .*", "Parse n9000000000000.000001 (overflow)"},

	{valueCheck("123").ZeroClone().IsZero(), Equals, true, "ZeroClone is zero"},
	{valueCheck("123").ZeroClone().IsNative(), Equals, false, "ZeroClone is not native"},
	{valueCheck("0").IsZero(), Equals, true, "IsZero true"},
	{valueCheck("123").IsZero(), Equals, false, "IsZero false"},
	{valueCheck("n123").ZeroClone().IsZero(), Equals, true, "native ZeroClone is zero"},
	{valueCheck("n123").ZeroClone().IsNative(), Equals, true, "native ZeroClone is native"},
	{valueCheck("n0").IsZero(), Equals, true, "native IsZero true"},
	{valueCheck("n123").IsZero(), Equals, false, "native IsZero false"},

	{zeroNonNative.IsNative(), Equals, false, "zeroNonNative"},
	{zeroNative.IsNative(), Equals, true, "zeroNative"},

	{valueCheck("-0.01").Abs().String(), Equals, "0.01", "Abs -0.01"},
	{valueCheck("0.01").Abs().String(), Equals, "0.01", "Abs 0.01"},
	{valueCheck("n-0.01").Abs().String(), Equals, "0.01", "Abs n-0.01"},
	{valueCheck("n0.01").Abs().String(), Equals, "0.01", "Abs n0.01"},
	{valueCheck("n-20000").Abs().String(), Equals, "0.02", "Abs n-20000"},
	{valueCheck("n20000").Abs().String(), Equals, "0.02", "Abs n20000"},

	{valueCheck("123").Negate().String(), Equals, "-123", "Negate 123"},
	{valueCheck("-123").Negate().String(), Equals, "123", "Negate -123"},
	{valueCheck("0").Negate().String(), Equals, "0", "Negate 0"},
	{valueCheck("n123.").Negate().String(), Equals, "-123", "Negate n123"},
	{valueCheck("n-123.").Negate().String(), Equals, "123", "Negate n-123"},
	{valueCheck("n0").Negate().String(), Equals, "0", "Negate n0"},

	{equalValCheck("0", "0"), Equals, true, "0==0"},
	{equalValCheck("1", "1"), Equals, true, "1==1"},
	{equalValCheck("1", "0.1"), Equals, false, "1==0.1"},
	{equalValCheck("10", "0.1"), Equals, false, "10==0.1"},
	{equalValCheck("-1", "1"), Equals, false, "-1==1"},
	{equalValCheck("n0", "0"), Equals, true, "n0==0"},
	{equalValCheck("n1", "1"), Equals, true, "n1.==1"},
	{equalValCheck("n1", "0"), Equals, false, "n1==0"},
	{equalValCheck("n1", "n1"), Equals, true, "n1==n1"},

	{addValCheck("0", "0").String(), Equals, "0", "0+0"},
	{addValCheck("0", "1").String(), Equals, "1", "0+1"},
	{addValCheck("0", "0.0001").String(), Equals, "0.0001", "0+0.0001"},
	{addValCheck("1", "0").String(), Equals, "1", "1+0"},
	{addValCheck("1", "1").String(), Equals, "2", "1+1"},
	{addValCheck("-1", "1").String(), Equals, "0", "-1+1"},
	{addValCheck("-1", "-1").String(), Equals, "-2", "-1+-1"},
	{addValCheck("1", "-1").String(), Equals, "0", "1+-1"},
	{addValCheck("n0", "n0").String(), Equals, "0", "n0+n0"},
	{addValCheck("n0", "n1").String(), Equals, "0.000001", "n0+n1"},
	{addValCheck("n0", "n0.0001").String(), Equals, "0.0001", "n0+0.0001"},
	{addValCheck("n1", "n0").String(), Equals, "0.000001", "n1+n0"},
	{addValCheck("n1", "n1").String(), Equals, "0.000002", "n1+n1"},
	{addValCheck("n-1", "n1").String(), Equals, "0", "n-1+n1"},
	{addValCheck("n-1", "n-1").String(), Equals, "-0.000002", "n-1+n-1"},
	{addValCheck("n1", "n-1").String(), Equals, "0", "n1+n-1"},
	{ErrorCheck(valueCheck("n1").Add(*valueCheck("1"))), ErrorMatches, "Cannot add native and non-native values", "n1+1"},

	{subValCheck("0", "0").String(), Equals, "0", "0-0"},
	{subValCheck("1", "1").String(), Equals, "0", "1-1"},
	{subValCheck("-1", "0").String(), Equals, "-1", "-1-0"},
	{subValCheck("1", "-1").String(), Equals, "2", "1--1"},
	{subValCheck("0", "0.0001").String(), Equals, "-0.0001", "0-0.0001"},
	{subValCheck("n0", "n0").String(), Equals, "0", "n0-n0"},
	{subValCheck("n1", "n1").String(), Equals, "0", "n1-n1"},
	{subValCheck("n-1", "n0").String(), Equals, "-0.000001", "n-1n-0"},
	{subValCheck("n1", "n-1").String(), Equals, "0.000002", "n1-n-1"},
	{subValCheck("n0", "n0.0001").String(), Equals, "-0.0001", "n0-n0.0001"},
	{ErrorCheck(valueCheck("n1").Subtract(*valueCheck("1"))), ErrorMatches, "Cannot add native and non-native values", "n1+1"},

	{mulValCheck("0", "0").String(), Equals, "0", "0*0"},
	{mulValCheck("1", "0").String(), Equals, "0", "1*0"},
	{mulValCheck("0", "1").String(), Equals, "0", "0*1"},
	{mulValCheck("1", "1").String(), Equals, "1", "1*1"},
	{mulValCheck("1000", "0.001").String(), Equals, "1", "1000*0.001"},
	{mulValCheck("1000", "2").String(), Equals, "2000", "1000*2"},
	{mulValCheck("1000", "-2").String(), Equals, "-2000", "1000*-2"},
	{mulValCheck("-1000", "2").String(), Equals, "-2000", "1000*-2"},
	{mulValCheck("-1000", "-2").String(), Equals, "2000", "-1000*-2"},
	{mulValCheck("n0", "n0").String(), Equals, "0", "n0*n0"},
	{mulValCheck("n1", "n0").String(), Equals, "0", "n1*n0"},
	{mulValCheck("n0", "n1").String(), Equals, "0", "n0*n1"},
	{mulValCheck("n1", "n1").String(), Equals, "0.000001", "n1*n1"},
	{mulValCheck("n1.", "n1.").String(), Equals, "1000000", "n1.*n1."}, // Unintuitive case
	{mulValCheck("n1.", "2").String(), Equals, "2", "n1.*2"},
	{mulValCheck("n1.", "0.000001").String(), Equals, "0.000001", "n1.*0.000001"},
	{mulValCheck("n-1000.", "2").String(), Equals, "-2000", "n1000.*-2"},
	{mulValCheck("n-1000.", "-2").String(), Equals, "2000", "n-1000.*-2"},

	{ErrorCheck(valueCheck("0").Divide(*valueCheck("0"))), ErrorMatches, "Division by zero", "0/0"},
	{divValCheck("0", "1").String(), Equals, "0", "0/1"},
	{divValCheck("1", "2").String(), Equals, "0.5", "1/2"},
	{divValCheck("-1", "2").String(), Equals, "-0.5", "-1/2"},
	{divValCheck("1", "-200").String(), Equals, "-0.005", "1/-200"},
	{divValCheck("n0.", "n1.").String(), Equals, "0", "n0./n1."},
	{divValCheck("n1.", "n2.").String(), Equals, "0", "n1./n2. (underflow)"},
	{divValCheck("n-1.", "n2.").String(), Equals, "0", "n-1./n2. (underflow)"},
	{divValCheck("n1.", "n-200.").String(), Equals, "0", "n1./n-200. (underflow)"},

	{divValCheck("0", "n1").String(), Equals, "0", "0/n1"},
	{divValCheck("1", "n2000000").String(), Equals, "0.0000005", "1/n2000000"},
	{divValCheck("n-1000000", "2").String(), Equals, "-0.5", "n-1000000/2"},
	{divValCheck("1", "n-200000000").String(), Equals, "-0.000000005", "1/n-200000000"},

	{ratioValCheck("n1.", "n2.").String(), Equals, "0.5", "n1./n2. ratio"},
	{ratioValCheck("n-1.", "n2.").String(), Equals, "-0.5", "n-1./n2. ratio"},
	{ratioValCheck("n1.", "n-200.").String(), Equals, "-0.005", "n1./n-200. ratio"},
	{ratioValCheck("0", "n1").String(), Equals, "0", "0/n1 ratio"},
	{ratioValCheck("1", "n2000000").String(), Equals, "0.5", "1/n2000000 ratio"},
	{ratioValCheck("n-1000000", "2").String(), Equals, "-0.5", "n-1000000/2 ratio"},
	{ratioValCheck("1", "n-200000000").String(), Equals, "-0.005", "1/n-200000000 ratio"},

	{valueCheck("1").Compare(*valueCheck("1")), Equals, 0, "1 Compare 1"},
	{valueCheck("0").Compare(*valueCheck("1")), Equals, -1, "0 Compare 1"},
	{valueCheck("1").Compare(*valueCheck("0")), Equals, 1, "1 Compare 0"},
	{valueCheck("0").Compare(*valueCheck("0")), Equals, 0, "0 Compare 0"},
	{valueCheck("0").Compare(*valueCheck("-1")), Equals, 1, "0 Compare -1"},
	{valueCheck("-1").Compare(*valueCheck("0")), Equals, -1, "-1 Compare 0"},
	{valueCheck("-1").Compare(*valueCheck("1")), Equals, -1, "-1 Compare 1"},
	{valueCheck("1").Compare(*valueCheck("-1")), Equals, 1, "1 Compare -1"},
	{valueCheck("-1").Compare(*valueCheck("2")), Equals, -1, "-1 Compare 2"},
	{valueCheck("-2").Compare(*valueCheck("1")), Equals, -1, "-2 Compare 1"},
	{valueCheck("1").Compare(*valueCheck("0.002")), Equals, 1, "1 Compare 0.002"},
	{valueCheck("-1").Compare(*valueCheck("0.002")), Equals, -1, "-1 Compare 0.002"},
	{valueCheck("1").Compare(*valueCheck("-0.002")), Equals, 1, "1 Compare -0.002"},
	{valueCheck("-1").Compare(*valueCheck("-0.002")), Equals, -1, "-1 Compare -0.002"},
	{valueCheck("0.002").Compare(*valueCheck("1")), Equals, -1, "0.002 Compare 1"},
	{valueCheck("-0.002").Compare(*valueCheck("1")), Equals, -1, "-0.002 Compare 1"},
	{valueCheck("0.002").Compare(*valueCheck("-1")), Equals, 1, "0.002 Compare -1"},
	{valueCheck("-0.002").Compare(*valueCheck("-1")), Equals, 1, "-0.002 Compare -1"},

	{valueCheck("n1").Compare(*valueCheck("n1")), Equals, 0, "n1 Compare n1"},
	{valueCheck("n0").Compare(*valueCheck("n1")), Equals, -1, "n0 Compare n1"},
	{valueCheck("n1").Compare(*valueCheck("n0")), Equals, 1, "n1 Compare n0"},
	{valueCheck("n0").Compare(*valueCheck("n0")), Equals, 0, "n0 Compare n0"},
	{valueCheck("n0").Compare(*valueCheck("n-1")), Equals, 1, "n0 Compare n-1"},
	{valueCheck("n-1").Compare(*valueCheck("n0")), Equals, -1, "n-1 Compare n0"},
	{valueCheck("n-1").Compare(*valueCheck("n1")), Equals, -1, "n-1 Compare n1"},
	{valueCheck("n1").Compare(*valueCheck("n-1")), Equals, 1, "n1 Compare n-1"},
	{valueCheck("n-1").Compare(*valueCheck("n2")), Equals, -1, "n-1 Compare n2"},
	{valueCheck("n-2").Compare(*valueCheck("n1")), Equals, -1, "n-2 Compare n1"},

	{valueCheck("n2000").Compare(*valueCheck("2000")), Equals, 0, "n2000 Compare 2000"},
	{valueCheck("n1").Compare(*valueCheck("0.002")), Equals, 1, "n1 Compare 0.002"},
	{valueCheck("n0").Compare(*valueCheck("0.002")), Equals, -1, "n0 Compare 0.002"},
	{valueCheck("n1000000").Compare(*valueCheck("-0.002")), Equals, 1, "n1000000 Compare -0.002"},

	{valueCheck("1").Less(*valueCheck("1")), Equals, false, "1<1"},
	{valueCheck("0").Less(*valueCheck("1")), Equals, true, "0<1"},
	{valueCheck("n1").Less(*valueCheck("1")), Equals, false, "n1<1"},
	{valueCheck("n1.").Less(*valueCheck("1")), Equals, false, "n1.<1"},

	{checkValBinaryMarshal(valueCheck("0")).String(), Equals, "0", "Binary marshal 0"},
	{checkValBinaryMarshal(valueCheck("n0.1")).String(), Equals, "0.1", "Binary marshal n0.1"},
	{checkValBinaryMarshal(valueCheck("n-0.1")).String(), Equals, "-0.1", "Binary marshal n-0.1"},
	{checkValBinaryMarshal(valueCheck("0.1")).String(), Equals, "0.1", "Binary marshal 0.1"},
	{checkValBinaryMarshal(valueCheck("-0.1")).String(), Equals, "-0.1", "Binary marshal -0.1"},

	{checkValHex(valueCheckCanonical(false, false, 0, -15)), Equals, "8000000000000000", "Zero hex"},
}

func subValCheck(a, b string) *Value {
	if sum, err := valueCheck(a).Subtract(*valueCheck(b)); err != nil {
		panic(err)
	} else {
		return sum
	}
}

func addValCheck(a, b string) *Value {
	if sum, err := valueCheck(a).Add(*valueCheck(b)); err != nil {
		panic(err)
	} else {
		return sum
	}
}

func mulValCheck(a, b string) *Value {
	if product, err := valueCheck(a).Multiply(*valueCheck(b)); err != nil {
		panic(err)
	} else {
		return product
	}
}

func divValCheck(a, b string) *Value {
	if quotient, err := valueCheck(a).Divide(*valueCheck(b)); err != nil {
		panic(err)
	} else {
		return quotient
	}
}

func ratioValCheck(a, b string) *Value {
	if ratio, err := valueCheck(a).Ratio(*valueCheck(b)); err != nil {
		panic(err)
	} else {
		return ratio
	}
}

func valueCheck(v string) *Value {
	native := false
	if v[0] == 'n' {
		v = v[1:]
		native = true
	}

	if a, err := NewValue(v, native); err != nil {
		panic(err)
	} else {
		return a
	}
}

func valueCheckCanonical(native, negative bool, num uint64, offset int64) *Value {
	v := newValue(native, negative, num, offset)
	if err := v.canonicalise(); err != nil {
		panic(err)
	}
	return v
}

func equalValCheck(a, b string) bool {
	return valueCheck(a).Equals(*valueCheck(b))
}

func (s *ValueSuite) TestValue(c *C) {
	valueTests.Test(c)
}

func checkValBinaryMarshal(v1 *Value) *Value {
	var b []byte
	var err error

	if b, err = v1.MarshalBinary(); err != nil {
		panic(err)
	}

	v2 := &Value{}
	if err = v2.UnmarshalBinary(b); err != nil {
		panic(err)
	}

	return v2
}

func checkValHex(v1 *Value) string {
	var b []byte
	var err error

	if b, err = v1.MarshalBinary(); err != nil {
		panic(err)
	}

	return string(b2h(b))
}
