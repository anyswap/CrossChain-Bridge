package mongodb

const (
	tbSwapins        string = "Swapins"
	tbSwapouts       string = "Swapouts"
	tbSwapinResults  string = "SwapinResults"
	tbSwapoutResults string = "SwapoutResults"
)

type MgoSwap struct {
	Key       string `bson:"_id"`
	TxId      string `bson:"txid"`
	Status    uint16 `bson:"status"`
	Timestamp uint64 `bson:"timestamp"`
	Memo      string `bson:"memo"`
}

type MgoSwapResult struct {
	Key        string `bson:"_id"`
	TxId       string `bson:"txid"`
	TxHeight   uint64 `bson:"txheight"`
	TxTime     uint64 `bson:"txtime"`
	From       string `bson:"from"`
	To         string `bson:"to"`
	Bind       string `bson:"bind"`
	Value      string `bson:"value"`
	SwapTx     string `bson:"swaptx"`
	SwapHeight uint64 `bson:"swapheight"`
	SwapTime   uint64 `bson:"swaptime"`
	Status     uint16 `bson:"status"`
	Timestamp  uint64 `bson:"timestamp"`
	Memo       string `bson:"memo"`
}

type SwapResultUpdateItems struct {
	SwapTx     string
	SwapHeight uint64
	SwapTime   uint64
	Status     uint16
	Timestamp  uint64
	Memo       string
}
