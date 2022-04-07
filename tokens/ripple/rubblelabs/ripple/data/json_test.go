package data

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/juju/testing/checkers"
	. "gopkg.in/check.v1"
)

type JSONSuite struct{}

var _ = Suite(&JSONSuite{})

func compare(c *C, filename string, expected, obtained []byte) {
	expectedFields := make(map[string]interface{})
	err := json.Unmarshal(expected, &expectedFields)
	c.Assert(err, IsNil)

	obtainedFields := make(map[string]interface{})
	err = json.Unmarshal(obtained, &obtainedFields)
	c.Assert(err, IsNil)
	c.Check(obtainedFields, checkers.DeepEquals, expectedFields)
}

func (s *JSONSuite) TestTransactionsJSON(c *C) {
	files, err := filepath.Glob("testdata/transaction_*.json")
	c.Assert(err, IsNil)
	for _, f := range files {
		b, err := ioutil.ReadFile(f)
		c.Assert(err, IsNil)
		var txm TransactionWithMetaData
		c.Assert(json.Unmarshal(b, &txm), IsNil)
		out, err := json.MarshalIndent(txm, "", "  ")
		c.Assert(err, IsNil)
		compare(c, f, b, out)
	}
}
