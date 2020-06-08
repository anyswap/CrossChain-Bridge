package mongodb

import (
	"fmt"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/log"
	"gopkg.in/mgo.v2"
)

var (
	database *mgo.Database
	session  *mgo.Session

	mongoURL string
	dbName   string
)

func InitMongodb(mongoURL_, dbName_ string) {
	mongoURL = mongoURL_
	dbName = dbName_
}

func MongoServerInit(mongoURL_, dbName_ string) {
	InitMongodb(mongoURL_, dbName_)
	MongoConnect()
	go CheckMongoSession()
}

func MongoReconnect() {
	log.Info("[mongodb] reconnect database", "dbName", dbName)
	MongoConnect()
	go CheckMongoSession()
}

func MongoConnect() {
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
func CheckMongoSession() {
	for {
		time.Sleep(60 * time.Second)
		EnsureMongoConnected()
	}
}

func EnsureMongoConnected() {
	defer func() {
		if r := recover(); r != nil {
			MongoReconnect()
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
			MongoReconnect()
		}
	}
}
