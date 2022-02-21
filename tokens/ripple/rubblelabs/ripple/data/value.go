package data

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

const (
	minOffset        int64  = -96
	maxOffset        int64  = 80
	minValue         uint64 = 1000000000000000
	maxValue         uint64 = 9999999999999999
	maxNative        uint64 = 9000000000000000000
	maxNativeNetwork uint64 = 100000000000000000
	notNative        uint64 = 0x8000000000000000
	positive         uint64 = 0x4000000000000000
	maxNativeSqrt    uint64 = 3000000000
	maxNativeDiv     uint64 = 2095475792 // MaxNative / 2^32
	tenTo14          uint64 = 100000000000000
	tenTo14m1        uint64 = tenTo14 - 1
	tenTo17          uint64 = tenTo14 * 1000
	tenTo17m1        uint64 = tenTo17 - 1
	xrpPrecision     uint64 = 1000000
)

var (
	bigTen        = big.NewInt(10)
	bigTenTo14    = big.NewInt(0).SetUint64(tenTo14)
	bigTenTo17    = big.NewInt(0).SetUint64(tenTo17)
	zeroNative    = *newValue(true, false, 0, 0)
	zeroNonNative = *newValue(false, false, 0, 0)
	xrpMultipler  = newValue(true, false, xrpPrecision, 0)
)

// Value is the numeric type in Ripple. It can store numbers in two different
// representations: native and non-native.
// Non-native numbers are stored as a mantissa (Num) in the range [1e15,1e16)
// plus an exponent (Offset) in the range [-96,80].
// Native values are stored as an integer number of "drips" each representing
// 1/1000000.
// Throughout the library, native values are interpreted as drips unless
// otherwise specified.
type Value struct {
	native   bool
	negative bool
	num      uint64
	offset   int64
}

func init() {
	if err := zeroNative.canonicalise(); err != nil {
		panic(err)
	}
	if err := zeroNonNative.canonicalise(); err != nil {
		panic(err)
	}
	if err := xrpMultipler.canonicalise(); err != nil {
		panic(err)
	}
}

func newValue(native, negative bool, num uint64, offset int64) *Value {
	return &Value{
		native:   native,
		negative: negative,
		num:      num,
		offset:   offset,
	}
}

// NewNativeValue returns a Value with n drops.
// If the value is impossible an error is returned.
func NewNativeValue(n int64) (*Value, error) {
	v := newValue(true, n < 0, uint64(n), 0)
	return v, v.canonicalise()
}

// NewNonNativeValue returns a Value of n*10^offset.
func NewNonNativeValue(n int64, offset int64) (*Value, error) {
	v := newValue(false, n < 0, uint64(n), offset)
	return v, v.canonicalise()
}

// Match fields:
// 0 = whole input
// 1 = sign
// 2 = integer portion
// 3 = whole fraction (with '.')
// 4 = fraction (without '.')
// 5 = whole exponent (with 'e')
// 6 = exponent sign
// 7 = exponent number
var valueRegex = regexp.MustCompile("([+-]?)(\\d*)(\\.(\\d*))?([eE]([+-]?)(\\d+))?")

// NewValue accepts a string representation of a value and a flag to indicate if it
// should be stored as native. If the native flag is set AND a decimal is used, the
// number is interpreted as XRP. If no decimal is used, it is interpreted as drips.
func NewValue(s string, native bool) (*Value, error) {
	var err error
	v := Value{
		native: native,
	}
	matches := valueRegex.FindStringSubmatch(s)
	if matches == nil {
		return nil, fmt.Errorf("Invalid Number: %s", s)
	}
	if len(matches[2])+len(matches[4]) > 32 {
		return nil, fmt.Errorf("Overlong Number: %s", s)
	}
	if matches[1] == "-" {
		v.negative = true
	}
	if len(matches[4]) == 0 {
		if v.num, err = strconv.ParseUint(matches[2], 10, 64); err != nil {
			return nil, fmt.Errorf("Invalid Number: %s Reason: %s", s, err.Error())
		}
		v.offset = 0
	} else {
		if v.num, err = strconv.ParseUint(matches[2]+matches[4], 10, 64); err != nil {
			return nil, fmt.Errorf("Invalid Number: %s Reason: %s", s, err.Error())
		}
		v.offset = -int64(len(matches[4]))
	}
	if len(matches[5]) > 0 {
		exp, err := strconv.ParseInt(matches[7], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("Invalid Number: %s %s", s, err.Error())
		}
		if matches[6] == "-" {
			v.offset -= exp
		} else {
			v.offset += exp
		}
	}
	if v.IsNative() && len(matches[3]) > 0 {
		v.offset += 6
	}
	return &v, v.canonicalise()
}

func (v *Value) canonicalise() error {
	if v.IsNative() {
		if v.num == 0 {
			v.offset = 0
			v.negative = false
		} else {
			for v.offset < 0 {
				v.num /= 10
				v.offset++
			}
			for v.offset > 0 {
				v.num *= 10
				v.offset--
			}
			if v.num > maxNative {
				return fmt.Errorf("Native amount out of range: %s", v.debug())
			}
		}
	} else {
		if v.num == 0 {
			v.offset = -100
			v.negative = false
		} else {
			for v.num < minValue && v.offset > minOffset {
				v.num *= 10
				v.offset--
			}
			for v.num > maxValue {
				if v.offset >= maxOffset {
					return fmt.Errorf("Value overflow: %s", v.debug())
				}
				v.num /= 10
				v.offset++
			}
			if v.offset < minOffset || v.num < minValue {
				v.num = 0
				v.offset = 0
				v.negative = false
			}
			if v.offset > maxOffset {
				return fmt.Errorf("Value overflow: %s", v.debug())
			}
		}
	}
	return nil
}

// Native returns a clone of the value in native format.
func (v Value) Native() (*Value, error) {
	v.native = true
	return &v, v.canonicalise()
}

// NonNative returns a clone of the value in non-native format.
func (v Value) NonNative() (*Value, error) {
	v.native = false
	return &v, v.canonicalise()
}

// Clone returns a Value which is a copy of v.
func (v Value) Clone() *Value {
	return newValue(v.native, v.negative, v.num, v.offset)
}

// ZeroClone returns a zero Value, native or non-native depending on v's setting.
func (v Value) ZeroClone() *Value {
	if v.IsNative() {
		return zeroNative.Clone()
	}
	return zeroNonNative.Clone()
}

// Abs returns a copy of v with a positive sign.
func (v Value) Abs() *Value {
	return newValue(v.native, false, v.num, v.offset)
}

// Negate returns a new Value with the opposite sign of v.
func (v Value) Negate() *Value {
	return newValue(v.native, !v.negative, v.num, v.offset)
}

func (a Value) factor(b Value) (int64, int64, int64) {
	ao, bo := a.offset, b.offset
	av, bv := int64(a.num), int64(b.num)
	if a.negative {
		av = -av
	}
	if b.negative {
		bv = -bv
	}

	if a.IsZero() {
		return av, bv, bo
	}
	if b.IsZero() {
		return av, bv, ao
	}

	// FIXME: This can silently underflow
	for ; ao < bo; ao++ {
		av /= 10
	}
	for ; bo < ao; bo++ {
		bv /= 10
	}
	return av, bv, ao
}

// Add adds a to b and returns the sum as a new Value.
func (a Value) Add(b Value) (*Value, error) {
	switch {
	case a.IsNative() != b.IsNative():
		return nil, fmt.Errorf("Cannot add native and non-native values")
	case a.IsZero():
		return b.Clone(), nil
	case b.IsZero():
		return a.Clone(), nil
	default:
		av, bv, ao := a.factor(b)
		v := newValue(a.native, (av+bv) < 0, abs(av+bv), ao)
		return v, v.canonicalise()
	}
}

func (a Value) Subtract(b Value) (*Value, error) {
	return a.Add(*b.Negate())
}

func normalise(a, b Value) (uint64, uint64, int64, int64) {
	av, bv := a.num, b.num
	ao, bo := a.offset, b.offset
	if a.IsNative() {
		for ; av < minValue; av *= 10 {
			ao--
		}
	}
	if b.IsNative() {
		for ; bv < minValue; bv *= 10 {
			bo--
		}
	}
	return av, bv, ao, bo
}

func (a Value) Multiply(b Value) (*Value, error) {
	if a.IsZero() || b.IsZero() {
		return a.ZeroClone(), nil
	}
	if a.IsNative() && b.IsNative() {
		min := min64(a.num, b.num)
		max := max64(a.num, b.num)
		if min > maxNativeSqrt || (((max >> 32) * min) > maxNativeDiv) {
			return nil, fmt.Errorf("Native value overflow: %s*%s", a.debug(), b.debug())
		}
		return NewNativeValue(int64(min * max))
	}
	av, bv, ao, bo := normalise(a, b)
	// Compute (numerator * denominator) / 10^14 with rounding
	// 10^16 <= result <= 10^18
	m := big.NewInt(0).SetUint64(av)
	m.Mul(m, big.NewInt(0).SetUint64(bv))
	m.Div(m, bigTenTo14)
	// 10^16 <= product <= 10^18
	if len(m.Bytes()) > 64 {
		return nil, fmt.Errorf("Multiply: %s*%s", a.debug(), b.debug())
	}
	v := newValue(a.native, a.negative != b.negative, m.Uint64()+7, ao+bo+14)
	return v, v.canonicalise()
}

func (num Value) Divide(den Value) (*Value, error) {
	if den.IsZero() {
		return nil, fmt.Errorf("Division by zero")
	}
	if num.IsZero() {
		return num.ZeroClone(), nil
	}
	av, bv, ao, bo := normalise(num, den)
	// Compute (numerator * 10^17) / denominator
	d := big.NewInt(0).SetUint64(av)
	d.Mul(d, bigTenTo17)
	d.Div(d, big.NewInt(0).SetUint64(bv))
	// 10^16 <= quotient <= 10^18
	if len(d.Bytes()) > 64 {
		return nil, fmt.Errorf("Divide: %s/%s", num.debug(), den.debug())
	}
	v := newValue(num.native, num.negative != den.negative, d.Uint64()+5, ao-bo-17)
	return v, v.canonicalise()
}

// Ratio returns the ratio a/b. XRP are interpreted at face value rather than drips.
// The result of Ratio is always a non-native Value for additional precision.
func (a Value) Ratio(b Value) (*Value, error) {
	var err error
	num := &a
	den := &b

	if num.IsNative() {
		num, err = num.NonNative()
		if err != nil {
			return nil, err
		}
		num, err = num.Divide(*xrpMultipler)
		if err != nil {
			return nil, err
		}
	}
	if den.IsNative() {
		den, err = den.NonNative()
		if err != nil {
			return nil, err
		}
		den, err = den.Divide(*xrpMultipler)
		if err != nil {
			return nil, err
		}
	}
	quotient, err := num.Divide(*den)
	if err != nil {
		return nil, err
	}
	return quotient, nil
}

// Less compares values and returns true if a is less than b.
func (a Value) Less(b Value) bool {
	return a.Compare(b) < 0
}

func (a Value) Equals(b Value) bool {
	return a.Compare(b) == 0
}

// Compare returns an integer comparing two Values.
// The result will be 0 if a==b, -1 if a < b, and +1 if a > b.
func (a Value) Compare(b Value) int {
	return a.Rat().Cmp(b.Rat())
}

// isScientific indicates when the value should be String()ed in scientific notation.
func (v Value) isScientific() bool {
	return v.offset != 0 && (v.offset < -25 || v.offset > -5)
}

func (v Value) IsNative() bool {
	return v.native
}

func (v Value) IsNegative() bool {
	return v.negative
}

func (v Value) IsZero() bool {
	return v.num == 0
}

func (v *Value) Bytes() []byte {
	if v == nil {
		return nil
	}
	var u uint64
	if !v.negative && (v.num > 0 || v.IsNative()) {
		u |= 1 << 62
	}
	if v.IsNative() {
		u |= v.num & ((1 << 62) - 1)
	} else {
		u |= 1 << 63
		u |= v.num & ((1 << 54) - 1)
		if v.num > 0 {
			u |= uint64(v.offset+97) << 54
		}
	}
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], u)
	return b[:]
}

func (v Value) MarshalBinary() ([]byte, error) {
	return v.Bytes(), nil
}

func (v *Value) UnmarshalBinary(b []byte) error {
	buf := bytes.NewBuffer(b)
	return v.Unmarshal(buf)
}

// Rat returns the value as a big.Rat.
func (v Value) Rat() *big.Rat {
	n := big.NewInt(int64(v.num))
	if v.negative {
		n.Neg(n)
	}

	d := big.NewInt(1)
	if v.offset < 0 {
		d.Exp(big.NewInt(10), big.NewInt(-v.offset), nil)
	} else if v.offset > 0 {
		mult := big.NewInt(1)
		mult.Exp(big.NewInt(10), big.NewInt(v.offset), nil)
		n.Mul(n, mult)
	}

	res := big.NewRat(0, 1)
	res.SetFrac(n, d)
	return res
}

func (v Value) Float() float64 {
	switch {
	case v.negative && v.native:
		return -float64(v.num) / 1000000
	case v.native:
		return float64(v.num) / 1000000
	case v.negative:
		return -float64(v.num) * math.Pow10(int(v.offset))
	default:
		return float64(v.num) * math.Pow10(int(v.offset))
	}
}

// String returns the Value as a string for human consumption. Native values are
// represented as decimal XRP rather than drips.
func (v Value) String() string {
	if v.IsZero() {
		return "0"
	}
	if !v.IsNative() && v.isScientific() {
		value := strconv.FormatUint(v.num, 10)
		origLen := len(value)
		value = strings.TrimRight(value, "0")
		offset := strconv.FormatInt(v.offset+int64(origLen-len(value)), 10)
		if v.negative {
			return "-" + value + "e" + offset
		}
		return value + "e" + offset
	}
	rat := v.Rat()
	if v.IsNative() {
		rat.Quo(rat, big.NewRat(int64(xrpPrecision), 1))
	}
	left := rat.FloatString(0)
	if rat.IsInt() {
		return left
	}
	length := len(left)
	if v.negative {
		length -= 1
	}
	return strings.TrimRight(rat.FloatString(32-length), "0")
}

func (v Value) debug() string {
	return fmt.Sprintf("Native: %t Negative: %t Value: %d Offset: %d", v.native, v.negative, v.num, v.offset)
}
