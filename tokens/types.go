package tokens

import (
	"fmt"
	"math/big"
)

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
	TxTo      string   `json:"txto"`
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
	Reswapping bool       `json:"reswapping,omitempty"`
}

// BuildTxArgs struct
type BuildTxArgs struct {
	SwapInfo    `json:"swapInfo,omitempty"`
	From        string     `json:"from,omitempty"`
	To          string     `json:"to,omitempty"`
	Value       *big.Int   `json:"value,omitempty"`
	OriginValue *big.Int   `json:"originValue,omitempty"`
	SwapValue   *big.Int   `json:"swapvalue,omitempty"`
	Memo        string     `json:"memo,omitempty"`
	Input       *[]byte    `json:"input,omitempty"`
	Extra       *AllExtras `json:"extra,omitempty"`
}

// GetReplaceNum get rplace swap count
func (args *BuildTxArgs) GetReplaceNum() uint64 {
	if args.Extra != nil {
		return args.Extra.ReplaceNum
	}
	return 0
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

// SetTxNonce set tx nonce
func (args *BuildTxArgs) SetTxNonce(nonce uint64) {
	var extra *EthExtraArgs
	if args.Extra == nil || args.Extra.EthExtra == nil {
		extra = &EthExtraArgs{}
		args.Extra = &AllExtras{EthExtra: extra}
	} else {
		extra = args.Extra.EthExtra
	}
	extra.Nonce = &nonce
}

// GetTxGasPrice get tx gas price
func (args *BuildTxArgs) GetTxGasPrice() *big.Int {
	if args.Extra != nil && args.Extra.EthExtra != nil && args.Extra.EthExtra.GasPrice != nil {
		return args.Extra.EthExtra.GasPrice
	}
	return nil
}

// AllExtras struct
type AllExtras struct {
	BtcExtra   *BtcExtraArgs `json:"btcExtra,omitempty"`
	EthExtra   *EthExtraArgs `json:"ethExtra,omitempty"`
	ReplaceNum uint64        `json:"replaceNum,omitempty"`
}

// EthExtraArgs struct
type EthExtraArgs struct {
	Gas       *uint64  `json:"gas,omitempty"`
	GasPrice  *big.Int `json:"gasPrice,omitempty"`
	GasTipCap *big.Int `json:"gasTipCap,omitempty"`
	GasFeeCap *big.Int `json:"gasFeeCap,omitempty"`
	Nonce     *uint64  `json:"nonce,omitempty"`
}

// BtcOutPoint struct
type BtcOutPoint struct {
	Hash  string `json:"hash"`
	Index uint32 `json:"index"`
}

// BtcExtraArgs struct
type BtcExtraArgs struct {
	RelayFeePerKb     *int64         `json:"relayFeePerKb,omitempty"`
	ChangeAddress     *string        `json:"-"`
	PreviousOutPoints []*BtcOutPoint `json:"previousOutPoints,omitempty"`
}

// P2shAddressInfo struct
type P2shAddressInfo struct {
	BindAddress        string
	P2shAddress        string
	RedeemScript       string
	RedeemScriptDisasm string
}
