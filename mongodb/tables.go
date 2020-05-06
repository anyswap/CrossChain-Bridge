package mongodb

const (
	tbSwapins        string = "Swapins"
	tbSwapouts       string = "Swapouts"
	tbSwapinResults  string = "SwapinResults"
	tbSwapoutResults string = "SwapoutResults"
)

type MgoSwap struct {
	Key       string     `bson:"_id"`
	TxId      string     `bson:"txid"`
	Status    SwapStatus `bson:"status"`
	Timestamp int64      `bson:"timestamp"`
	Memo      string     `bson:"memo"`
}

type MgoSwapResult struct {
	Key        string     `bson:"_id"`
	TxId       string     `bson:"txid"`
	TxHeight   uint64     `bson:"txheight"`
	TxTime     uint64     `bson:"txtime"`
	From       string     `bson:"from"`
	To         string     `bson:"to"`
	Bind       string     `bson:"bind"`
	Value      string     `bson:"value"`
	SwapTx     string     `bson:"swaptx"`
	SwapHeight uint64     `bson:"swapheight"`
	SwapTime   uint64     `bson:"swaptime"`
	SwapValue  string     `bson:"swapvalue"`
	Status     SwapStatus `bson:"status"`
	Timestamp  int64      `bson:"timestamp"`
	Memo       string     `bson:"memo"`
}

type SwapResultUpdateItems struct {
	SwapTx     string
	SwapHeight uint64
	SwapTime   uint64
	SwapValue  string
	Status     SwapStatus
	Timestamp  int64
	Memo       string
}
