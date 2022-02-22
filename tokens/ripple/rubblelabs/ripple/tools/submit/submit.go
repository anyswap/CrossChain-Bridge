package main

import (
	"flag"
	"log"
	"os"

	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/config"
)

var (
	host = flag.String("host", "wss://s-east.ripple.com:443", "websockets host")
)

func checkErr(err error) {
	if err != nil {
		log.Fatalln(err.Error())
	}
}

func main() {
	flag.Parse()
	actions, err := config.Parse(os.Stdin)
	checkErr(err)
	checkErr(actions.Prepare())
	checkErr(actions.Submit(*host))
	log.Printf("Submitted %d transactions", actions.Count())
}
