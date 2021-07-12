package mongodb

import (
	"gopkg.in/mgo.v2"
)

var (
	collSwapin            *mgo.Collection
	collSwapout           *mgo.Collection
	collSwapinResult      *mgo.Collection
	collSwapoutResult     *mgo.Collection
	collP2shAddress       *mgo.Collection
	collSwapStatistics    *mgo.Collection
	collLatestScanInfo    *mgo.Collection
	collRegisteredAddress *mgo.Collection
	collBlacklist         *mgo.Collection
	collLatestSwapNonces  *mgo.Collection
	collSwapHistory       *mgo.Collection
	collUsedRValue        *mgo.Collection
)

func isSwapin(collection *mgo.Collection) bool {
	return collection == collSwapin || collection == collSwapinResult
}

// do this when reconnect to the database
func deinintCollections() {
	collSwapin = database.C(tbSwapins)
	collSwapout = database.C(tbSwapouts)
	collSwapinResult = database.C(tbSwapinResults)
	collSwapoutResult = database.C(tbSwapoutResults)
	collP2shAddress = database.C(tbP2shAddresses)
	collSwapStatistics = database.C(tbSwapStatistics)
	collLatestScanInfo = database.C(tbLatestScanInfo)
	collRegisteredAddress = database.C(tbRegisteredAddress)
	collBlacklist = database.C(tbBlacklist)
	collLatestSwapNonces = database.C(tbLatestSwapNonces)
	collSwapHistory = database.C(tbSwapHistory)
	collUsedRValue = database.C(tbUsedRValues)
}

func initCollections() {
	initCollection(tbSwapins, &collSwapin, "inittime", "status")
	initCollection(tbSwapouts, &collSwapout, "inittime", "status")
	initCollection(tbSwapinResults, &collSwapinResult, "from", "inittime")
	initCollection(tbSwapoutResults, &collSwapoutResult, "from", "inittime")
	initCollection(tbP2shAddresses, &collP2shAddress, "p2shaddress")
	initCollection(tbSwapStatistics, &collSwapStatistics)
	initCollection(tbLatestScanInfo, &collLatestScanInfo)
	initCollection(tbRegisteredAddress, &collRegisteredAddress)
	initCollection(tbBlacklist, &collBlacklist)
	initCollection(tbLatestSwapNonces, &collLatestSwapNonces, "address")
	initCollection(tbSwapHistory, &collSwapHistory, "txid")
	initCollection(tbUsedRValues, &collUsedRValue)

	initDefaultValue()
}

func initCollection(table string, collection **mgo.Collection, indexKey ...string) {
	*collection = database.C(table)
	if len(indexKey) != 0 && indexKey[0] != "" {
		_ = (*collection).EnsureIndexKey(indexKey...)
	}
}

func initDefaultValue() {
	_ = collLatestScanInfo.Insert(
		&MgoLatestScanInfo{
			Key: keyOfSrcLatestScanInfo,
		},
		&MgoLatestScanInfo{
			Key: keyOfDstLatestScanInfo,
		},
	)
}
