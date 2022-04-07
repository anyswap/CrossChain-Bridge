// Simple tool to find the seeds for ripple account ids which match a regular expression
package main

import (
	"crypto/rand"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/crypto"
)

var (
	name        = flag.String("name", "ripple", "desired name to appear in ripple account id")
	insensitive = flag.Bool("insensitive", true, "ignore case sensitivity")
	ed25519key  = flag.Bool("ed25519", false, "create an ed25519 key")
)

func checkErr(err error) {
	if err != nil {
		log.Fatalln(err.Error())
	}
}

var count uint64

type Trial struct {
	Seed []byte
	Id   crypto.Hash
	Key  crypto.Key
}

func search(c chan *Trial, target *regexp.Regexp) {
	sequence := uint32(0)
	batch := make([]byte, 1024*4)
	for {
		_, err := rand.Read(batch)
		checkErr(err)
		for i := 0; i < len(batch)-16; i++ {
			trial := &Trial{
				Seed: make([]byte, 16),
			}
			copy(trial.Seed, batch[i:])
			if *ed25519key {
				trial.Key, err = crypto.NewEd25519Key(trial.Seed)
			} else {
				trial.Key, err = crypto.NewECDSAKey(trial.Seed)
			}
			checkErr(err)
			if *ed25519key {
				trial.Id, err = crypto.AccountId(trial.Key, nil)
			} else {
				trial.Id, err = crypto.AccountId(trial.Key, &sequence)
			}
			checkErr(err)
			atomic.AddUint64(&count, 1)
			if target.MatchString(trial.Id.String()) {
				c <- trial
			}
		}
	}
}

func main() {
	flag.Parse()
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	match := *name
	if *insensitive {
		match = "(?i)" + match
	}
	target, err := regexp.Compile(match)
	checkErr(err)
	kill := make(chan os.Signal, 1)
	signal.Notify(kill, os.Interrupt, os.Kill)
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.Printf("Searching for \"%s\" with %d processors", *name, runtime.NumCPU())
	c := make(chan *Trial, 1000)
	for i := 0; i < runtime.NumCPU(); i++ {
		go search(c, target)
	}
	start := time.Now()
	for {
		select {
		case <-kill:
			num := atomic.LoadUint64(&count)
			log.Printf("Tested: %d seeds at %.2f/sec", num, float64(num)/time.Since(start).Seconds())
			return
		case trial := <-c:
			s, err := crypto.NewFamilySeed(trial.Seed)
			checkErr(err)
			log.Println(s, trial.Id)
		}
	}
}
