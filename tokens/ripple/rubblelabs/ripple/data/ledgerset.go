package data

import (
	"fmt"
	"github.com/willf/bitset"
	"sort"
	"time"
)

type LedgerSlice []uint32

func (s LedgerSlice) Len() int            { return len(s) }
func (s LedgerSlice) Swap(i, j int)       { s[i], s[j] = s[j], s[i] }
func (s LedgerSlice) Less(i, j int) bool  { return s[i] < s[j] }
func (s LedgerSlice) Sorted() LedgerSlice { sort.Sort(s); return s }

type LedgerRange struct {
	Start uint32
	End   uint32
	Max   uint32
}

type Work struct {
	*LedgerRange
	MissingLedgers LedgerSlice
	MissingNodes   []Hash256
}

type LedgerSet struct {
	ledgers  *bitset.BitSet
	start    uint32
	taken    map[uint32]time.Time
	returned uint64
	duration time.Duration
}

func NewLedgerSet(start, capacity uint32) *LedgerSet {
	return &LedgerSet{
		ledgers: bitset.New(uint(capacity)).Complement(),
		start:   start,
		taken:   make(map[uint32]time.Time),
	}
}

func (l *LedgerSet) String() string {
	var rate float64
	if l.returned > 0 {
		rate = l.duration.Seconds() / float64(l.returned)
	}
	return fmt.Sprintf("Count: %d Taken: %d Avg: %0.04f secs", l.Count(), l.Taken(), rate)
}

func (l *LedgerSet) Taken() uint32 {
	return uint32(len(l.taken))
}

func (l *LedgerSet) Max() uint32 {
	return uint32(l.ledgers.Len())
}

func (l *LedgerSet) Count() uint32 {
	return uint32(l.ledgers.Len() - l.ledgers.Count())
}

func (l *LedgerSet) Extend(i uint32) {
	for j, length := uint(i-1), l.ledgers.Len(); j > length; j-- {
		l.ledgers.Set(j)
	}
}

func (l *LedgerSet) Set(i uint32) time.Duration {
	l.Extend(i)
	l.ledgers.Clear(uint(i))
	if when, ok := l.taken[i]; ok {
		delete(l.taken, i)
		l.returned++
		duration := time.Now().Sub(when)
		l.duration += duration
		return duration
	}
	return time.Duration(0)
}

func (l *LedgerSet) take(i uint32) bool {
	if !l.ledgers.Test(uint(i)) {
		return false
	}
	when, ok := l.taken[i]
	if !ok || (ok && time.Now().Sub(when).Seconds() > 90) {
		l.taken[i] = time.Now()
		return true
	}
	return false
}

func (l *LedgerSet) TakeMiddle(r *LedgerRange) LedgerSlice {
	ledgers := make(LedgerSlice, 0, r.Max)
	for start, end := max(r.Start, l.start), min(r.End, uint32(l.ledgers.Len())); start <= end && uint32(len(ledgers)) < r.Max; start++ {
		if l.take(start) {
			ledgers = append(ledgers, uint32(start))
		}
	}
	return ledgers
}

func (l *LedgerSet) TakeBottom(n uint32) LedgerSlice {
	r := &LedgerRange{l.start, uint32(l.ledgers.Len()), n}
	return l.TakeMiddle(r)
}

func (l *LedgerSet) TakeTop(n uint32) LedgerSlice {
	ledgers := make(LedgerSlice, 0, n)
	for i := uint32(l.ledgers.Len()) - 1; i >= l.start && len(ledgers) < int(n); i-- {
		if l.take(i) {
			ledgers = append(ledgers, i)
		}
	}
	return ledgers.Sorted()
}
