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
)

func MongoServerInit(mongoURL, dbName string) {
	var err error
	url := fmt.Sprintf("mongodb://%v/%v", mongoURL, dbName)
	log.Println("mongodb init server start.")
	for {
		session, err = mgo.Dial(url)
		if err == nil {
			break
		}
		log.Printf("mongodb dial error, err=%v\n", err)
		time.Sleep(time.Duration(1) * time.Second)
	}
	session.SetMode(mgo.Monotonic, true)
	session.SetSafe(&mgo.Safe{FSync: true})
	database = session.DB(dbName)
	log.Println("mongodb init server finished.")
}
