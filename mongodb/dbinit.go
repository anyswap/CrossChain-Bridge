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
	initCollections()
	go checkMongoSession()
}

func initMongodb(url, db string) {
	mongoURL = url
	dbName = db
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
		log.Warn("[mongodb] dial error", "err", err)
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
		if err := ensureMongoConnected(); err != nil {
			log.Info("[mongodb] check session error", "err", err)
			log.Info("[mongodb] reconnect database", "dbName", dbName)
			mongoConnect()
		}
	}
}

func sessionPing() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recover from error %v", r)
		}
	}()
	for i := 0; i < 6; i++ {
		err = session.Ping()
		if err == nil {
			break
		}
		time.Sleep(10 * time.Second)
	}
	return err
}

func ensureMongoConnected() (err error) {
	err = sessionPing()
	if err != nil {
		log.Error("[mongodb] session ping error", "err", err)
		log.Info("[mongodb] refresh session.", "dbName", dbName)
		session.Refresh()
		database = session.DB(dbName)
		deinintCollections()
		err = sessionPing()
	}
	return err
}
