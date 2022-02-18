package data

import (
	. "gopkg.in/check.v1"
)

type LedgerSetSuite struct{}

var _ = Suite(&LedgerSetSuite{})

func (s *LedgerSetSuite) TestLedgerSet(c *C) {
	l := NewLedgerSet(32570, 32670)
	l.Set(32570)
	l.Set(32572)
	l.Set(32670)
	ten := l.TakeBottom(10)
	c.Assert(len(ten), Equals, 10)
	c.Assert(ten, DeepEquals, LedgerSlice{32571, 32573, 32574, 32575, 32576, 32577, 32578, 32579, 32580, 32581})
	topTen := l.TakeTop(10)
	c.Assert(len(topTen), Equals, 10)
	c.Assert(topTen, DeepEquals, LedgerSlice{32660, 32661, 32662, 32663, 32664, 32665, 32666, 32667, 32668, 32669})
	tooLarge := l.TakeBottom(105)
	c.Assert(len(tooLarge), Equals, 78)
	tooLargeTop := l.TakeTop(105)
	c.Assert(len(tooLargeTop), Equals, 0)
}

func (s *LedgerSetSuite) TestLedgerSetMiddle(c *C) {
	l := NewLedgerSet(32570, 32670)
	r := &LedgerRange{
		Start: 32580,
		End:   32620,
		Max:   4,
	}
	middle := l.TakeMiddle(r)
	c.Assert(len(middle), Equals, 4)
	c.Assert(middle, DeepEquals, LedgerSlice{32580, 32581, 32582, 32583})
	c.Assert(l.Max(), Equals, uint32(32670))
	l.Extend(32690)
	c.Assert(l.Max(), Equals, uint32(32690))
	l.Extend(40000)
	c.Assert(l.Max(), Equals, uint32(40000))
}

// func (s *LedgerSetSuite) TestLargeLedgerSet(c *C) {
// 	l := NewLedgerSet(32570, 5500000)
// 	l.Set(32570)
// 	l.Set(5500000)
// 	for {
// 		left := l.TakeBottom(10000)
// 		if len(left) == 0 {
// 			break
// 		}
// 		fmt.Println(left[0])
// 		for _, n := range left {
// 			l.Set(n)
// 		}
// 	}
// 	fmt.Println(l.String())
// }
