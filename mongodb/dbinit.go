// Package mongodb is a wrapper of mongo-go-driver that
// defines the collections and CRUD apis on them.
package mongodb

import (
	"context"
	"sync"
	"time"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	client    *mongo.Client
	clientCtx = context.Background()

	appIdentifier string
	databaseName  string

	// MgoWaitGroup wait all mongodb related task done
	MgoWaitGroup = new(sync.WaitGroup)
)

// HasClient has client
func HasClient() bool {
	return client != nil
}

// MongoServerInit int mongodb server session
func MongoServerInit(appName string, hosts []string, dbName, user, pass string) {
	appIdentifier = appName
	databaseName = dbName

	clientOpts := &options.ClientOptions{
		AppName: &appName,
		Hosts:   hosts,
		Auth: &options.Credential{
			AuthSource: dbName,
			Username:   user,
			Password:   pass,
		},
	}

	if err := connect(clientOpts); err != nil {
		log.Fatal("[mongodb] connect database failed", "hosts", hosts, "dbName", dbName, "appName", appName, "err", err)
	}

	log.Info("[mongodb] connect database success", "hosts", hosts, "dbName", dbName, "appName", appName)

	utils.TopWaitGroup.Add(1)
	go utils.WaitAndCleanup(doCleanup)
}

func doCleanup() {
	defer utils.TopWaitGroup.Done()
	MgoWaitGroup.Wait()

	err := client.Disconnect(clientCtx)
	if err != nil {
		log.Error("[mongodb] close connection failed", "appName", appIdentifier, "err", err)
	} else {
		log.Info("[mongodb] close connection success", "appName", appIdentifier)
	}
}

func connect(opts *options.ClientOptions) (err error) {
	ctx, cancel := context.WithTimeout(clientCtx, 10*time.Second)
	defer cancel()

	client, err = mongo.Connect(ctx, opts)
	if err != nil {
		return err
	}

	err = client.Ping(clientCtx, nil)
	if err != nil {
		return err
	}

	initCollections()
	return nil
}
