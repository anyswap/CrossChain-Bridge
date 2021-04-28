package mongodb

const (
	tbSwapins           string = "Swapins"
	tbSwapouts          string = "Swapouts"
	tbSwapinResults     string = "SwapinResults"
	tbSwapoutResults    string = "SwapoutResults"
	tbP2shAddresses     string = "P2shAddresses"
	tbSwapStatistics    string = "SwapStatistics"
	tbLatestScanInfo    string = "LatestScanInfo"
	tbRegisteredAddress string = "RegisteredAddress"
	tbBlacklist         string = "Blacklist"
	tbSwapAgreement     string = "SwapAgreement"
	tbSolanaScannedTx   string = "SolanaScannedTx"
	tbLatestSwapNonces  string = "LatestSwapNonces"

	keyOfSrcLatestScanInfo string = "srclatest"
	keyOfDstLatestScanInfo string = "dstlatest"
)

// MgoSwap registered swap
type MgoSwap struct {
	Key       string     `bson:"_id"` // txid + pairid + bind
	PairID    string     `bson:"pairid"`
	TxID      string     `bson:"txid"`
	TxTo      string     `bson:"txto"`
	TxType    uint32     `bson:"txtype"`
	Bind      string     `bson:"bind"`
	Status    SwapStatus `bson:"status"`
	InitTime  int64      `bson:"inittime"`
	Timestamp int64      `bson:"timestamp"`
	Memo      string     `bson:"memo"`
}

// MgoSwapResult swap result (verified swap)
type MgoSwapResult struct {
	Key        string     `bson:"_id"` // txid + pairid + bind
	PairID     string     `bson:"pairid"`
	TxID       string     `bson:"txid"`
	TxTo       string     `bson:"txto"`
	TxHeight   uint64     `bson:"txheight"`
	TxTime     uint64     `bson:"txtime"`
	From       string     `bson:"from"`
	To         string     `bson:"to"`
	Bind       string     `bson:"bind"`
	Value      string     `bson:"value"`
	SwapTx     string     `bson:"swaptx"`
	OldSwapTxs []string   `bson:"oldswaptxs"`
	SwapHeight uint64     `bson:"swapheight"`
	SwapTime   uint64     `bson:"swaptime"`
	SwapValue  string     `bson:"swapvalue"`
	SwapType   uint32     `bson:"swaptype"`
	SwapNonce  uint64     `bson:"swapnonce"`
	Status     SwapStatus `bson:"status"`
	InitTime   int64      `bson:"inittime"`
	Timestamp  int64      `bson:"timestamp"`
	Memo       string     `bson:"memo"`
}

// SwapResultUpdateItems swap update items
type SwapResultUpdateItems struct {
	SwapTx     string
	OldSwapTxs []string
	SwapHeight uint64
	SwapTime   uint64
	SwapValue  string
	SwapType   uint32
	SwapNonce  uint64
	Status     SwapStatus
	Timestamp  int64
	Memo       string
}

// MgoP2shAddress key is the bind address
type MgoP2shAddress struct {
	Key         string `bson:"_id"`
	P2shAddress string `bson:"p2shaddress"`
}

// MgoRegisteredAddress key is address (in whitelist)
type MgoRegisteredAddress struct {
	Key       string `bson:"_id"`
	Timestamp int64  `bson:"timestamp"`
}

// MgoSwapStatistics swap statistics
type MgoSwapStatistics struct {
	Key                string `bson:"_id"` // pairid
	PairID             string `bson:"pairid"`
	StableSwapinCount  int    `bson:"swapincount"`
	TotalSwapinValue   string `bson:"totalswapinvalue"`
	TotalSwapinFee     string `bson:"totalswapinfee"`
	StableSwapoutCount int    `bson:"swapoutcount"`
	TotalSwapoutValue  string `bson:"totalswapoutvalue"`
	TotalSwapoutFee    string `bson:"totalswapoutfee"`
}

// MgoLatestScanInfo latest scan info
type MgoLatestScanInfo struct {
	Key         string `bson:"_id"`
	BlockHeight uint64 `bson:"blockheight"`
	Timestamp   int64  `bson:"timestamp"`
}

// MgoBlackAccount key is address
type MgoBlackAccount struct {
	Key       string `bson:"_id"` // address + pairid
	Address   string `bson:"address"`
	PairID    string `bson:"pairid"`
	Timestamp int64  `bson:"timestamp"`
}

// MgoSwapAgreement is hex encoded swapagreement
type MgoSwapAgreement struct {
	Key       string `bson:"_id"`
	Type      string `bson:"type"`
	Value     string `bson:"value"`
	Cancelled bool   `bson:"cancelled"`
}

// MgoSolanaScannedTx is MgoSolanaScannedTx
type MgoSolanaScannedTx struct {
	Address string `bson:"_id"`
	Txid    string `bson:"txid"`
}

// MgoLatestSwapNonce latest swap nonce
type MgoLatestSwapNonce struct {
	Key       string `bson:"_id"` // address + swaptype
	Address   string `bson:"address"`
	IsSwapin  bool   `bson:"isswapin"`
	SwapNonce uint64 `bson:"swapnonce"`
	Timestamp int64  `bson:"timestamp"`
}
