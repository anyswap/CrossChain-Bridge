package terra

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/cosmos"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/tendermint/tendermint/crypto/tmhash"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	core "github.com/terra-project/core/types"
)

// Bridge struct
type Bridge struct {
	*cosmos.Bridge
}

// InitSDK init cosmos sdk
func InitSDK() {
	config := sdk.GetConfig()
	config.SetCoinType(core.CoinType)
	config.SetFullFundraiserPath(core.FullFundraiserPath)
	config.SetBech32PrefixForAccount(core.Bech32PrefixAccAddr, core.Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(core.Bech32PrefixValAddr, core.Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(core.Bech32PrefixConsAddr, core.Bech32PrefixConsPub)
	config.Seal()
}

// BeforeConfig run before loading bridge and token config
func (b *Bridge) BeforeConfig() {
	ctypes.RegisterAmino(cosmos.CDC)
	sdk.RegisterCodec(cosmos.CDC)
	cosmos.CDC.RegisterConcrete(cosmos.MsgSend{}, "bank/MsgSend", nil)
	cosmos.CDC.RegisterConcrete(cosmos.MsgMultiSend{}, "bank/MsgMultiSend", nil)
	cosmos.CDC.RegisterConcrete(authtypes.StdTx{}, "core/StdTx", nil)
	cosmos.CDC.RegisterConcrete(&authtypes.BaseAccount{}, "core/Account", nil)
	InitSDK()
	initTxHashCdc()
	cosmos.CaluculateTxHash = CaluculateTxHash
	cosmos.ChainIDs["columbus-5"] = true
	cosmos.ChainIDs["tequila-0004"] = true
	cosmos.SignBytesModifier = TerraSignBytesModifier
	tokens.IsSwapoutToStringAddress = true
}

// AfterConfig run after loading bridge and token config
func (b *Bridge) AfterConfig() {
	cosmos.GetFeeAmount = b.FeeGetter()
	b.Bridge.InitLatestBlockNumber()
	// Load coins from token configs
	b.LoadCoins()
	// You must add this coin
	// b.SupportedCoins["LUNA"] = cosmos.CosmosCoin{"uluna", 6}
	// You can add these coins to config
	/*
		b.SupportedCoins["UST"] = cosmos.CosmosCoin{"uusd", 6}
		b.SupportedCoins["KRT"] = cosmos.CosmosCoin{"ukrw", 6}
		b.SupportedCoins["SDT"] = cosmos.CosmosCoin{"usdr", 6}
		b.SupportedCoins["CNY"] = cosmos.CosmosCoin{"ucny", 6}
		b.SupportedCoins["JPY"] = cosmos.CosmosCoin{"ujpy", 6}
		b.SupportedCoins["EUR"] = cosmos.CosmosCoin{"ueur", 6}
		b.SupportedCoins["GBP"] = cosmos.CosmosCoin{"ugbp", 6}
		b.SupportedCoins["MNT"] = cosmos.CosmosCoin{"umnt", 6}
	*/
	if luna, ok := b.SupportedCoins["LUNA"]; !ok || luna.Denom != "uluna" || luna.Decimal != 6 {
		log.Info("Terra init coins", "luna", luna, "ok", ok, "check denom", (luna.Denom != "uluna"), "check decimal", luna.Decimal != 6)
		//log.Fatalf("Terra bridge must have Luna token config")
	}
	b.MainCoin = b.SupportedCoins["LUNA"]

	pairs := tokens.GetTokenPairsConfig()
	for _, tokenCfg := range pairs {
		token := tokens.GetTokenConfig(tokenCfg.PairID, false)
		if token.TaxCap <= 0 {
			log.Fatalf("Invalid tax cap")
		}
		if token.TaxRate <= 0 || token.TaxRate >= 0.01 {
			log.Fatalf("Invalid tax tax rate")
		}
		if token.GasRate <= 0 {
			log.Fatalf("Invalid tax gas rate")
		}
	}

	log.Info("Terra bridge init success", "coins", b.SupportedCoins)
}

// NewCrossChainBridge new bridge
func NewCrossChainBridge(isSrc bool) *Bridge {
	return &Bridge{
		Bridge: cosmos.NewCrossChainBridge(isSrc),
	}
}

// SetChainAndGateway set chain and gateway config
func (b *Bridge) SetChainAndGateway(chainCfg *tokens.ChainConfig, gatewayCfg *tokens.GatewayConfig) {
	b.CrossChainBridgeBase.SetChainAndGateway(chainCfg, gatewayCfg)
	b.InitLatestBlockNumber()
	b.VerifyChainID()
}

// VerifyChainID verify chain id
func (b *Bridge) VerifyChainID() {
	chainID := strings.ToLower(b.ChainConfig.NetID)
	if !cosmos.ChainIDs[chainID] {
		log.Fatalf("unsupported cosmos network: %v", b.ChainConfig.NetID)
	}
}

// VerifyTokenConfig verify token config
func (b *Bridge) VerifyTokenConfig(tokenCfg *tokens.TokenConfig) error {
	if !b.IsValidAddress(tokenCfg.DepositAddress) {
		return fmt.Errorf("invalid deposit address: %v", tokenCfg.DepositAddress)
	}
	return nil
}

// InitLatestBlockNumber init latest block number
func (b *Bridge) InitLatestBlockNumber() {
	chainCfg := b.ChainConfig
	gatewayCfg := b.GatewayConfig
	var latest uint64
	var err error
	for {
		latest, err = b.GetLatestBlockNumber()
		if err == nil {
			tokens.SetLatestBlockHeight(latest, b.IsSrc)
			log.Info("get latst block number succeed.", "number", latest, "BlockChain", chainCfg.BlockChain, "NetID", chainCfg.NetID)
			break
		}
		log.Error("get latst block number failed.", "BlockChain", chainCfg.BlockChain, "NetID", chainCfg.NetID, "err", err)
		log.Println("retry query gateway", gatewayCfg.APIAddress)
		time.Sleep(3 * time.Second)
	}
}

// DefaultSwapoutGas is terra default gas
var DefaultSwapoutGas uint64 = 98000
var Denominator int64 = 10000

// FeeGetter returns terra fee getter
func (b *Bridge) FeeGetter() func(pairID string, tx *cosmos.StdSignContent) authtypes.StdFee {
	return func(pairID string, tx *cosmos.StdSignContent) authtypes.StdFee {
		tokenCfg := b.GetTokenConfig(pairID)
		denom := tokenCfg.Unit
		var amount int64
		switch denom {
		case "uluna":
			amount = 150000
		case "uusd":
			if len(tx.Msgs) != 1 {
				amount = 5000000
				break
			}
			sendmsg, ok := tx.Msgs[0].(cosmos.MsgSend)
			if !ok {
				amount = 5000000
				break
			}
			if len(sendmsg.Amount) != 1 {
				amount = 5000000
				break
			}
			sendamt := sendmsg.Amount[0].Amount.BigInt()

			token := tokens.GetTokenConfig(pairID, false)
			taxrate := big.NewInt(int64(token.TaxRate * float64(Denominator)))

			// fee = swapoutgas * gasrate + tax
			// tax = min(sendamount * taxrate, taxcap)
			tax := new(big.Int).Div(new(big.Int).Mul(sendamt, taxrate), big.NewInt(Denominator)).Int64()
			taxcap := int64(token.TaxCap * 1e6)
			if tax > taxcap {
				tax = taxcap
			}
			gasfee := int64(float64(DefaultSwapoutGas) * token.GasRate)
			amount = tax + gasfee
		}

		feeAmount := sdk.Coins{sdk.Coin{Denom: denom, Amount: sdk.NewInt(amount)}}
		return authtypes.NewStdFee(DefaultSwapoutGas, feeAmount)
	}
}

// TerraSignBytesModifier is used to build terra special sign bytes
var TerraSignBytesModifier = func(bz []byte) []byte {
	signString := string(bz)
	signString = strings.Replace(signString, "cosmos-sdk/MsgSend", "bank/MsgSend", -1)
	signString = strings.Replace(signString, "cosmos-sdk/MsgMultiSend", "bank/MsgMultiSend", -1)
	return []byte(signString)
}

var txhashcdc *codec.Codec

func initTxHashCdc() {
	txhashcdc = codec.New()
	codec.RegisterCrypto(txhashcdc)
	cosmos.RegisterCodec(txhashcdc)
	txhashcdc.RegisterConcrete(authtypes.StdTx{}, "auth/StdTx", nil)
	txhashcdc.RegisterInterface((*sdk.Msg)(nil), nil)
}

func CaluculateTxHash(signedTx cosmos.HashableStdTx) (string, error) {
	txBytes, err := txhashcdc.MarshalBinaryLengthPrefixed(signedTx.ToStdTx())
	if err != nil {
		return "", err
	}
	txHash := fmt.Sprintf("%X", tmhash.Sum(txBytes))
	return txHash, nil
}
