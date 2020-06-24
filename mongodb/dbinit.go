package mongodb

import (
	"fmt"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"gopkg.in/mgo.v2"
)

var (
	database *mgo.Database
	session  *mgo.Session

	mongoURL string
	dbName   string
)

// MongoServerInit int mongodb server session
func MongoServerInit(mongourl, dbname string) {
	initMongodb(mongourl, dbname)
	mongoConnect()
	InitCollections()
	go checkMongoSession()
}

func initMongodb(url, db string) {
	mongoURL = url
	dbName = db
}

func mongoReconnect() {
	log.Info("[mongodb] reconnect database", "dbName", dbName)
	mongoConnect()
	go checkMongoSession()
}

func mongoConnect() {
	if session != nil { // when reconnect
		session.Close()
	}
	log.Info("[mongodb] connect database start.", "dbName", dbName)
	url := fmt.Sprintf("mongodb://%v/%v", mongoURL, dbName)
	var err error
	for {
		session, err = mgo.Dial(url)
		if err == nil {
			break
		}
		log.Printf("[mongodb] dial error, err=%v\n", err)
		time.Sleep(1 * time.Second)
	}
	session.SetMode(mgo.Monotonic, true)
	session.SetSafe(&mgo.Safe{FSync: true})
	database = session.DB(dbName)
	deinintCollections()
	log.Info("[mongodb] connect database finished.", "dbName", dbName)
}

// fix 'read tcp 127.0.0.1:43502->127.0.0.1:27917: i/o timeout'
func checkMongoSession() {
	for {
		time.Sleep(60 * time.Second)
		ensureMongoConnected()
	}
}

func ensureMongoConnected() {
	defer func() {
		if r := recover(); r != nil {
			mongoReconnect()
		}
	}()
	err := session.Ping()
	if err != nil {
		log.Info("[mongodb] refresh session.", "dbName", dbName)
		session.Refresh()
		err = session.Ping()
		if err == nil {
			database = session.DB(dbName)
			deinintCollections()
		} else {
			mongoReconnect()
		}
	}
}
