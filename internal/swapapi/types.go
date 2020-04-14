package swapapi

type ServerInfo struct {
	SrcAsset  string
	DestAsset string
}

type SwapStatistics struct {
	TotalValue string
}

type PostResult struct {
	Status  string
	Message string
}

type SwapInfo struct {
	TxId       string
	TxHeight   uint64
	SwapTx     string
	SwapHeight uint64
	Memo       string
}
