// Tool to explain transactions either individually, in a ledger or belonging to an account.
package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/data"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/terminal"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/websockets"
)

const usage = `Usage: explain [tx hash|ledger sequence|ripple address|-] [options]

Examples:

explain 6000000
	Explain all transactions for ledger 6000000

explain rrpNnNLKrartuEqfJGpqyDwPj1AFPg9vn1
	Explain all transactions for account rrpNnNLKrartuEqfJGpqyDwPj1AFPg9vn1

explain 955A4C0B7C66FC97EA4C72634CDCDBF50BB17AAA647EC6C8C592788E5B95173C
	Explain transaction 955A4C0B7C66FC97EA4C72634CDCDBF50BB17AAA647EC6C8C592788E5B95173C

explain -
	Explain binary transactions received through stdin

Options:
`

var argumentRegex = regexp.MustCompile(`(^[0-9a-fA-F]{64}$)|(^\d+$)|(^[r][a-km-zA-HJ-NP-Z0-9]{26,34}$)|(-)`)

var (
	flags        = flag.CommandLine
	host         = flags.String("host", "wss://s-east.ripple.com:443", "websockets host")
	trades       = flag.Bool("t", false, "hide trades")
	balances     = flag.Bool("b", false, "hide balances")
	paths        = flag.Bool("p", false, "hide paths")
	transactions = flag.Bool("tx", false, "hide transactions")
	pageSize     = flag.Int("page_size", 20, "page size for account_tx requests")
)

func showUsage() {
	fmt.Println(usage)
	flags.PrintDefaults()
	os.Exit(1)
}

func checkErr(err error) {
	if err != nil {
		terminal.Println(err.Error(), terminal.Default)
		os.Exit(1)
	}
}

func explain(txm *data.TransactionWithMetaData, flag terminal.Flag) {
	if !*transactions {
		terminal.Println(txm, flag)
	}
	if !*paths {
		for _, path := range txm.PathSet() {
			terminal.Println(path, flag|terminal.Indent)
		}
	}
	if !*trades {
		trades, err := data.NewTradeSlice(txm)
		checkErr(err)
		for _, trade := range trades {
			terminal.Println(trade, flag|terminal.Indent)
		}
	}
	if !*balances {
		balanceMap, err := txm.Balances()
		checkErr(err)
		for account, balances := range balanceMap {
			terminal.Println(account, flag|terminal.Indent)
			for _, balance := range *balances {
				terminal.Println(balance, flag|terminal.DoubleIndent)
			}
		}
	}
}

func main() {
	if len(os.Args) == 1 {
		showUsage()
	}
	flags.Parse(os.Args[2:])
	matches := argumentRegex.FindStringSubmatch(os.Args[1])
	r, err := websockets.NewRemote(*host)
	checkErr(err)
	log.Infoln("Connected to: ", *host)
	switch {
	case len(matches) == 0:
		showUsage()
	case len(matches[1]) > 0:
		hash, err := data.NewHash256(matches[1])
		checkErr(err)
		fmt.Println("Getting transaction: ", hash.String())
		result, err := r.Tx(*hash)
		checkErr(err)
		explain(&result.TransactionWithMetaData, terminal.Default)
	case len(matches[2]) > 0:
		seq, err := strconv.ParseUint(matches[2], 10, 32)
		checkErr(err)
		ledger, err := r.Ledger(seq, true)
		checkErr(err)
		fmt.Println("Getting transactions for: ", seq)
		for _, txm := range ledger.Ledger.Transactions {
			explain(txm, terminal.Default)
		}
	case len(matches[3]) > 0:
		account, err := data.NewAccountFromAddress(matches[3])
		checkErr(err)
		fmt.Println("Getting transactions for: ", account.String())
		for txm := range r.AccountTx(*account, *pageSize, -1, -1) {
			explain(txm, terminal.ShowLedgerSequence)
		}
	case len(matches[4]) > 0:
		r := bufio.NewReader(os.Stdin)
		for line, err := r.ReadString('\n'); err == nil; line, err = r.ReadString('\n') {
			// TODO: Accept nodeid:nodedata format
			b, err := hex.DecodeString(line[:len(line)-1])
			checkErr(err)
			var nodeid data.Hash256
			v, err := data.ReadPrefix(bytes.NewReader(b), nodeid)
			checkErr(err)
			terminal.Println(v, terminal.Default)
		}
	}
}
