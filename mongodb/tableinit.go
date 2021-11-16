package mongodb

import (
	"github.com/anyswap/CrossChain-Bridge/log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	tbSwapins           string = "Swapins"
	tbSwapouts          string = "Swapouts"
	tbSwapinResults     string = "SwapinResults"
	tbSwapoutResults    string = "SwapoutResults"
	tbP2shAddresses     string = "P2shAddresses"
	tbLatestScanInfo    string = "LatestScanInfo"
	tbRegisteredAddress string = "RegisteredAddress"
	tbBlacklist         string = "Blacklist"
	tbLatestSwapNonces  string = "LatestSwapNonces"
	tbSwapHistory       string = "SwapHistory"
	tbUsedRValues       string = "UsedRValues"

	keyOfSrcLatestScanInfo string = "srclatest"
	keyOfDstLatestScanInfo string = "dstlatest"
)

var (
	database *mongo.Database

	collSwapin            *mongo.Collection
	collSwapout           *mongo.Collection
	collSwapinResult      *mongo.Collection
	collSwapoutResult     *mongo.Collection
	collP2shAddress       *mongo.Collection
	collLatestScanInfo    *mongo.Collection
	collRegisteredAddress *mongo.Collection
	collBlacklist         *mongo.Collection
	collLatestSwapNonces  *mongo.Collection
	collSwapHistory       *mongo.Collection
	collUsedRValue        *mongo.Collection
)

func isSwapin(collection *mongo.Collection) bool {
	return collection == collSwapin || collection == collSwapinResult
}

func initCollections() {
	database = client.Database(databaseName)

	initCollection(tbSwapins, &collSwapin, "inittime", "status")
	initCollection(tbSwapouts, &collSwapout, "inittime", "status")
	initCollection(tbSwapinResults, &collSwapinResult, "inittime", "status")
	initCollection(tbSwapoutResults, &collSwapoutResult, "inittime", "status")
	initCollection(tbP2shAddresses, &collP2shAddress, "p2shaddress")
	initCollection(tbLatestScanInfo, &collLatestScanInfo)
	initCollection(tbRegisteredAddress, &collRegisteredAddress)
	initCollection(tbBlacklist, &collBlacklist)
	initCollection(tbLatestSwapNonces, &collLatestSwapNonces, "address")
	initCollection(tbSwapHistory, &collSwapHistory, "txid")
	initCollection(tbUsedRValues, &collUsedRValue)
}

func initCollection(table string, collection **mongo.Collection, indexKey ...string) {
	*collection = database.Collection(table)
	if len(indexKey) != 0 {
		createOneIndex(*collection, indexKey...)
	}
}

func createOneIndex(coll *mongo.Collection, indexes ...string) {
	keys := make([]bson.E, len(indexes))
	for i, index := range indexes {
		keys[i] = bson.E{Key: index, Value: 1}
	}
	model := mongo.IndexModel{Keys: keys}
	_, err := coll.Indexes().CreateOne(clientCtx, model)
	if err != nil {
		log.Error("[mongodb] create indexes failed", "collection", coll.Name(), "indexes", indexes, "err", err)
	}
}
