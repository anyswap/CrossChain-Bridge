package data

import (
	"encoding/json"

	internal "github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/testing"
	. "gopkg.in/check.v1"
)

type CodecSuite struct{}

var _ = Suite(&CodecSuite{})

func dump(test internal.TestData, v interface{}) CommentInterface {
	out, _ := json.Marshal(v)
	return Commentf("Test: %s\nJSON:%s\n", test.Description, string(out))
}

func (s *CodecSuite) TestParseTransactions(c *C) {
	for _, test := range internal.Transactions {
		tx, err := ReadTransaction(test.Reader())
		c.Assert(err, IsNil)
		msg := dump(test, tx)
		signable := tx.GetTransactionType() != SET_FEE && tx.GetTransactionType() != AMENDMENT
		ok, err := CheckSignature(tx)
		if signable {
			c.Assert(err, IsNil, msg)
		}
		c.Assert(ok, Equals, signable, msg)
		_, raw, err := Raw(tx)
		c.Assert(err, IsNil, msg)
		c.Assert(string(b2h(raw)), Equals, test.Encoded, msg)
	}
}

func (s *CodecSuite) TestValidations(c *C) {
	for _, test := range internal.Validations {
		v, err := ReadValidation(test.Reader())
		c.Assert(err, IsNil)
		msg := dump(test, v)
		ok, err := CheckSignature(v)
		c.Assert(ok, Equals, true, msg)
		c.Assert(err, IsNil, msg)
		_, raw, err := Raw(v)
		c.Assert(err, IsNil, msg)
		c.Assert(string(b2h(raw)), Equals, test.Encoded, msg)
	}
}

func (s *CodecSuite) TestParseNodes(c *C) {
	for _, test := range internal.Nodes {
		nodeId, err := NewHash256(test.NodeId())
		c.Assert(err, IsNil)
		n, err := ReadPrefix(test.Reader(), *nodeId)
		msg := dump(test, n)
		c.Assert(err, IsNil, msg)
		c.Check(n.NodeId().String(), Equals, nodeId.String(), Commentf(test.Description))
		c.Assert(err, IsNil, msg)
		generatedNodeId, value, err := Node(n)
		c.Assert(err, IsNil, msg)
		c.Assert(string(b2h(value))[16:], Equals, test.Encoded[16:], msg)
		c.Assert(generatedNodeId.String(), Equals, nodeId.String(), Commentf(test.Description))
		c.Assert(n.GetHash().IsZero(), Equals, false)
	}
}

func (s *CodecSuite) TestBadNodes(c *C) {
	for _, test := range internal.BadNodes {
		nodeid, err := NewHash256(test.NodeId())
		c.Assert(err, IsNil)
		n, err := ReadPrefix(test.Reader(), *nodeid)
		msg := dump(test, n)
		c.Assert(err, Not(IsNil), msg)
	}
}

func (s *CodecSuite) TestParseMetaData(c *C) {
	for _, test := range internal.Nodes {
		nodeId, err := NewHash256(test.NodeId())
		c.Assert(err, IsNil)
		n, err := ReadPrefix(test.Reader(), *nodeId)
		msg := dump(test, n)
		c.Assert(err, IsNil, msg)
		txm, ok := n.(*TransactionWithMetaData)
		if !ok {
			continue
		}
		for _, a := range txm.MetaData.AffectedNodes {
			effect, current, previous, _ := a.AffectedNode()
			c.Assert(effect, Not(IsNil))
			c.Assert(current, Not(IsNil))
			c.Assert(previous, Not(IsNil))
		}
	}
}
