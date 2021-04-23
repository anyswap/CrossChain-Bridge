package tools

import (
	"fmt"
	"sort"
)

// WeightedString weighted string
type WeightedString struct {
	Content string
	Weight  uint64
}

// WeightedStringSlice weighted string slice
type WeightedStringSlice []*WeightedString

// Len impl Sortable
func (s WeightedStringSlice) Len() int {
	return len(s)
}

// Swap impl Sortable
func (s WeightedStringSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less impl Sortable
func (s WeightedStringSlice) Less(i, j int) bool {
	return s[i].Weight > s[j].Weight
}

// Add add item
func (s WeightedStringSlice) Add(content string, weight uint64) WeightedStringSlice {
	s = append(s, &WeightedString{
		Content: content,
		Weight:  weight,
	})
	return s
}

// Reverse reverse items
func (s WeightedStringSlice) Reverse() {
	length := s.Len()
	for i := 0; i < length/2; i++ {
		s.Swap(i, length-i-1)
	}
}

// Sort sort items
func (s WeightedStringSlice) Sort() WeightedStringSlice {
	sort.Stable(s)
	return s
}

// GetStrings get strings (commonly sort at first)
func (s WeightedStringSlice) GetStrings() (result []string) {
	for _, wstr := range s {
		result = append(result, wstr.Content)
	}
	return result
}

func (s WeightedStringSlice) String() (result string) {
	result += "["
	for _, wstr := range s {
		result += fmt.Sprintf(" (%v, %v) ", wstr.Content, wstr.Weight)
	}
	result += "]"
	return result
}
