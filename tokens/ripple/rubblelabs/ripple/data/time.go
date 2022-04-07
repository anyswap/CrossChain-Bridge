package data

import (
	"time"
)

const (
	rippleTimeEpoch  int64  = 946684800
	rippleTimeFormat string = "2006-Jan-02 15:04:05 UTC"
)

// Represents a time as the number of seconds since the Ripple epoch: January 1st, 2000 (00:00 UTC)
type RippleTime struct {
	T uint32
}

type rippleHumanTime struct {
	RippleTime
}

func NewRippleTime(t uint32) *RippleTime {
	return &RippleTime{t}
}

func convertToRippleTime(t time.Time) uint32 {
	return uint32(t.Sub(time.Unix(rippleTimeEpoch, 0)).Nanoseconds() / 1000000000)
}

func (t RippleTime) Time() time.Time {
	return time.Unix(int64(t.T)+rippleTimeEpoch, 0)
}

func Now() *RippleTime {
	return &RippleTime{convertToRippleTime(time.Now())}
}

// Accepts time formatted as 2006-Jan-02 15:04:05
func (t *RippleTime) SetString(s string) error {
	v, err := time.Parse(rippleTimeFormat, s)
	if err != nil {
		return err
	}
	t.SetUint32(convertToRippleTime(v))
	return nil
}

func (t *RippleTime) SetUint32(n uint32) {
	t.T = n
}

func (t RippleTime) Uint32() uint32 {
	return t.T
}

func (t RippleTime) human() *rippleHumanTime {
	return &rippleHumanTime{t}
}

// Returns time formatted as 2006-Jan-02 15:04:05
func (t RippleTime) String() string {
	return t.Time().UTC().Format(rippleTimeFormat)
}

// Returns time formatted as 15:04:05
func (t RippleTime) Short() string {
	return t.Time().UTC().Format("15:04:05")
}
