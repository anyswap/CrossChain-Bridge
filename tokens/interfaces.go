// Package tokens defines the common interfaces and supported bridges in sub directories.
package tokens

// CrossChainBridge interface
type CrossChainBridge interface {
	IsSrcEndpoint() bool

	SetChainAndGateway(*ChainConfig, *GatewayConfig)

	GetChainConfig() *ChainConfig
	GetGatewayConfig() *GatewayConfig
	GetTokenConfig(pairID string) *TokenConfig

	VerifyTokenConfig(*TokenConfig) error
	IsValidAddress(address string) bool

	InitAfterConfig()

	GetTransaction(txHash string) (interface{}, error)
	GetTransactionStatus(txHash string) (*TxStatus, error)
	VerifyTransaction(pairID, txHash string, allowUnstable bool) (*TxSwapInfo, error)
	VerifyMsgHash(rawTx interface{}, msgHash []string) error

	BuildRawTransaction(args *BuildTxArgs) (rawTx interface{}, err error)
	SignTransaction(rawTx interface{}, pairID string) (signedTx interface{}, txHash string, err error)
	DcrmSignTransaction(rawTx interface{}, args *BuildTxArgs) (signedTx interface{}, txHash string, err error)
	SendTransaction(signedTx interface{}) (txHash string, err error)

	GetLatestBlockNumber() (uint64, error)
	GetLatestBlockNumberOf(apiAddress string) (uint64, error)
}

// NonceSetter interface (for eth-like)
type NonceSetter interface {
	GetTxBlockInfo(txHash string) (blockHeight, blockTime uint64)
	GetPoolNonce(address, height string) (uint64, error)
	SetNonce(pairID string, value uint64)
	AdjustNonce(pairID string, value uint64) (nonce uint64)
	InitNonces(nonces map[string]uint64)
}

// ForkChecker fork checker interface
type ForkChecker interface {
	GetBlockHashOf(urls []string, height uint64) (hash string, err error)
}
