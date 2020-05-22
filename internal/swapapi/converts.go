package swapapi

import (
	"github.com/fsn-dev/crossChain-Bridge/mongodb"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
)

func ConvertMgoSwapToSwapInfo(ms *mongodb.MgoSwap) *SwapInfo {
	return &SwapInfo{
		TxId:      ms.TxId,
		Status:    ms.Status,
		Timestamp: ms.Timestamp,
		Memo:      ms.Memo,
	}
}

func ConvertMgoSwapsToSwapInfos(msSlice []*mongodb.MgoSwap) []*SwapInfo {
	result := make([]*SwapInfo, len(msSlice))
	for k, v := range msSlice {
		result[k] = ConvertMgoSwapToSwapInfo(v)
	}
	return result
}

func ConvertMgoSwapResultToSwapInfo(mr *mongodb.MgoSwapResult) *SwapInfo {
	var confirmations uint64
	if mr.SwapHeight != 0 {
		var latest uint64
		switch mr.SwapType {
		case uint32(tokens.Swap_Swapin):
			latest = tokens.DstLatestBlockHeight
		case uint32(tokens.Swap_Swapout), uint32(tokens.Swap_Recall):
			latest = tokens.SrcLatestBlockHeight
		}
		if latest > mr.SwapHeight {
			confirmations = latest - mr.SwapHeight
		}
	}
	return &SwapInfo{
		TxId:          mr.TxId,
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
		Status:        mr.Status,
		Timestamp:     mr.Timestamp,
		Memo:          mr.Memo,
		Confirmations: confirmations,
	}
}

func ConvertMgoSwapResultsToSwapInfos(mrSlice []*mongodb.MgoSwapResult) []*SwapInfo {
	result := make([]*SwapInfo, len(mrSlice))
	for k, v := range mrSlice {
		result[k] = ConvertMgoSwapResultToSwapInfo(v)
	}
	return result
}
