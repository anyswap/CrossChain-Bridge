package tokens

import (
	"errors"
	"math/big"
	"strings"
)

// btc extra default values
var (
	BtcMinRelayFee   int64 = 400
	BtcRelayFeePerKb int64 = 2000
	BtcFromPublicKey string

	BtcUtxoAggregateMinCount  = 20
	BtcUtxoAggregateMinValue  = uint64(1000000)
	BtcUtxoAggregateToAddress = ""
)

// TokenConfig struct
type TokenConfig struct {
	BlockChain      string
	NetID           string
	ID              string `json:",omitempty"`
	Name            string
	Symbol          string
	Decimals        *uint8
	Description     string `json:",omitempty"`
	DcrmAddress     string
	ContractAddress string `json:",omitempty"`
	Confirmations   *uint64
	MaximumSwap     *float64 // whole unit (eg. BTC, ETH, FSN), not Satoshi
	MinimumSwap     *float64 // whole unit
	SwapFeeRate     *float64
	InitialHeight   uint64
	MinTimeToRetry  int64 // unit second
}

// IsErc20 return is token is erc20
func (c *TokenConfig) IsErc20() bool {
	return strings.EqualFold(c.ID, "ERC20")
}

// GatewayConfig struct
type GatewayConfig struct {
	APIAddress []string
}

// SwapType type
type SwapType uint32

// SwapType constants
const (
	NoSwapType SwapType = iota
	SwapinType
	SwapoutType
	SwapRecallType
)

func (s SwapType) String() string {
	switch s {
	case NoSwapType:
		return "noswap"
	case SwapinType:
		return "swapin"
	case SwapoutType:
		return "swapout"
	case SwapRecallType:
		return "recall"
	default:
		return "unknown swap type"
	}
}

// SwapTxType type
type SwapTxType uint32

// SwapTxType constants
const (
	SwapinTx     SwapTxType = iota // 0
	SwapoutTx                      // 1
	P2shSwapinTx                   // 2
)

func (s SwapTxType) String() string {
	switch s {
	case SwapinTx:
		return "swapintx"
	case SwapoutTx:
		return "swapouttx"
	case P2shSwapinTx:
		return "p2shswapintx"
	default:
		return "unknown swaptx type"
	}
}

// TxSwapInfo struct
type TxSwapInfo struct {
	Hash      string   `json:"hash"`
	Height    uint64   `json:"height"`
	Timestamp uint64   `json:"timestamp"`
	From      string   `json:"from"`
	To        string   `json:"to"`
	Bind      string   `json:"bind"`
	Value     *big.Int `json:"value"`
}

// TxStatus struct
type TxStatus struct {
	Receipt       interface{} `json:"receipt,omitempty"`
	Confirmations uint64      `json:"confirmations"`
	BlockHeight   uint64      `json:"block_height"`
	BlockHash     string      `json:"block_hash"`
	BlockTime     uint64      `json:"block_time"`
}

// SwapInfo struct
type SwapInfo struct {
	SwapID     string     `json:"swapid,omitempty"`
	SwapType   SwapType   `json:"swaptype,omitempty"`
	TxType     SwapTxType `json:"txtype,omitempty"`
	Bind       string     `json:"bind,omitempty"`
	Identifier string     `json:"identifier,omitempty"`
}

// BuildTxArgs struct
type BuildTxArgs struct {
	SwapInfo `json:"swapInfo,omitempty"`
	From     string     `json:"from,omitempty"`
	To       string     `json:"to,omitempty"`
	Value    *big.Int   `json:"value,omitempty"`
	Memo     string     `json:"memo,omitempty"`
	Input    *[]byte    `json:"input,omitempty"`
	Extra    *AllExtras `json:"extra,omitempty"`
}

// GetExtraArgs get extra args
func (args *BuildTxArgs) GetExtraArgs() *BuildTxArgs {
	return &BuildTxArgs{
		SwapInfo: args.SwapInfo,
		Extra:    args.Extra,
	}
}

// GetTxNonce get tx nonce
func (args *BuildTxArgs) GetTxNonce() uint64 {
	if args.Extra != nil && args.Extra.EthExtra != nil && args.Extra.EthExtra.Nonce != nil {
		return *args.Extra.EthExtra.Nonce
	}
	return 0
}

// AllExtras struct
type AllExtras struct {
	BtcExtra *BtcExtraArgs `json:"btcExtra,omitempty"`
	EthExtra *EthExtraArgs `json:"ethExtra,omitempty"`
}

// EthExtraArgs struct
type EthExtraArgs struct {
	Gas      *uint64  `json:"gas,omitempty"`
	GasPrice *big.Int `json:"gasPrice,omitempty"`
	Nonce    *uint64  `json:"nonce,omitempty"`
}

// BtcOutPoint struct
type BtcOutPoint struct {
	Hash  string `json:"hash"`
	Index uint32 `json:"index"`
}

// BtcExtraArgs struct
type BtcExtraArgs struct {
	RelayFeePerKb *int64  `json:"relayFeePerKb,omitempty"`
	ChangeAddress *string `json:"changeAddress,omitempty"`
	FromPublicKey *string `json:"fromPublickey,omitempty"`

	PreviousOutPoints []*BtcOutPoint `json:"previousOutPoints,omitempty"`
}

// BtcExtraConfig used to build swpout to btc tx
type BtcExtraConfig struct {
	MinRelayFee            int64
	RelayFeePerKb          int64
	FromPublicKey          string
	UtxoAggregateMinCount  int
	UtxoAggregateMinValue  uint64
	UtxoAggregateToAddress string
}

// P2shAddressInfo struct
type P2shAddressInfo struct {
	BindAddress        string
	P2shAddress        string
	RedeemScript       string
	RedeemScriptDisasm string
}

// CheckConfig check config
//nolint:gocyclo // keep TokenConfig check as whole
func (c *TokenConfig) CheckConfig(isSrc bool) error {
	if c.BlockChain == "" {
		return errors.New("token must config 'BlockChain'")
	}
	if c.NetID == "" {
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
	if *c.MaximumSwap < 0 {
		return errors.New("token 'MaximumSwap' is negative")
	}
	if c.MinimumSwap == nil {
		return errors.New("token must config 'MinimumSwap'")
	}
	if *c.MinimumSwap < 0 {
		return errors.New("token 'MinimumSwap' is negative")
	}
	if c.SwapFeeRate == nil {
		return errors.New("token must config 'SwapFeeRate'")
	}
	if *c.SwapFeeRate < 0 {
		return errors.New("token 'SwapFeeRate' is negative")
	}
	if c.DcrmAddress == "" {
		return errors.New("token must config 'DcrmAddress'")
	}
	if !isSrc && c.ContractAddress == "" {
		return errors.New("token must config 'ContractAddress' for destination chain")
	}
	if isSrc && c.IsErc20() && c.ContractAddress == "" {
		return errors.New("token must config 'ContractAddress' for ERC20 in source chain")
	}
	return nil
}
