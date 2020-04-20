package main

import (
	"fmt"
	"log"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/mongodb"
	"github.com/fsn-dev/crossChain-Bridge/rpc/server"
	"github.com/fsn-dev/crossChain-Bridge/worker"
)

type logWriter struct {
}

func (writer logWriter) Write(bytes []byte) (int, error) {
	return fmt.Print(time.Now().UTC().Format("2006-01-02T15:04:05.999Z") + " " + string(bytes))
}

func init() {
	log.SetFlags(0)
	log.SetOutput(new(logWriter))
}

func main() {
	mongoURL := "test:test@localhost:27917"
	dbName := "testdb"
	mongodb.MongoServerInit(mongoURL, dbName)

	worker.StartWork()

	time.Sleep(100 * time.Millisecond)

	for {
		server.StartAPIServer()
		time.Sleep(time.Duration(60) * time.Second)
		log.Println("restart API server")
	}
}
