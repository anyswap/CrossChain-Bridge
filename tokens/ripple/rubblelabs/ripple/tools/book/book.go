package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/data"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/terminal"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/websockets"
)

const usage = `Usage: book [currency currency] [options]

Examples:

book XRP USD/rvYAfWj5gh67oV6fW32ZzP3Aw4Eubs59B
	Show all offers for where the taker pays USD/rvYAfWj5gh67oV6fW32ZzP3Aw4Eubs59B and the taker gets XRP

Options:
`

var (
	host = flag.String("host", "wss://s1.ripple.com:443", "websockets host")
)

func showUsage() {
	fmt.Println(usage)
	flag.PrintDefaults()
	os.Exit(1)
}

func checkErr(err error) {
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) != 3 {
		showUsage()
	}
	flag.CommandLine.Parse(os.Args[3:])

	remote, err := websockets.NewRemote(*host)
	checkErr(err)
	gets, err := data.NewAsset(os.Args[1])
	checkErr(err)
	pays, err := data.NewAsset(os.Args[2])
	checkErr(err)
	var zeroAccount data.Account
	result, err := remote.BookOffers(zeroAccount, "closed", *pays, *gets)
	checkErr(err)
	// fmt.Println(*result.LedgerSequence) //TODO: wait for nikb fix
	for _, offer := range result.Offers {
		terminal.Println(offer, terminal.Default)
	}
}
