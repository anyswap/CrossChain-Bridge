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

// GatewayConfig struct
type GatewayConfig struct {
	APIAddress    []string
	APIAddressExt []string
	Extras        *GatewayExtras `json:",omitempty"`
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
	EnableReplaceSwap  bool
	EnableDynamicFeeTx bool

	CallByContractWhitelist []string `json:",omitempty"`

	MinReserveFee              string
	BaseFeePercent             int64
	BaseGasPrice               string `json:",omitempty"`
	MaxGasPriceFluctPercent    uint64 `json:",omitempty"`
	ReplacePlusGasPricePercent uint64 `json:",omitempty"`
	WaitTimeToReplace          int64  // seconds
	MaxReplaceCount            int
	FixedGasPrice              string `json:",omitempty"`
	MaxGasPrice                string `json:",omitempty"`

	PlusGasTipCapPercent uint64
	PlusGasFeeCapPercent uint64
	BlockCountFeeHistory int
	MaxGasTipCap         string
	MaxGasFeeCap         string

	// cached values
	chainID       *big.Int
	fixedGasPrice *big.Int
	maxGasPrice   *big.Int
	minReserveFee *big.Int
	maxGasTipCap  *big.Int
	maxGasFeeCap  *big.Int

	callByContractWhitelist map[string]struct{}
}

// TokenPriceConfig struct
type TokenPriceConfig struct {
	Contract   string
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
	DcrmPubkey             string   `json:"-"`
	ContractAddress        string   `json:",omitempty"`
	ContractCodeHash       string   `json:",omitempty"`
	MaximumSwap            *float64 // whole unit (eg. BTC, ETH, FSN), not Satoshi
	MinimumSwap            *float64 // whole unit
	BigValueThreshold      *float64
	SwapFeeRate            *float64
	MaximumSwapFee         *float64
	MinimumSwapFee         *float64
	TokenPrice             float64 `toml:"-"`
	PlusGasPricePercentage uint64  `json:",omitempty"`
	DisableSwap            bool
	IsDelegateContract     bool
	DelegateToken          string `json:",omitempty"`
	IsAnyswapAdapter       bool   `json:",omitempty"` // PRQ
	IsMappingTokenProxy    bool   `json:",omitempty"` // VTX

	DefaultGasLimit          uint64 `json:",omitempty"`
	AllowSwapinFromContract  bool   `json:",omitempty"`
	AllowSwapoutFromContract bool   `json:",omitempty"`

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
}

// CheckConfig check chain config
func (c *ChainConfig) CheckConfig(isServer bool) error {
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
	if c.MaxGasPriceFluctPercent > 100 {
		return errors.New("'MaxGasPriceFluctPercent' is too large (>100)")
	}
	if c.ReplacePlusGasPricePercent > 100 {
		return errors.New("'ReplacePlusGasPricePercent' is too large (>100)")
	}
	if c.BaseGasPrice != "" {
		if _, err := common.GetBigIntFromStr(c.BaseGasPrice); err != nil {
			return errors.New("wrong 'BaseGasPrice'")
		}
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
	if c.EnableDynamicFeeTx {
		if c.MaxGasTipCap != "" {
			bi, err := common.GetBigIntFromStr(c.MaxGasTipCap)
			if err != nil {
				return errors.New("wrong 'MaxGasTipCap'")
			}
			c.maxGasTipCap = bi
		}
		if c.MaxGasFeeCap != "" {
			bi, err := common.GetBigIntFromStr(c.MaxGasFeeCap)
			if err != nil {
				return errors.New("wrong 'MaxGasFeeCap'")
			}
			c.maxGasFeeCap = bi
		}
		if c.PlusGasTipCapPercent > 100 {
			return errors.New("too large 'PlusGasTipCapPercent'")
		}
		if c.PlusGasFeeCapPercent > 100 {
			return errors.New("too large 'PlusGasFeeCapPercent'")
		}
		if c.BlockCountFeeHistory > 1024 {
			return errors.New("too large 'BlockCountFeeHistory'")
		}
		if isServer {
			if c.maxGasTipCap == nil {
				return errors.New("server must config 'MaxGasTipCap'")
			}
			if c.maxGasFeeCap == nil {
				return errors.New("server must config 'MaxGasFeeCap'")
			}
			if c.maxGasTipCap.Cmp(c.maxGasFeeCap) > 0 {
				return errors.New("must satisfy 'MaxGasTipCap <= MaxGasFeeCap'")
			}
		}
	}
	log.Info("check chain config success",
		"blockChain", c.BlockChain,
		"fixedGasPrice", c.FixedGasPrice,
		"maxGasPrice", c.MaxGasPrice,
		"baseFeePercent", c.BaseFeePercent,
	)
	return nil
}

// CheckConfig check token config
//nolint:funlen,gocyclo // keep TokenConfig check as whole
func (c *TokenConfig) CheckConfig(isSrc bool) (err error) {
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
	if c.PlusGasPricePercentage > MaxPlusGasPricePercentage {
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
	if c.IsErc20() && c.ContractAddress == "" {
		return errors.New("token must config 'ContractAddress' for ERC20 in source chain")
	}
	if c.AllowSwapinFromContract {
		if !isSrc || !c.IsErc20() {
			return errors.New("only source ERC20 token allow swapin from contract")
		}
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
		if c.DelegateToken == "" || !common.IsHexAddress(c.DelegateToken) {
			return errors.New("wrong 'DelegateToken' address")
		}
		if c.IsProxyErc20() {
			return errors.New("token can not be both IsDelegateContract and ProxyERC20")
		}
		if c.IsMappingTokenProxy {
			return errors.New("token can not be both IsDelegateContract and IsMappingTokenProxy")
		}
	} else if c.DelegateToken != "" {
		return errors.New("token forbid config 'DelegateToken' if 'IsDelegateContract' is false")
	}
	c.TokenPrice = 0
	err = c.loadTokenPrice(isSrc)
	if err != nil {
		return err
	}
	c.CalcAndStoreValue()
	err = c.LoadDcrmAddressPrivateKey()
	if err != nil {
		return err
	}
	err = c.VerifyDcrmPublicKey()
	if err != nil {
		return err
	}
	log.Info("check token config success",
		"id", c.ID, "name", c.Name, "symbol", c.Symbol, "decimals", *c.Decimals,
		"depositAddress", c.DepositAddress, "contractAddress", c.ContractAddress,
	)
	return nil
}

// CalcAndStoreValue calc and store value (minus duplicate calculation)
func (c *TokenConfig) CalcAndStoreValue() {
	maxSwap := *c.MaximumSwap
	minSwap := *c.MinimumSwap
	bigSwap := *c.BigValueThreshold
	maxFee := *c.MaximumSwapFee
	minFee := *c.MinimumSwapFee
	if c.TokenPrice > 0 {
		// convert to token amount
		maxSwap /= c.TokenPrice
		minSwap /= c.TokenPrice
		bigSwap /= c.TokenPrice
		maxFee /= c.TokenPrice
		minFee /= c.TokenPrice
	}
	smallBiasValue := 0.0001
	c.maxSwap = ToBits(maxSwap+smallBiasValue, *c.Decimals)
	c.minSwap = ToBits(minSwap-smallBiasValue, *c.Decimals)
	c.maxSwapFee = ToBits(maxFee, *c.Decimals)
	c.minSwapFee = ToBits(minFee, *c.Decimals)
	c.bigValThreshhold = ToBits(bigSwap+smallBiasValue, *c.Decimals)
	log.Info("calc and store token swap and fee success",
		"name", c.Name, "decimals", *c.Decimals, "contractAddress", c.ContractAddress,
		"maxSwap", c.maxSwap, "minSwap", c.minSwap, "bigValThreshhold", c.bigValThreshhold,
		"maxSwapFee", c.maxSwapFee, "minSwapFee", c.minSwapFee, "swapFeeRate", c.SwapFeeRate,
	)
}

// SetChainID set chainID
func (c *ChainConfig) SetChainID(chainID *big.Int) {
	c.chainID = chainID
}

// GetChainID get chainID
func (c *ChainConfig) GetChainID() *big.Int {
	return c.chainID
}

// IsInCallByContractWhitelist is in call by contract whitelist
func (c *ChainConfig) IsInCallByContractWhitelist(caller string) bool {
	if c.callByContractWhitelist == nil {
		return false
	}
	_, exist := c.callByContractWhitelist[strings.ToLower(caller)]
	return exist
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

// GetMinReserveFee get min reserve fee
func (c *ChainConfig) GetMinReserveFee() *big.Int {
	return c.minReserveFee
}

// GetMaxGasTipCap get max gas tip cap
func (c *ChainConfig) GetMaxGasTipCap() *big.Int {
	return c.maxGasTipCap
}

// GetMaxGasFeeCap get max fee gas cap
func (c *ChainConfig) GetMaxGasFeeCap() *big.Int {
	return c.maxGasFeeCap
}

// IsErc20 return if token is erc20
func (c *TokenConfig) IsErc20() bool {
	return strings.EqualFold(c.ID, "ERC20") || c.IsProxyErc20()
}

// IsProxyErc20 return if token is proxy contract of erc20
func (c *TokenConfig) IsProxyErc20() bool {
	return strings.EqualFold(c.ID, "ProxyERC20")
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
			return fmt.Errorf("wrong private key, %w", err)
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
