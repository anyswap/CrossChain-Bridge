package swapapi

import (
	"encoding/hex"
	"errors"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// ConvertMgoSwapToSwapInfo convert
func ConvertMgoSwapToSwapInfo(ms *mongodb.MgoSwap) *SwapInfo {
	return &SwapInfo{
		PairID:    ms.PairID,
		TxID:      ms.TxID,
		TxTo:      ms.TxTo,
		Bind:      ms.Bind,
		Status:    ms.Status,
		StatusMsg: ms.Status.String(),
		Timestamp: ms.Timestamp,
		Memo:      ms.Memo,
	}
}

// ConvertMgoSwapsToSwapInfos convert
func ConvertMgoSwapsToSwapInfos(msSlice []*mongodb.MgoSwap) []*SwapInfo {
	result := make([]*SwapInfo, len(msSlice))
	for k, v := range msSlice {
		result[k] = ConvertMgoSwapToSwapInfo(v)
	}
	return result
}

// ConvertMgoSwapResultToSwapInfo convert
func ConvertMgoSwapResultToSwapInfo(mr *mongodb.MgoSwapResult) *SwapInfo {
	var confirmations uint64
	if mr.SwapHeight != 0 {
		var latest uint64
		switch mr.SwapType {
		case uint32(tokens.SwapinType):
			latest = tokens.DstLatestBlockHeight
		case uint32(tokens.SwapoutType):
			latest = tokens.SrcLatestBlockHeight
		}
		if latest > mr.SwapHeight {
			confirmations = latest - mr.SwapHeight
		}
	}
	return &SwapInfo{
		PairID:        mr.PairID,
		TxID:          mr.TxID,
		TxTo:          mr.TxTo,
		TxHeight:      mr.TxHeight,
		TxTime:        mr.TxTime,
		From:          mr.From,
		To:            mr.To,
		Bind:          mr.Bind,
		Value:         mr.Value,
		SwapTx:        mr.SwapTx,
		SwapHeight:    mr.SwapHeight,
		SwapTime:      mr.SwapTime,
		SwapValue:     mr.SwapValue,
		SwapType:      mr.SwapType,
		SwapNonce:     mr.SwapNonce,
		Status:        mr.Status,
		StatusMsg:     mr.Status.String(),
		Timestamp:     mr.Timestamp,
		Memo:          mr.Memo,
		Confirmations: confirmations,
	}
}

// ConvertMgoSwapResultsToSwapInfos convert
func ConvertMgoSwapResultsToSwapInfos(mrSlice []*mongodb.MgoSwapResult) []*SwapInfo {
	result := make([]*SwapInfo, len(mrSlice))
	for k, v := range mrSlice {
		result[k] = ConvertMgoSwapResultToSwapInfo(v)
	}
	return result
}

// ConvertMgoSwapAgreementToSwapAgreement convert
func ConvertMgoSwapAgreementToSwapAgreement(mp *mongodb.MgoSwapAgreement) (tokens.SwapAgreement, error) {
	bz, err := hex.DecodeString(mp.Value)
	if err != nil {
		return nil, err
	}
	var p tokens.SwapAgreement
	err = tokens.TokenCDC.UnmarshalJSON(bz, &p)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(p.Type(), mp.Type) == false {
		return nil, errors.New("Swapin agreement type not match")
	}
	return p, nil
}
