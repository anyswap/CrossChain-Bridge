package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/data"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/websockets"
)

func checkErr(err error, quit bool) {
	if err != nil {
		log.Println(err.Error())
		if quit {
			os.Exit(1)
		}
	}
}

var (
	host    = flag.String("host", "wss://s-east.ripple.com:443", "websockets host to connect to")
	account = flag.String("account", "", "optional account to monitor")
)

func stream(r *websockets.Remote, filter *data.Account) {
	confirmation, err := r.Subscribe(true, true, false, false)
	checkErr(err, true)
	log.Printf("Subscribed at: %d ", confirmation.LedgerSequence)

	for {
		msg, ok := <-r.Incoming
		if !ok {
			return
		}
		switch msg := msg.(type) {
		case *websockets.TransactionStreamMsg:
			msg.Transaction.LedgerSequence = msg.LedgerSequence
			trades, err := data.NewTradeSlice(&msg.Transaction)
			checkErr(err, false)
			if filter != nil {
				trades = trades.Filter(*filter)
			}
			for _, trade := range trades {
				log.Println(trade)
			}
		}
	}
}

func download(r *websockets.Remote, start, end uint32, filter *data.Account) {
	for ledger := start; ledger <= end; ledger++ {
		result, err := r.Ledger(ledger, true)
		checkErr(err, true)
		for _, tx := range result.Ledger.Transactions {
			tx.LedgerSequence = result.Ledger.LedgerSequence
			trades, err := data.NewTradeSlice(tx)
			checkErr(err, true)
			if filter != nil {
				trades = trades.Filter(*filter)
			}
			for _, trade := range trades {
				fmt.Println(trade)
			}
		}
	}
}

func parseledger(s string) uint32 {
	ledger, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		log.Fatalf("bad ledger: %s", s)
	}
	return uint32(ledger)
}

func main() {
	flag.Parse()
	var (
		filter *data.Account
		err    error
	)
	if len(*account) > 0 {
		filter, err = data.NewAccountFromAddress(*account)
		checkErr(err, true)
	}

	r, err := websockets.NewRemote(*host)
	checkErr(err, true)
	switch flag.NArg() {
	case 0:
		stream(r, filter)
	case 1:
		ledger := parseledger(flag.Arg(0))
		download(r, ledger, ledger, filter)
	case 2:
		start, end := parseledger(flag.Arg(0)), parseledger(flag.Arg(1))
		download(r, start, end, filter)
	default:
		log.Fatalf("bad number of arguments: %d", flag.NArg())
	}
}
