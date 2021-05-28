package worker

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/leveldb"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

const (
	identifierKey = "bridge-identifier"

	allowMismatchNonceTimeInterval = 1800 // seconds
)

var (
	lvldbHandle *leveldb.Database
)

func getEthAcceptRecordKey(args *tokens.BuildTxArgs, isPrefix bool) string {
	if isPrefix {
		return fmt.Sprintf("%s:%d", args.SwapID, args.SwapType)
	}
	return fmt.Sprintf("%s:%d:%d", args.SwapID, args.SwapType, args.GetTxNonce())
}

func int64ToBytes(i int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}

func bytesToInt64(buf []byte) int64 {
	return int64(binary.BigEndian.Uint64(buf))
}

// AddAcceptRecord add accept record
func AddAcceptRecord(args *tokens.BuildTxArgs) (err error) {
	if lvldbHandle == nil {
		return nil
	}
	swapNonce := args.GetTxNonce()
	if swapNonce == 0 {
		return nil
	}
	key := []byte(getEthAcceptRecordKey(args, false))
	err = lvldbHandle.Put(key, int64ToBytes(now()))
	if err != nil {
		log.Warn("add accept record failed", "key", string(key), "err", err)
	} else {
		log.Info("add accept record success", "key", string(key))
	}
	return err
}

// GetAcceptRecord get accept record
func GetAcceptRecord(args *tokens.BuildTxArgs) (int64, error) {
	key := []byte(getEthAcceptRecordKey(args, false))
	bs, err := lvldbHandle.Get(key)
	if err != nil {
		return 0, err
	}
	return bytesToInt64(bs), nil
}

// FindAcceptRecords find accept records
func FindAcceptRecords(args *tokens.BuildTxArgs) (result map[string]int64, err error) {
	result = make(map[string]int64)
	prefix := []byte(getEthAcceptRecordKey(args, true))
	iter := lvldbHandle.NewIterator(prefix, nil)
	for iter.Next() {
		result[string(iter.Key())] = bytesToInt64(iter.Value())
	}
	iter.Release()
	err = iter.Error()
	return result, err
}

// CheckAcceptRecord check accept record
func CheckAcceptRecord(args *tokens.BuildTxArgs) (err error) {
	if lvldbHandle == nil {
		return nil
	}
	swapNonce := args.GetTxNonce()
	if swapNonce == 0 {
		return nil
	}
	_, err = GetAcceptRecord(args)
	if err == nil {
		return nil
	}
	nowTime := now()
	prefix := []byte(getEthAcceptRecordKey(args, true))
	iter := lvldbHandle.NewIterator(prefix, nil)
	for iter.Next() {
		lastTime := bytesToInt64(iter.Value())
		if lastTime+allowMismatchNonceTimeInterval > nowTime {
			log.Warn("find record with nonce mismatch recently", "args", args, "oldKey", string(iter.Key()), "oldTime", lastTime, "nowTime", nowTime)
			return errNonceMismatch
		}
	}
	return nil
}

func getLeveldbPath() string {
	dataDir := params.GetDataDir()
	identifier := params.GetIdentifier()
	path := strings.ToLower(fmt.Sprintf("%s/%s", dataDir, identifier))
	return path
}

func closeLeveldb() {
	if lvldbHandle == nil {
		return
	}
	err := lvldbHandle.Close()
	if err != nil {
		log.Error("close leveldb failed", "err", err)
	} else {
		log.Info("close leveldb success")
	}
}

func openLeveldb() {
	if lvldbHandle != nil {
		log.Crit("forbid to reopen accept database")
	}

	if params.GetDataDir() == "" || params.GetIdentifier() == "" {
		log.Info("ignore open leveldb", "datadir", params.GetDataDir(), "identifier", params.GetIdentifier())
		return
	}

	path := getLeveldbPath()
	db, err := leveldb.New(path, 16, 16, false)
	if err != nil {
		log.Crit("open accept database failed", "path", path, "err", err)
	}
	log.Info("open accept database success", "path", path)

	configIdentifier := params.GetIdentifier()

	identifierVal, err := db.Get([]byte(identifierKey))
	identifierInDB := string(identifierVal)
	if err != nil {
		if !leveldb.IsNotFoundErr(err) {
			log.Fatal("get identifier from database failed", "err", err)
		}
		err = db.Put([]byte(identifierKey), []byte(configIdentifier)) // init identifier
		if err != nil {
			log.Fatal("write identifier to database failed", "identifier", configIdentifier, "err", err)
		} else {
			log.Info("write identifier to database success", "identifier", configIdentifier)
		}
	} else {
		log.Info("get identifier from database success", "identifier", identifierInDB)
		if identifierInDB != configIdentifier {
			log.Fatal("identifier mismatch", "indb", identifierInDB, "inconfig", configIdentifier)
		}
	}

	lvldbHandle = db
}
