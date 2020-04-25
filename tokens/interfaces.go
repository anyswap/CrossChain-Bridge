package tokens

import (
	"errors"
	"math/big"
)

const (
	LockMemoPrefix   = "SWAPTO:"
	UnlockMemoPrefix = "SWAPTX:"
	RecallMemoPrefix = "RECALL:"
)

var (
	SrcBridge CrossChainBridge
	DstBridge CrossChainBridge

	// first 4 bytes of `Keccak256Hash([]byte("Swapin(bytes32,address,uint256)"))`
	SwapinFuncHash = [4]byte{0xec, 0x12, 0x6c, 0x77}
)

var (
	ErrBridgeSourceNotSupported      = errors.New("bridge source not supported")
	ErrBridgeDestinationNotSupported = errors.New("bridge destination not supported")

	ErrTodo = errors.New("developing: TODO")

	ErrTxNotStable         = errors.New("tx not stable")
	ErrTxWithWrongValue    = errors.New("tx with wrong value")
	ErrTxWithWrongReceiver = errors.New("tx with wrong receiver")
	ErrTxWithWrongMemo     = errors.New("tx with wrong memo")
	ErrTxWithWrongStatus   = errors.New("tx with wrong status")
	ErrTxWithWrongReceipt  = errors.New("tx with wrong receipt")
)

type TokenConfig struct {
	BlockChain      *string
	NetID           *string
	ID              string
	Name            string
	Symbol          string
	Decimals        *uint8
	Description     string
	DcrmAddress     *string
	ContractAddress *string
	Confirmations   *uint64
	MaximumSwap     *float64 // whole unit (eg. BTC, ETH, FSN), not Satoshi
	MinimumSwap     *float64 // whole unit
	SwapFeeRate     *float64
}

func (c *TokenConfig) CheckConfig(isSrc bool) error {
	if c.BlockChain == nil {
		return errors.New("token must config 'BlockChain'")
	}
	if c.NetID == nil {
		return errors.New("token must config 'NetID'")
	}
	if c.Decimals == nil {
		return errors.New("token must config 'Decimals'")
	}
	if c.Confirmations == nil {
		return errors.New("token must config 'Confirmations'")
	}
	if c.MaximumSwap == nil {
		return errors.New("token must config 'MaximumSwap'")
	}
	if c.MinimumSwap == nil {
		return errors.New("token must config 'MinimumSwap'")
	}
	if c.SwapFeeRate == nil {
		return errors.New("token must config 'SwapFeeRate'")
	}
	if c.DcrmAddress == nil {
		return errors.New("token must config 'DcrmAddress'")
	}
	if !isSrc && c.ContractAddress == nil {
		return errors.New("token must config 'ContractAddress' for destination chain")
	}
	return nil
}

type GatewayConfig struct {
	ApiAddress string
}

type TxSwapInfo struct {
	Hash      string `json:"hash"`
	Height    uint64 `json:"height"`
	Timestamp uint64 `json:"timestamp"`
	From      string `json:"from"`
	To        string `json:"to"`
	Bind      string `json:"bind"`
	Value     string `json:"value"`
}

type BuildTxArgs struct {
	IsSwapin      bool     `json:"isSwapin,omitempty"`
	From          string   `json:"from"`
	To            string   `json:"to"`
	Value         *big.Int `json:"value"`
	Memo          string   `json:"memo,omitempty"`
	Gas           *uint64  `json:"gas,omitempty"`           // eth
	GasPrice      *big.Int `json:"gasPrice,omitempty"`      // eth
	Nonce         *uint64  `json:"nonce,omitempty"`         // eth
	Input         *[]byte  `json:"input,omitempty"`         // eth erc20 ...
	FeeRate       *int64   `json:"feeRate,omitempty"`       // btc
	ChangeAddress *string  `json:"changeAddress,omitempty"` // btc
	FromPublicKey *string  `json:"fromPublickey,omitempty"` // btc
}

type CrossChainBridge interface {
	GetTokenAndGateway() (*TokenConfig, *GatewayConfig)
	SetTokenAndGateway(*TokenConfig, *GatewayConfig)

	IsValidAddress(address string) bool

	IsTransactionStable(txHash string) bool
	VerifyTransaction(txHash string) (*TxSwapInfo, error)

	BuildRawTransaction(args *BuildTxArgs) (rawTx interface{}, err error)
	DcrmSignTransaction(rawTx interface{}) (signedTx interface{}, err error)
	SendTransaction(signedTx interface{}) (txHash string, err error)
}

type CrossChainBridgeBase struct {
	tokenConfig   *TokenConfig
	gatewayConfig *GatewayConfig
}

func (b *CrossChainBridgeBase) GetTokenAndGateway() (*TokenConfig, *GatewayConfig) {
	return b.tokenConfig, b.gatewayConfig
}

func (b *CrossChainBridgeBase) SetTokenAndGateway(tokenCfg *TokenConfig, gatewayCfg *GatewayConfig) {
	b.tokenConfig = tokenCfg
	b.gatewayConfig = gatewayCfg
	err := tokenCfg.CheckConfig(true)
	if err != nil {
		panic(err)
	}
}
