package mongodb

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/log"
	"gopkg.in/mgo.v2"
)

var (
	database *mgo.Database
	session  *mgo.Session

	dialInfo *mgo.DialInfo

	// MgoWaitGroup wait all mongodb related task done
	MgoWaitGroup = new(sync.WaitGroup)

	errSessionIsClosed = errors.New("session is closed")
)

// HasSession has session connected
func HasSession() bool {
	return session != nil
}

// MongoServerInit int mongodb server session
func MongoServerInit(addrs []string, dbname, user, pass string) {
	initDialInfo(addrs, dbname, user, pass)
	mongoConnect()
	initCollections()
	go checkMongoSession()
}

func initDialInfo(addrs []string, db, user, pass string) {
	dialInfo = &mgo.DialInfo{
		Addrs:    addrs,
		Database: db,
		Username: user,
		Password: pass,
	}

	utils.TopWaitGroup.Add(1)
	go utils.WaitAndCleanup(doCleanup)
}

func doCleanup() {
	defer utils.TopWaitGroup.Done()
	if !HasSession() {
		return
	}
	err := session.Fsync(false)
	if err != nil {
		log.Warn("[mongodb] session flush failed", "err", err)
	} else {
		log.Info("[mongodb] session flush success")
	}
	MgoWaitGroup.Wait()
	session.Close()
	session = nil
	log.Info("[mongodb] session close success")
}

func mongoConnect() {
	defer func(oldSession *mgo.Session) {
		if oldSession != nil { // when reconnect
			oldSession.Close()
		}
	}(session)
	log.Info("[mongodb] connect database start.", "addrs", dialInfo.Addrs, "dbName", dialInfo.Database)
	var err error
	for {
		session, err = mgo.DialWithInfo(dialInfo)
		if err == nil {
			break
		}
		log.Warn("[mongodb] dial error", "err", err)
		time.Sleep(1 * time.Second)
	}
	session.SetMode(mgo.Monotonic, true)
	session.SetSafe(&mgo.Safe{FSync: true})
	database = session.DB(dialInfo.Database)
	deinintCollections()
	log.Info("[mongodb] connect database finished.", "dbName", dialInfo.Database)
}

// fix 'read tcp 127.0.0.1:43502->127.0.0.1:27917: i/o timeout'
func checkMongoSession() {
	for {
		time.Sleep(60 * time.Second)
		if err := ensureMongoConnected(); err != nil {
			if !HasSession() {
				return
			}
			log.Info("[mongodb] check session error", "err", err)
			log.Info("[mongodb] reconnect database", "dbName", dialInfo.Database)
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
		if !HasSession() {
			return errSessionIsClosed
		}
		log.Error("[mongodb] session ping error", "err", err)
		log.Info("[mongodb] refresh session.", "dbName", dialInfo.Database)
		session.Refresh()
		database = session.DB(dialInfo.Database)
		deinintCollections()
		err = sessionPing()
	}
	return err
}
