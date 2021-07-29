package colx

import (
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc"
)

var (
	cfgMinRelayFee       int64  = 300000000
	cfgMinRelayFeePerKb  int64  = 100000000
	cfgMaxRelayFeePerKb  int64  = 1000000000
	cfgPlusFeePercentage uint64 = 20
	cfgEstimateFeeBlocks        = 3

	cfgFromPublicKey string

	cfgUtxoAggregateMinCount  = 1
	cfgUtxoAggregateMinValue  = uint64(100000000)
	cfgUtxoAggregateToAddress string
)

// Init init colx extra
func Init(btcExtra *tokens.BtcExtraConfig) {
	if btc.BridgeInstance == nil {
		return
	}

	if btcExtra == nil {
		log.Fatal("Colx bridge must config 'BtcExtra'")
	}

	initFromPublicKey()
	initRelayFee(btcExtra)
	initAggregate(btcExtra)
}

func initFromPublicKey() {
	if len(tokens.GetTokenPairsConfig()) != 1 {
		log.Fatalf("Colx bridge does not support multiple tokens")
	}

	pairCfg, exist := tokens.GetTokenPairsConfig()[PairID]
	if !exist {
		log.Fatalf("Colx bridge must have pairID %v", PairID)
	}

	cfgFromPublicKey = pairCfg.SrcToken.DcrmPubkey
	_, err := btc.BridgeInstance.GetCompressedPublicKey(cfgFromPublicKey, true)
	if err != nil {
		log.Fatal("wrong colx dcrm public key", "err", err)
	}
}

func initRelayFee(btcExtra *tokens.BtcExtraConfig) {
	if btcExtra.MinRelayFee > 0 {
		cfgMinRelayFee = btcExtra.MinRelayFee
		maxMinRelayFee, _ := newAmount(5)
		minRelayFee := colxAmountType(cfgMinRelayFee)
		if minRelayFee > maxMinRelayFee {
			log.Fatal("BtcMinRelayFee is too large", "value", minRelayFee, "max", maxMinRelayFee)
		}
	}

	if btcExtra.EstimateFeeBlocks > 0 {
		cfgEstimateFeeBlocks = btcExtra.EstimateFeeBlocks
		if cfgEstimateFeeBlocks > 25 {
			log.Fatal("EstimateFeeBlocks is too large, must <= 25")
		}
	}

	if btcExtra.PlusFeePercentage > 0 {
		cfgPlusFeePercentage = btcExtra.PlusFeePercentage
		if cfgPlusFeePercentage > 5000 {
			log.Fatal("PlusFeePercentage is too large, must <= 5000")
		}
	}

	if btcExtra.MaxRelayFeePerKb > 0 {
		cfgMaxRelayFeePerKb = btcExtra.MaxRelayFeePerKb
	}

	if btcExtra.MinRelayFeePerKb > 0 {
		cfgMinRelayFeePerKb = btcExtra.MinRelayFeePerKb
	}

	if cfgMinRelayFeePerKb > cfgMaxRelayFeePerKb {
		log.Fatal("MinRelayFeePerKb is larger than MaxRelayFeePerKb", "min", cfgMinRelayFeePerKb, "max", cfgMaxRelayFeePerKb)
	}

	log.Info("Init Btc extra", "MinRelayFee", cfgMinRelayFee, "MinRelayFeePerKb", cfgMinRelayFeePerKb, "MaxRelayFeePerKb", cfgMaxRelayFeePerKb, "PlusFeePercentage", cfgPlusFeePercentage)
}

func initAggregate(btcExtra *tokens.BtcExtraConfig) {
	if btcExtra.UtxoAggregateMinCount > 0 {
		cfgUtxoAggregateMinCount = btcExtra.UtxoAggregateMinCount
	}

	if btcExtra.UtxoAggregateMinValue > 0 {
		cfgUtxoAggregateMinValue = btcExtra.UtxoAggregateMinValue
	}

	cfgUtxoAggregateToAddress = btcExtra.UtxoAggregateToAddress
	if !btc.BridgeInstance.IsValidAddress(cfgUtxoAggregateToAddress) {
		log.Fatal("wrong utxo aggregate to address", "toAddress", cfgUtxoAggregateToAddress)
	}

	log.Info("Init Btc extra", "UtxoAggregateMinCount", cfgUtxoAggregateMinCount, "UtxoAggregateMinValue", cfgUtxoAggregateMinValue, "UtxoAggregateToAddress", cfgUtxoAggregateToAddress)
}
