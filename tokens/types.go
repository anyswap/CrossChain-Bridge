package tokens

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tools"
	"github.com/anyswap/CrossChain-Bridge/tools/crypto"
)

// BtcExtraConfig used to build swpout to btc tx
type BtcExtraConfig struct {
	MinRelayFee       int64
	MinRelayFeePerKb  int64
	MaxRelayFeePerKb  int64
	PlusFeePercentage uint64
	EstimateFeeBlocks int

	UtxoAggregateMinCount  int
	UtxoAggregateMinValue  uint64
	UtxoAggregateToAddress string
}

// ChainConfig struct
type ChainConfig struct {
	BlockChain    string
	NetID         string
	Confirmations *uint64
	InitialHeight *uint64

	// judge by the 'from' chain (eg. src for swapin)
	EnableScan              bool
	EnableScanPool          bool
	EnablePassBigValue      bool
	EnableCheckTxBlockHash  bool
	EnableCheckTxBlockIndex bool

	// judge by the 'to' chain (eg. dst for swapin)
	EnableReplaceSwap bool

	AllowCallByContract             bool
	CallByContractWhitelist         []string `json:",omitempty"`
	CallByContractCodeHashWhitelist []string `json:",omitempty"`

	MinReserveFee              string
	BaseFeePercent             int64
	MaxGasPriceFluctPercent    uint64 `json:",omitempty"`
	ReplacePlusGasPricePercent uint64 `json:",omitempty"`
	WaitTimeToReplace          int64  // seconds
	MaxReplaceCount            int
	FixedGasPrice              string `json:",omitempty"`
	MaxGasPrice                string `json:",omitempty"`
	MinGasPrice                string `json:",omitempty"`

	// calced value
	fixedGasPrice *big.Int
	maxGasPrice   *big.Int
	minGasPrice   *big.Int
	minReserveFee *big.Int

	callByContractWhitelist         map[string]struct{}
	callByContractCodeHashWhitelist map[string]struct{}
}

// IsInCallByContractWhitelist is in call by contract whitelist
func (c *ChainConfig) IsInCallByContractWhitelist(caller string) bool {
	if c.callByContractWhitelist == nil {
		return false
	}
	_, exist := c.callByContractWhitelist[strings.ToLower(caller)]
	return exist
}

// HasCallByContractCodeHashWhitelist has call by contract code hash whitelist
func (c *ChainConfig) HasCallByContractCodeHashWhitelist() bool {
	return len(c.callByContractCodeHashWhitelist) > 0
}

// IsInCallByContractCodeHashWhitelist is in call by contract code hash whitelist
func (c *ChainConfig) IsInCallByContractCodeHashWhitelist(codehash string) bool {
	if c.callByContractCodeHashWhitelist == nil {
		return false
	}
	_, exist := c.callByContractCodeHashWhitelist[codehash]
	return exist
}

// GatewayConfig struct
type GatewayConfig struct {
	APIAddress    []string
	APIAddressExt []string
	Extras        *GatewayExtras
}

// GatewayExtras struct
type GatewayExtras struct {
	BlockExtra *BlockExtraArgs
}

// BlockExtraArgs struct
type BlockExtraArgs struct {
	CoreAPIs         []BlocknetCoreAPIArgs
	UTXOAPIAddresses []string
}

// BlocknetCoreAPIArgs struct
type BlocknetCoreAPIArgs struct {
	APIAddress  string
	RPCUser     string
	RPCPassword string
	DisableTLS  bool
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
	DcrmPubkey             string   `json:"-"`
	ContractAddress        string   `json:",omitempty"`
	ContractCodeHash       string   `json:",omitempty"`
	MaximumSwap            *float64 // whole unit (eg. BTC, ETH, FSN), not Satoshi
	MinimumSwap            *float64 // whole unit
	BigValueThreshold      *float64
	SwapFeeRate            *float64
	MaximumSwapFee         *float64
	MinimumSwapFee         *float64
	PlusGasPricePercentage uint64 `json:",omitempty"`
	DisableSwap            bool
	IsDelegateContract     bool
	DelegateToken          string `json:",omitempty"`
	IsAnyswapAdapter       bool   `json:",omitempty"`

	DefaultGasLimit          uint64 `json:",omitempty"`
	AllowSwapinFromContract  bool   `json:",omitempty"`
	AllowSwapoutFromContract bool   `json:",omitempty"`

	BigValueWhitelist []string `json:",omitempty"`

	// use private key address instead
	DcrmAddressKeyStore string `json:"-"`
	DcrmAddressPassword string `json:"-"`
	DcrmAddressKeyFile  string `json:"-"`
	dcrmAddressPriKey   *ecdsa.PrivateKey

	// calced value
	maxSwap          *big.Int
	minSwap          *big.Int
	maxSwapFee       *big.Int
	minSwapFee       *big.Int
	bigValThreshhold *big.Int

	bigValueWhitelist map[string]struct{}
}

// IsInBigValueWhitelist is in big value whitelist
func (c *TokenConfig) IsInBigValueWhitelist(caller string) bool {
	if c.bigValueWhitelist == nil {
		return false
	}
	_, exist := c.bigValueWhitelist[strings.ToLower(caller)]
	return exist
}

// IsErc20 return if token is erc20
func (c *TokenConfig) IsErc20() bool {
	return strings.EqualFold(c.ID, "ERC20") || c.IsProxyErc20()
}

// IsProxyErc20 return if token is proxy contract of erc20
func (c *TokenConfig) IsProxyErc20() bool {
	return strings.EqualFold(c.ID, "ProxyERC20")
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
}

// IsSwapin is swapin type
func (s *SwapInfo) IsSwapin() bool {
	return s.SwapType == SwapinType
}

// BuildTxArgs struct
type BuildTxArgs struct {
	SwapInfo    `json:"swapInfo,omitempty"`
	From        string     `json:"from,omitempty"`
	To          string     `json:"to,omitempty"`
	OriginFrom  string     `json:"originFrom,omitempty"`
	OriginTxTo  string     `json:"originTxTo,omitempty"`
	Value       *big.Int   `json:"value,omitempty"`
	OriginValue *big.Int   `json:"originValue,omitempty"`
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

// AllExtras struct
type AllExtras struct {
	BtcExtra   *BtcExtraArgs `json:"btcExtra,omitempty"`
	EthExtra   *EthExtraArgs `json:"ethExtra,omitempty"`
	ReplaceNum uint64        `json:"replaceNum,omitempty"`
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
	if c.BaseFeePercent < -90 || c.BaseFeePercent > 500 {
		return errors.New("'BaseFeePercent' must be in range [-90, 500]")
	}
	if c.FixedGasPrice != "" {
		fixedGasPrice, err := common.GetBigIntFromStr(c.FixedGasPrice)
		if err != nil {
			return err
		}
		c.fixedGasPrice = fixedGasPrice
	}
	if c.MaxGasPrice != "" {
		maxGasPrice, err := common.GetBigIntFromStr(c.MaxGasPrice)
		if err != nil {
			return err
		}
		c.maxGasPrice = maxGasPrice
	}
	if c.MinGasPrice != "" {
		minGasPrice, err := common.GetBigIntFromStr(c.MinGasPrice)
		if err != nil {
			return err
		}
		c.minGasPrice = minGasPrice
	}
	if c.MinReserveFee != "" {
		bi, ok := new(big.Int).SetString(c.MinReserveFee, 10)
		if !ok {
			return fmt.Errorf("wrong 'MinReserveFee' value '%v'", c.MinReserveFee)
		}
		c.minReserveFee = bi
	}
	if len(c.CallByContractWhitelist) > 0 {
		c.callByContractWhitelist = make(map[string]struct{}, len(c.CallByContractWhitelist))
		for _, addr := range c.CallByContractWhitelist {
			if !common.IsHexAddress(addr) {
				return fmt.Errorf("wrong address '%v' in 'CallByContractWhitelist'", addr)
			}
			key := strings.ToLower(addr)
			if _, exist := c.callByContractWhitelist[key]; exist {
				return fmt.Errorf("duplicate address '%v' in 'CallByContractWhitelist'", addr)
			}
			c.callByContractWhitelist[key] = struct{}{}
		}
	}
	if len(c.CallByContractCodeHashWhitelist) > 0 {
		c.callByContractCodeHashWhitelist = make(map[string]struct{}, len(c.CallByContractCodeHashWhitelist))
		for _, codehash := range c.CallByContractCodeHashWhitelist {
			if !common.IsHexHash(codehash) {
				return fmt.Errorf("wrong codeHash '%v' in CallByContractCodeHashWhitelist", codehash)
			}
			if _, exist := c.callByContractCodeHashWhitelist[codehash]; exist {
				return fmt.Errorf("duplicate codeHash '%v' in 'CallByContractCodeHashWhitelist'", codehash)
			}
			c.callByContractCodeHashWhitelist[codehash] = struct{}{}
		}
	}
	if c.minGasPrice != nil {
		if c.fixedGasPrice != nil {
			return errors.New("FixedGasPrice and MinGasPrice are conflicted")
		}
		if c.maxGasPrice != nil && c.minGasPrice.Cmp(c.maxGasPrice) > 0 {
			return errors.New("MinGasPrice > MaxGasPrice")
		}
	}
	log.Info("check chain config success",
		"blockChain", c.BlockChain,
		"fixedGasPrice", c.FixedGasPrice,
		"maxGasPrice", c.MaxGasPrice,
		"minGasPrice", c.MinGasPrice,
		"baseFeePercent", c.BaseFeePercent,
	)
	return nil
}

// IsFixedGasPrice is fixed gas price
func (c *ChainConfig) IsFixedGasPrice() bool {
	return c.fixedGasPrice != nil
}

// GetFixedGasPrice get fixed gas price
func (c *ChainConfig) GetFixedGasPrice() *big.Int {
	if c.fixedGasPrice != nil {
		return new(big.Int).Set(c.fixedGasPrice) // clone
	}
	return nil
}

// GetMaxGasPrice get max gas price
func (c *ChainConfig) GetMaxGasPrice() *big.Int {
	if c.maxGasPrice != nil {
		return new(big.Int).Set(c.maxGasPrice) // clone
	}
	return nil
}

// GetMinGasPrice get min gas price
func (c *ChainConfig) GetMinGasPrice() *big.Int {
	if c.minGasPrice != nil {
		return new(big.Int).Set(c.minGasPrice) // clone
	}
	return nil
}

// GetMinReserveFee get min reserve fee
func (c *ChainConfig) GetMinReserveFee() *big.Int {
	return c.minReserveFee
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
	maxPlusGasPricePercentage := uint64(10000)
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
	if c.IsProxyErc20() {
		if !isSrc {
			return errors.New("token ProxyERC20 is only support in source chain")
		}
		if c.ContractAddress == "" {
			return errors.New("token ProxyERC20 must config 'ContractAddress'")
		}
		if c.ContractCodeHash == "" {
			return errors.New("token ProxyERC20 must config 'ContractCodeHash'")
		}
	} else if c.ContractCodeHash != "" {
		return errors.New("token forbid config 'ContractCodeHash' if it's not ProxyERC20")
	}
	if c.IsDelegateContract {
		if c.ContractAddress == "" {
			return errors.New("token must config 'ContractAddress' if 'IsDelegateContract' is true")
		}
		if c.DelegateToken != "" && !common.IsHexAddress(c.DelegateToken) {
			return errors.New("wrong 'DelegateToken' address")
		}
	}
	// calc value and store
	c.CalcAndStoreValue()
	err := c.LoadDcrmAddressPrivateKey()
	if err != nil {
		return err
	}
	err = c.VerifyDcrmPublicKey()
	if err != nil {
		return err
	}
	if len(c.BigValueWhitelist) > 0 {
		c.bigValueWhitelist = make(map[string]struct{}, len(c.BigValueWhitelist))
		for _, addr := range c.BigValueWhitelist {
			if !common.IsHexAddress(addr) {
				return fmt.Errorf("wrong address '%v' in 'BigValueWhitelist'", addr)
			}
			key := strings.ToLower(addr)
			if _, exist := c.bigValueWhitelist[key]; exist {
				return fmt.Errorf("duplicate address '%v' in 'BigValueWhitelist'", addr)
			}
			c.bigValueWhitelist[key] = struct{}{}
		}
	}
	log.Info("check token config success",
		"id", c.ID, "name", c.Name, "symbol", c.Symbol, "decimals", *c.Decimals,
		"depositAddress", c.DepositAddress, "contractAddress", c.ContractAddress,
		"maxSwap", c.maxSwap, "minSwap", c.minSwap,
		"maxSwapFee", c.maxSwapFee, "minSwapFee", c.minSwapFee,
		"bigValThreshhold", c.bigValThreshhold, "bigValueWhitelist", c.bigValueWhitelist,
	)
	return nil
}

// CalcAndStoreValue calc and store value (minus duplicate calculation)
func (c *TokenConfig) CalcAndStoreValue() {
	smallBiasValue := 0.0001
	c.maxSwap = ToBits(*c.MaximumSwap+smallBiasValue, *c.Decimals)
	c.minSwap = ToBits(*c.MinimumSwap-smallBiasValue, *c.Decimals)
	c.maxSwapFee = ToBits(*c.MaximumSwapFee, *c.Decimals)
	c.minSwapFee = ToBits(*c.MinimumSwapFee, *c.Decimals)
	c.bigValThreshhold = ToBits(*c.BigValueThreshold+smallBiasValue, *c.Decimals)
}

// GetDcrmAddressPrivateKey get private key
func (c *TokenConfig) GetDcrmAddressPrivateKey() *ecdsa.PrivateKey {
	return c.dcrmAddressPriKey
}

// LoadDcrmAddressPrivateKey load private key
func (c *TokenConfig) LoadDcrmAddressPrivateKey() error {
	if c.DcrmAddressKeyFile != "" {
		priKey, err := crypto.LoadECDSA(c.DcrmAddressKeyFile)
		if err != nil {
			return fmt.Errorf("wrong private key, %v", err)
		}
		c.dcrmAddressPriKey = priKey
	} else if c.DcrmAddressKeyStore != "" {
		key, err := tools.LoadKeyStore(c.DcrmAddressKeyStore, c.DcrmAddressPassword)
		if err != nil {
			return err
		}
		c.dcrmAddressPriKey = key.PrivateKey
	}
	if c.dcrmAddressPriKey != nil {
		keyAddr := crypto.PubkeyToAddress(c.dcrmAddressPriKey.PublicKey)
		if !strings.EqualFold(keyAddr.String(), c.DcrmAddress) {
			return fmt.Errorf("dcrm address %v and its keystore address %v is not match", c.DcrmAddress, keyAddr.String())
		}
	} else {
		if c.DcrmPubkey == "" {
			return fmt.Errorf("token must config 'DcrmPubkey'")
		}
		if IsDcrmDisabled {
			return fmt.Errorf("dcrm is disabled but no private key is provided")
		}
	}
	return nil
}

// VerifyDcrmPublicKey verify public key
func (c *TokenConfig) VerifyDcrmPublicKey() error {
	if !common.IsHexAddress(c.DcrmAddress) {
		return nil
	}
	if c.dcrmAddressPriKey != nil && c.DcrmPubkey == "" {
		return nil
	}
	// ETH like address
	pkBytes := common.FromHex(c.DcrmPubkey)
	if len(pkBytes) != 65 || pkBytes[0] != 4 {
		return fmt.Errorf("wrong dcrm public key, shoule be uncompressed")
	}
	pubKey := ecdsa.PublicKey{
		Curve: crypto.S256(),
		X:     new(big.Int).SetBytes(pkBytes[1:33]),
		Y:     new(big.Int).SetBytes(pkBytes[33:65]),
	}
	pubAddr := crypto.PubkeyToAddress(pubKey)
	if !strings.EqualFold(pubAddr.String(), c.DcrmAddress) {
		return fmt.Errorf("dcrm address %v and public key address %v is not match", c.DcrmAddress, pubAddr.String())
	}
	return nil
}
