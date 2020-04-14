package main

import (
	"fmt"
	"log"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/rpc/server"
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
	for {
		server.StartAPIServer()
		time.Sleep(time.Duration(60) * time.Second)
		log.Println("restart API server")
	}
}
