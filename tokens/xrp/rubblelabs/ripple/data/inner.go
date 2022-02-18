package data

import (
	"fmt"
	"strings"
)

type InnerNodeFunc func(pos int, child Hash256) error

type InnerNode struct {
	Id       Hash256
	Type     NodeType
	Children [16]Hash256
}

type CompressedNodeEntry struct {
	Hash Hash256
	Pos  uint8
}

func (n InnerNode) GetType() string    { return nodeTypes[n.Type] }
func (n InnerNode) Prefix() HashPrefix { return HP_INNER_NODE }
func (n InnerNode) NodeType() NodeType { return n.Type }
func (n InnerNode) Ledger() uint32     { return 0 }
func (n InnerNode) GetHash() *Hash256  { return &n.Id }
func (n InnerNode) NodeId() *Hash256   { return &n.Id }

func (n InnerNode) Each(f InnerNodeFunc) error {
	for i, node := range n.Children {
		if !node.IsZero() {
			if err := f(i, node); err != nil {
				return err
			}
		}
	}
	return nil
}

func (n InnerNode) Count() int {
	var count int
	n.Each(func(i int, child Hash256) error {
		count++
		return nil
	})
	return count
}

func (n InnerNode) String() string {
	var s []string
	n.Each(func(i int, child Hash256) error {
		s = append(s, child.String())
		return nil
	})
	return fmt.Sprintf("%s: [%s]", n.GetType(), strings.Join(s, ","))
}
