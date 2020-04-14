package restapi

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/internal/swapapi"
	"github.com/gorilla/mux"
)

func writeResponse(w http.ResponseWriter, resp interface{}, err error) {
	if err == nil {
		jsonData, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	} else {
		fmt.Fprintln(w, err.Error())
	}
}

func SeverInfoHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	res, err := swapapi.GetServerInfo()
	writeResponse(w, res, err)
}

func StatisticsHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	res, err := swapapi.GetSwapStatistics()
	writeResponse(w, res, err)
}

func GetSwapinHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	txid := common.HexToHash(vars["txid"])
	res, err := swapapi.GetSwapin(&txid)
	writeResponse(w, res, err)
}

func GetSwapoutHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	txid := common.HexToHash(vars["txid"])
	res, err := swapapi.GetSwapout(&txid)
	writeResponse(w, res, err)
}

func getHistoryParams(r *http.Request) (*common.Address, int, int, error) {
	vars := mux.Vars(r)
	vals := r.URL.Query()

	address := common.HexToAddress(vars["address"])
	offset := 0
	limit := 20

	offset_val, exist := vals["offset"]
	if exist {
		bi, ok := new(big.Int).SetString(offset_val[0], 0)
		if !ok || !bi.IsUint64() || bi.Uint64() > uint64(common.MaxInt) {
			err := fmt.Errorf("wrong offset")
			return &address, offset, limit, err
		}
		offset = int(bi.Uint64())
	}

	limit_val, exist := vals["limit"]
	if exist {
		bi, ok := new(big.Int).SetString(limit_val[0], 0)
		if !ok || !bi.IsUint64() || bi.Uint64() > uint64(common.MaxInt) {
			err := fmt.Errorf("wrong offset")
			return &address, offset, limit, err
		}
		limit = int(bi.Uint64())
	}

	return &address, offset, limit, nil
}

func SwapinHistoryHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	address, offset, limit, err := getHistoryParams(r)
	if err != nil {
		writeResponse(w, nil, err)
	} else {
		res, err := swapapi.GetSwapinHistory(address, offset, limit)
		writeResponse(w, res, err)
	}
}

func SwapoutHistoryHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	address, offset, limit, err := getHistoryParams(r)
	if err != nil {
		writeResponse(w, nil, err)
	} else {
		res, err := swapapi.GetSwapoutHistory(address, offset, limit)
		writeResponse(w, res, err)
	}
}

func PostSwapinHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	txid := common.HexToHash(vars["txid"])
	res, err := swapapi.Swapin(&txid)
	writeResponse(w, res, err)
}

func PostSwapoutHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	txid := common.HexToHash(vars["txid"])
	res, err := swapapi.Swapout(&txid)
	writeResponse(w, res, err)
}

func RecallSwapinHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	txid := common.HexToHash(vars["txid"])
	res, err := swapapi.RecallSwapin(&txid)
	writeResponse(w, res, err)
}
