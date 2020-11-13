package tokens

import (
	"errors"
	"math/big"
)

// common errors
var (
	ErrSwapTypeNotSupported          = errors.New("swap type not supported in this endpoint")
	ErrBridgeSourceNotSupported      = errors.New("bridge source not supported")
	ErrBridgeDestinationNotSupported = errors.New("bridge destination not supported")
	ErrUnknownSwapType               = errors.New("unknown swap type")
	ErrMsgHashMismatch               = errors.New("message hash mismatch")
	ErrWrongCountOfMsgHashes         = errors.New("wrong count of msg hashed")
	ErrWrongRawTx                    = errors.New("wrong raw tx")
	ErrWrongExtraArgs                = errors.New("wrong extra args")
	ErrNoBtcBridge                   = errors.New("no btc bridge exist")
	ErrWrongSwapinTxType             = errors.New("wrong swapin tx type")
	ErrBuildSwapTxInWrongEndpoint    = errors.New("build swap in/out tx in wrong endpoint")
	ErrTxBeforeInitialHeight         = errors.New("transaction before initial block height")
	ErrAddressIsInBlacklist          = errors.New("address is in black list")

	ErrTodo = errors.New("developing: TODO")

	ErrTxNotFound           = errors.New("tx not found")
	ErrTxNotStable          = errors.New("tx not stable")
	ErrTxWithWrongReceiver  = errors.New("tx with wrong receiver")
	ErrTxWithWrongContract  = errors.New("tx with wrong contract")
	ErrTxWithWrongInput     = errors.New("tx with wrong input data")
	ErrTxWithWrongLogData   = errors.New("tx with wrong log data")
	ErrTxIsAggregateTx      = errors.New("tx is aggregate tx")
	ErrWrongP2shBindAddress = errors.New("wrong p2sh bind address")
	ErrNoBip32BindAddress   = errors.New("no bip32 bind address")
	ErrTxFuncHashMismatch   = errors.New("tx func hash mismatch")
	ErrDepositLogNotFound   = errors.New("deposit log not found or removed")
	ErrSwapoutLogNotFound   = errors.New("swapout log not found or removed")
	ErrUnknownPairID        = errors.New("unknown pair ID")
	ErrBindAddressMismatch  = errors.New("bind address mismatch")

	// errors should register
	ErrTxWithWrongMemo       = errors.New("tx with wrong memo")
	ErrTxWithWrongValue      = errors.New("tx with wrong value")
	ErrTxWithWrongReceipt    = errors.New("tx with wrong receipt")
	ErrTxWithWrongSender     = errors.New("tx with wrong sender")
	ErrTxSenderNotRegistered = errors.New("tx sender not registered")
	ErrTxIncompatible        = errors.New("tx incompatible")
	ErrBindAddrIsContract    = errors.New("bind address is contract")
	ErrRPCQueryError         = errors.New("rpc query error")
)

// ShouldRegisterSwapForError return true if this error should record in database
func ShouldRegisterSwapForError(err error) bool {
	switch err {
	case nil,
		ErrTxWithWrongMemo,
		ErrTxWithWrongValue,
		ErrTxWithWrongReceipt,
		ErrTxWithWrongSender,
		ErrTxSenderNotRegistered,
		ErrTxIncompatible,
		ErrBindAddrIsContract,
		ErrRPCQueryError:
		return true
	}
	return false
}

// CrossChainBridge interface
type CrossChainBridge interface {
	// is in the source (not destination) endpoint of the bridge
	IsSrcEndpoint() bool

	// chain, gateway and token config
	SetChainAndGateway(*ChainConfig, *GatewayConfig)
	GetChainConfig() *ChainConfig
	GetGatewayConfig() *GatewayConfig
	GetTokenConfig(pairID string) *TokenConfig
	VerifyTokenConfig(*TokenConfig) error

	// address validating
	IsValidAddress(address string) bool

	// Bip32 suppport
	GetBip32InputCode(address string) (string, error)
	PublicKeyToAddress(hexPubkey string) (string, error)

	// query and verify transaction
	GetTransaction(txHash string) (interface{}, error)
	GetTransactionStatus(txHash string) *TxStatus
	VerifyTransaction(pairID, txHash string, allowUnstable bool) (*TxSwapInfo, error)
	VerifyMsgHash(rawTx interface{}, msgHash []string) error

	// build, sign and send transaction
	BuildRawTransaction(args *BuildTxArgs) (rawTx interface{}, err error)
	SignTransaction(rawTx interface{}, pairID string) (signedTx interface{}, txHash string, err error)
	DcrmSignTransaction(rawTx interface{}, args *BuildTxArgs) (signedTx interface{}, txHash string, err error)
	SendTransaction(signedTx interface{}) (txHash string, err error)

	// query latest block number
	GetLatestBlockNumber() (uint64, error)
	GetLatestBlockNumberOf(apiAddress string) (uint64, error)

	// scan transaction job
	StartChainTransactionScanJob()
	StartPoolTransactionScanJob()

	// query coin or token balance
	GetBalance(accountAddress string) (*big.Int, error)
	GetTokenBalance(tokenType, tokenAddress, accountAddress string) (*big.Int, error)
	GetTokenSupply(tokenType, tokenAddress string) (*big.Int, error)

	// aggregate job
	StartAggregateJob()
	VerifyAggregateMsgHash(msgHash []string, args *BuildTxArgs) error
}

// NonceSetter interface (for eth-like)
type NonceSetter interface {
	GetPoolNonce(address, height string) (uint64, error)
	SetNonce(pairID string, value uint64)
	AdjustNonce(pairID string, value uint64) (nonce uint64)
	IncreaseNonce(pairID string, value uint64)
}
