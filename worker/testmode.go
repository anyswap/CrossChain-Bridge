package worker

import (
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// StartTestWork start test mode work
func StartTestWork() {
	go startTestWork()
}

func startTestWork() {
	for {
		args := <-params.ChanIn
		err := process(args)
		if err != nil {
			params.ChanOut <- err.Error()
		} else {
			params.ChanOut <- "success"
		}
	}
}

func process(opts map[string]string) error {
	log.Info("start to process", "opts", opts)

	// parse arguments
	pairID := opts["pairid"]
	if pairID == "" {
		return fmt.Errorf("error: empty pairID")
	}
	txid := opts["txid"]
	if txid == "" {
		return fmt.Errorf("error: empty txid")
	}

	var isSwapin bool
	var txType tokens.SwapTxType

	swapType := opts["swaptype"]
	switch swapType {
	case "swapin":
		isSwapin = true
		txType = tokens.SwapinTx
	case "swapout":
		txType = tokens.SwapoutTx
	default:
		return fmt.Errorf("unknown swap type '%v'", swapType)
	}

	log.Info("parse arguments sucess", "txid", txid, "pairID", pairID, "swapType", swapType)

	srcBridge := tokens.GetCrossChainBridge(isSwapin)
	dstBridge := tokens.GetCrossChainBridge(!isSwapin)

	fromTokenCfg, toTokenCfg := tokens.GetTokenConfigsByDirection(pairID, isSwapin)
	if fromTokenCfg == nil || toTokenCfg == nil {
		return tokens.ErrUnknownPairID
	}

	// verify tx
	swapInfo, err := srcBridge.VerifyTransaction(pairID, txid, false)
	if err != nil {
		return fmt.Errorf("verify tx failed: %w", err)
	}
	log.Info("verify tx success", "txid", txid)

	// build tx
	args := &tokens.BuildTxArgs{
		SwapInfo: tokens.SwapInfo{
			Identifier: params.GetIdentifier(),
			PairID:     pairID,
			SwapID:     txid,
			SwapType:   getSwapType(isSwapin),
			TxType:     txType,
			Bind:       swapInfo.Bind,
		},
		From:        toTokenCfg.DcrmAddress,
		OriginFrom:  swapInfo.From,
		OriginTxTo:  swapInfo.TxTo,
		OriginValue: swapInfo.Value,
	}
	rawTx, err := dstBridge.BuildRawTransaction(args)
	if err != nil {
		return fmt.Errorf("build tx failed: %w", err)
	}
	log.Info("build tx success")

	// sign tx
	var signedTx interface{}
	var signTxHash string
	tokenCfg := dstBridge.GetTokenConfig(pairID)
	if tokenCfg.GetDcrmAddressPrivateKey() != nil {
		signedTx, signTxHash, err = dstBridge.SignTransaction(rawTx, pairID)
	} else {
		signedTx, signTxHash, err = dstBridge.DcrmSignTransaction(rawTx, args)
	}
	if err != nil {
		return fmt.Errorf("sign tx failed: %w", err)
	}
	log.Info("sign tx success", "hash", signTxHash)

	// send tx
	txHash, err := dstBridge.SendTransaction(signedTx)
	if err != nil {
		return fmt.Errorf("send tx failed: %w", err)
	}
	log.Info("send tx success", "hash", txHash)
	return nil
}
