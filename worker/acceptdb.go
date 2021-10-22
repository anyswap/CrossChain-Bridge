package worker

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/leveldb"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/types"
)

const (
	identifierKey = "bridge-identifier"

	allowReswapTimeInterval = 1800 // seconds
)

var (
	lvldbHandle *leveldb.Database
)

func getSwapKeyPrefix(args *tokens.BuildTxArgs) string {
	return strings.ToLower(fmt.Sprintf("%s:%d:%s:%s:", args.SwapID, args.SwapType, args.PairID, args.Bind))
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
func AddAcceptRecord(args *tokens.BuildTxArgs, swapTx string) (err error) {
	if lvldbHandle == nil {
		return nil
	}
	key := []byte(getSwapKeyPrefix(args) + swapTx)
	return lvldbHandle.Put(key, int64ToBytes(now()))
}

// FindAcceptRecords find accept records
func FindAcceptRecords(args *tokens.BuildTxArgs) map[string]int64 {
	if lvldbHandle == nil {
		return nil
	}
	result := make(map[string]int64)
	prefix := []byte(getSwapKeyPrefix(args))
	iter := lvldbHandle.NewIterator(prefix, nil)
	for iter.Next() {
		result[string(iter.Key())] = bytesToInt64(iter.Value())
	}
	iter.Release()
	return result
}

// CheckAcceptRecord check accept record
func CheckAcceptRecord(args *tokens.BuildTxArgs) (err error) {
	if lvldbHandle == nil {
		return nil
	}
	isSwapin := args.SwapType == tokens.SwapinType
	resBridge := tokens.GetCrossChainBridge(!isSwapin)
	alreadySwapped := false
	nowTime := now()

	prefix := []byte(getSwapKeyPrefix(args))
	prefixLen := len(prefix)
	iter := lvldbHandle.NewIterator(prefix, nil)
	for iter.Next() {
		key := string(iter.Key())
		value := bytesToInt64(iter.Value())
		oldSwapTx := key[prefixLen:]
		log.Info("[accept] check saved record", "key", key, "value", value)
		txStatus, errt := resBridge.GetTransactionStatus(oldSwapTx)
		if errt == nil && txStatus != nil && txStatus.BlockHeight > 0 { // on chain
			if txStatus.Receipt != nil { // for eth like chain
				receipt, ok := txStatus.Receipt.(*types.RPCTxReceipt)
				if ok && receipt.IsStatusOk() {
					log.Warn("[accept] found already swapped tx", "key", key, "value", value)
					alreadySwapped = true
					break
				}
			} else {
				log.Warn("[accept] found already swapped tx", "key", key, "value", value)
				alreadySwapped = true
				break
			}
		} else if tx, err := resBridge.GetTransaction(oldSwapTx); err == nil { // in tx pool
			etx, ok := tx.(*types.RPCTransaction)
			if !ok {
				log.Warn("[accept] find already swapped tx in pool", "key", key, "value", value)
				alreadySwapped = true
				break
			}

			if args.Reswapping && value+allowReswapTimeInterval <= nowTime {
				continue // allow reswap old enough
			}

			txNonce := etx.GetAccountNonce()
			argNonce := args.GetTxNonce()
			if txNonce == argNonce {
				continue // allow replace always
			}

			log.Warn("[accept] find already swapped tx in pool", "key", key, "value", value, "txNonce", txNonce, "argNonce", argNonce)
			alreadySwapped = true
			break
		}
	}
	iter.Release()
	if alreadySwapped {
		return errAlreadySwapped
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
