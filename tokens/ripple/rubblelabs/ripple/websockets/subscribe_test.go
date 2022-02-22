package websockets

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/data"
	. "gopkg.in/check.v1"
)

func (s *MessagesSuite) TestLedgerSubscribeResponse(c *C) {
	msg := &SubscribeCommand{}
	readResponseFile(c, msg, "testdata/subscribe_ledger.json")

	// Response fields
	c.Assert(msg.Status, Equals, "success")
	c.Assert(msg.Type, Equals, "response")
	c.Assert(msg.Id, Equals, uint64(3))

	// Result fields
	c.Assert(msg.Result.FeeBase, Equals, uint64(10))
	c.Assert(msg.Result.FeeRef, Equals, uint64(10))
	c.Assert(msg.Result.LedgerSequence, Equals, uint32(6959228))
	c.Assert(msg.Result.LedgerHash.String(), Equals, "E23869F043A46C2735BCA40781A674C5F24460BAC26C6B7475550493A9180200")
	c.Assert(msg.Result.LedgerTime.String(), Equals, "2014-Jun-01 20:56:40")
	c.Assert(msg.Result.ReserveBase, Equals, uint64(20000000))
	c.Assert(msg.Result.ReserveIncrement, Equals, uint64(5000000))
	c.Assert(msg.Result.ValidatedLedgers, Equals, "32570-6959228")
	c.Assert(msg.Result.TxnCount, Equals, uint32(0))
}

func (s *MessagesSuite) TestLedgerStreamMsg(c *C) {
	msg := streamMessageFactory["ledgerClosed"]().(*LedgerStreamMsg)
	readResponseFile(c, msg, "testdata/ledger_stream.json")

	c.Assert(msg.FeeBase, Equals, uint64(10))
	c.Assert(msg.FeeRef, Equals, uint64(10))
	c.Assert(msg.LedgerSequence, Equals, uint32(6959229))
	c.Assert(msg.LedgerHash.String(), Equals, "21EB30937A47EA6B71B63183806FFE9308CCB786137AA00FFB32A7094C6426FA")
	c.Assert(msg.LedgerTime.String(), Equals, "2014-Jun-01 20:56:40")
	c.Assert(msg.ReserveBase, Equals, uint64(20000000))
	c.Assert(msg.ReserveIncrement, Equals, uint64(5000000))
	c.Assert(msg.ValidatedLedgers, Equals, "32570-6959229")
	c.Assert(msg.TxnCount, Equals, uint32(1))
}

func (s *MessagesSuite) TestTransactionSubscribeResponse(c *C) {
	msg := &SubscribeCommand{}
	readResponseFile(c, msg, "testdata/subscribe_transactions.json")

	// Response fields
	c.Assert(msg.Status, Equals, "success")
	c.Assert(msg.Type, Equals, "response")
	c.Assert(msg.Id, Equals, uint64(3))
}

func (s *MessagesSuite) TestTransactionStreamMsg(c *C) {
	msg := streamMessageFactory["transaction"]().(*TransactionStreamMsg)
	readResponseFile(c, msg, "testdata/transactions_stream.json")

	c.Assert(msg.EngineResult.String(), Equals, "tesSUCCESS")
	c.Assert(msg.EngineResultCode, Equals, 0)
	c.Assert(msg.EngineResultMessage, Equals, "The transaction was applied.")
	c.Assert(msg.LedgerHash.String(), Equals, "9B0E9D19E8246BA9B224078B73158ED8970B90DBFAAA68D73A2E0E2899B5AF5A")
	c.Assert(msg.LedgerSequence, Equals, uint32(6959249))
	c.Assert(msg.Status, Equals, "closed")
	c.Assert(msg.Validated, Equals, true)

	offer := msg.Transaction.Transaction.(*data.OfferCreate)

	c.Assert(offer.GetType(), Equals, "OfferCreate")
	c.Assert(offer.Account.String(), Equals, "rPEZyTnSyQyXBCwMVYyaafSVPL8oMtfG6a")
	c.Assert(offer.Fee.String(), Equals, "0.00005")
	c.Assert(msg.Transaction.GetHash().String(), Equals, "25174B56C40B090D4AFCDAC3F07DCCF8A49A096D62CE1CE6864A8624F790F980")
	c.Assert(offer.SigningPubKey.String(), Equals, "0309AEAA170F651170F85C85237CD25CD4200CF91C1C05A9B8A19E72912C2254DF")
	c.Assert(offer.TxnSignature.String(), Equals, "304402201480DBC8253B2E5CCB24001C6E6A0AE73C8FC8D6237B0AA1A5B1CADA92306070022013B02C3CE6E7AFD5F8F348BC40975D15056D414BBC11AD2EA04A65496482212E")
	c.Assert(offer.Sequence, Equals, uint32(753273))

	c.Assert(msg.Transaction.MetaData.TransactionResult.String(), Equals, "tesSUCCESS")
	c.Assert(msg.Transaction.MetaData.TransactionIndex, Equals, uint32(0))
	c.Assert(msg.Transaction.MetaData.AffectedNodes, HasLen, 7)

	offerNodeFields := msg.Transaction.MetaData.AffectedNodes[0].CreatedNode.NewFields.(*data.Offer)
	c.Assert(msg.Transaction.MetaData.AffectedNodes[0].CreatedNode.LedgerEntryType.String(), Equals, "Offer")
	c.Assert(offerNodeFields.TakerGets.String(), Equals, "6400.064/XRP")
	c.Assert(offerNodeFields.TakerPays.String(), Equals, "174.72/CNY/razqQKzJRdB4UxFPWf5NEpEG3WMkmwgcXA")
	c.Assert(offerNodeFields.Account.String(), Equals, "rPEZyTnSyQyXBCwMVYyaafSVPL8oMtfG6a")
	c.Assert(int(*offerNodeFields.OwnerNode), Equals, 0x41FA)
	c.Assert(offerNodeFields.BookDirectory.String(), Equals, "7254404DF6B7FBFFEF34DC38867A7E7DE610B513997B78804D09B2E54D0BD965")
	c.Assert(int(*offerNodeFields.Sequence), Equals, 753273)

	c.Assert(*offer.OfferSequence, Equals, uint32(753240))
	c.Assert(offer.TakerGets.String(), Equals, "6400.064/XRP")
	c.Assert(offer.TakerPays.String(), Equals, "174.72/CNY/razqQKzJRdB4UxFPWf5NEpEG3WMkmwgcXA")
}

func (s *MessagesSuite) TestServerSubscribeResponse(c *C) {
	msg := &SubscribeCommand{}
	readResponseFile(c, msg, "testdata/subscribe_server.json")

	// Response fields
	c.Assert(msg.Status, Equals, "success")
	c.Assert(msg.Type, Equals, "response")
	c.Assert(msg.Id, Equals, uint64(3))

	// Result fields
	c.Assert(msg.Result.Status, Equals, "full")
	c.Assert(msg.Result.LoadBase, Equals, uint64(256))
	c.Assert(msg.Result.LoadFactor, Equals, uint64(256))
}

func (s *MessagesSuite) TestServerStreamMsg(c *C) {
	msg := streamMessageFactory["serverStatus"]().(*ServerStreamMsg)
	readResponseFile(c, msg, "testdata/server_stream.json")

	c.Assert(msg.Status, Equals, "syncing")
	c.Assert(msg.LoadBase, Equals, uint64(256))
	c.Assert(msg.LoadFactor, Equals, uint64(256))
}

func (s *MessagesSuite) TestProposedTransactionStreamMsg(c *C) {
	msg := streamMessageFactory["transaction"]().(*TransactionStreamMsg)
	readResponseFile(c, msg, "testdata/proposed_transaction_stream.json")

	c.Assert(msg.EngineResult.String(), Equals, "tesSUCCESS")
	c.Assert(msg.EngineResultCode, Equals, 0)
	c.Assert(msg.EngineResultMessage, Equals, "The transaction was applied.")
	c.Assert(msg.Status, Equals, "proposed")
	c.Assert(msg.Validated, Equals, false)

	offer := msg.Transaction.Transaction.(*data.OfferCreate)

	c.Assert(offer.GetType(), Equals, "OfferCreate")
	c.Assert(offer.Account.String(), Equals, "rHsZHqa5oMQNL5hFm4kfLd47aEMYjPstpg")
	c.Assert(offer.Fee.String(), Equals, "0.011")
	c.Assert(msg.Transaction.GetHash().String(), Equals, "2E55F1B9A147B647F6699FD877CA2AEC5A7A0A22A6F2D609DB6DABC445EF9862")
	c.Assert(offer.SigningPubKey.String(), Equals, "025718736160FA6632F48EA4354A35AB0340F8D7DC7083799B9C57C3E937D71851")
	c.Assert(offer.TxnSignature.String(), Equals, "30440220572BCA1D98177F3A7082B4A77EBAA4D5977237CE0591678CFECEC8D3C0457FB802200E400395F505F07F7DCBE46F74D0D5CAFCD12D6078F5FD98DE1D72D85B887DC1")
	c.Assert(offer.Sequence, Equals, uint32(10379931))

	c.Assert(*offer.OfferSequence, Equals, uint32(10379905))
	c.Assert(offer.TakerGets.String(), Equals, "28865.168964/XRP")
	c.Assert(offer.TakerPays.String(), Equals, "4285.465077979/CNY/razqQKzJRdB4UxFPWf5NEpEG3WMkmwgcXA")
}

func BenchmarkProposedTransactionStreamJSON(b *testing.B) {
	bites, err := ioutil.ReadFile("testdata/proposed_transaction_stream.json")
	if err != nil {
		b.Error(err)
	}
	for i := 0; i < b.N; i++ {
		var tsm TransactionStreamMsg
		if err := json.Unmarshal(bites, &tsm); err != nil {
			b.Error(err)
		}
	}
}
func BenchmarkTransactionStreamJSON(b *testing.B) {
	bites, err := ioutil.ReadFile("testdata/transactions_stream.json")
	if err != nil {
		b.Error(err)
	}
	for i := 0; i < b.N; i++ {
		var tsm TransactionStreamMsg
		if err := json.Unmarshal(bites, &tsm); err != nil {
			b.Error(err)
		}
	}
}
