package tokens

import (
	"errors"
	"fmt"
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

	maxPlusGasPricePercentage uint64 = 10000
)

// ChainConfig struct
type ChainConfig struct {
	BlockChain    string
	NetID         string
	Confirmations *uint64
	InitialHeight *uint64
	EnableScan    bool
}

// GatewayConfig struct
type GatewayConfig struct {
	APIAddress []string
}

// TokenConfig struct
type TokenConfig struct {
	ID                     string `json:",omitempty"`
	Name                   string
	Symbol                 string
	Decimals               *uint8
	Description            string `json:",omitempty"`
	DepositAddress         string `json:",omitempty"`
	DcrmAddress            string
	ContractAddress        string   `json:",omitempty"`
	MaximumSwap            *float64 // whole unit (eg. BTC, ETH, FSN), not Satoshi
	MinimumSwap            *float64 // whole unit
	BigValueThreshold      *float64
	SwapFeeRate            *float64
	MaximumSwapFee         *float64
	MinimumSwapFee         *float64
	PlusGasPricePercentage uint64 `json:",omitempty"`
	DisableSwap            bool

	// calced value
	maxSwap          *big.Int
	minSwap          *big.Int
	maxSwapFee       *big.Int
	minSwapFee       *big.Int
	bigValThreshhold *big.Int
}

// IsErc20 return is token is erc20
func (c *TokenConfig) IsErc20() bool {
	return strings.EqualFold(c.ID, "ERC20")
}

// SwapType type
type SwapType uint32

// SwapType constants
const (
	NoSwapType SwapType = iota
	SwapinType
	SwapoutType
)

func (s SwapType) String() string {
	switch s {
	case NoSwapType:
		return "noswap"
	case SwapinType:
		return "swapin"
	case SwapoutType:
		return "swapout"
	default:
		return fmt.Sprintf("unknown swap type %d", s)
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
		return fmt.Sprintf("unknown swaptx type %d", s)
	}
}

// TxSwapInfo struct
type TxSwapInfo struct {
	PairID    string   `json:"pairid"`
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
	PairID     string     `json:"pairid,omitempty"`
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

// CheckConfig check chain config
func (c *ChainConfig) CheckConfig() error {
	if c.BlockChain == "" {
		return errors.New("token must config 'BlockChain'")
	}
	if c.NetID == "" {
		return errors.New("token must config 'NetID'")
	}
	if c.Confirmations == nil {
		return errors.New("token must config 'Confirmations'")
	}
	if c.InitialHeight == nil {
		return errors.New("token must config 'InitialHeight'")
	}
	return nil
}

// CheckConfig check token config
//nolint:gocyclo // keep TokenConfig check as whole
func (c *TokenConfig) CheckConfig(isSrc bool) error {
	if c.Decimals == nil {
		return errors.New("token must config 'Decimals'")
	}
	if c.MaximumSwap == nil || *c.MaximumSwap < 0 {
		return errors.New("token must config 'MaximumSwap' (non-negative)")
	}
	if c.MinimumSwap == nil || *c.MinimumSwap < 0 {
		return errors.New("token must config 'MinimumSwap' (non-negative)")
	}
	if *c.MinimumSwap > *c.MaximumSwap {
		return errors.New("wrong token config, MinimumSwap > MaximumSwap")
	}
	if c.SwapFeeRate == nil || *c.SwapFeeRate < 0 || *c.SwapFeeRate > 1 {
		return errors.New("token must config 'SwapFeeRate' (in range (0,1))")
	}
	if c.MaximumSwapFee == nil || *c.MaximumSwapFee < 0 {
		return errors.New("token must config 'MaximumSwapFee' (non-negative)")
	}
	if c.MinimumSwapFee == nil || *c.MinimumSwapFee < 0 {
		return errors.New("token must config 'MinimumSwapFee' (non-negative)")
	}
	if *c.MinimumSwapFee > *c.MaximumSwapFee {
		return errors.New("wrong token config, MinimumSwapFee > MaximumSwapFee")
	}
	if *c.MinimumSwap < *c.MinimumSwapFee {
		return errors.New("wrong token config, MinimumSwap < MinimumSwapFee")
	}
	if *c.SwapFeeRate == 0.0 && *c.MinimumSwapFee > 0.0 {
		return errors.New("wrong token config, MinimumSwapFee should be 0 if SwapFeeRate is 0")
	}
	if c.PlusGasPricePercentage > maxPlusGasPricePercentage {
		return errors.New("too large 'PlusGasPricePercentage' value")
	}
	if c.BigValueThreshold == nil {
		return errors.New("token must config 'BigValueThreshold'")
	}
	if c.DcrmAddress == "" {
		return errors.New("token must config 'DcrmAddress'")
	}
	if isSrc && c.DepositAddress == "" {
		return errors.New("token must config 'DepositAddress' for source chain")
	}
	if !isSrc && c.ContractAddress == "" {
		return errors.New("token must config 'ContractAddress' for destination chain")
	}
	if isSrc && c.IsErc20() && c.ContractAddress == "" {
		return errors.New("token must config 'ContractAddress' for ERC20 in source chain")
	}
	// calc value and store
	c.CalcAndStoreValue()
	return nil
}

// CalcAndStoreValue calc and store value (minus duplicate calculation)
func (c *TokenConfig) CalcAndStoreValue() {
	c.maxSwap = ToBits(*c.MaximumSwap, *c.Decimals)
	c.minSwap = ToBits(*c.MinimumSwap, *c.Decimals)
	c.maxSwapFee = ToBits(*c.MaximumSwapFee, *c.Decimals)
	c.minSwapFee = ToBits(*c.MinimumSwapFee, *c.Decimals)
	c.bigValThreshhold = ToBits(*c.BigValueThreshold, *c.Decimals)
}
