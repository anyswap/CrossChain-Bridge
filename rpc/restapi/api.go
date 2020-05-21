package restapi

import (
	"encoding/json"
	"fmt"
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

func GetRawSwapinHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	txid := vars["txid"]
	res, err := swapapi.GetRawSwapin(&txid)
	writeResponse(w, res, err)
}

func GetRawSwapinResultHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	txid := vars["txid"]
	res, err := swapapi.GetRawSwapinResult(&txid)
	writeResponse(w, res, err)
}

func GetSwapinHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	txid := vars["txid"]
	res, err := swapapi.GetSwapin(&txid)
	writeResponse(w, res, err)
}

func GetRawSwapoutHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	txid := vars["txid"]
	res, err := swapapi.GetRawSwapout(&txid)
	writeResponse(w, res, err)
}

func GetRawSwapoutResultHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	txid := vars["txid"]
	res, err := swapapi.GetRawSwapoutResult(&txid)
	writeResponse(w, res, err)
}

func GetSwapoutHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	txid := vars["txid"]
	res, err := swapapi.GetSwapout(&txid)
	writeResponse(w, res, err)
}

func getHistoryParams(r *http.Request) (address string, offset int, limit int, err error) {
	vars := mux.Vars(r)
	vals := r.URL.Query()

	address = vars["address"]

	offset_val, exist := vals["offset"]
	if exist {
		offset, err = common.GetIntFromStr(offset_val[0])
		if err != nil {
			return address, offset, limit, err
		}
	}

	limit_val, exist := vals["limit"]
	if exist {
		limit, err = common.GetIntFromStr(limit_val[0])
		if err != nil {
			return address, offset, limit, err
		}
	}

	return address, offset, limit, nil
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
	txid := vars["txid"]
	res, err := swapapi.Swapin(&txid)
	writeResponse(w, res, err)
}

func PostSwapoutHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	txid := vars["txid"]
	res, err := swapapi.Swapout(&txid)
	writeResponse(w, res, err)
}

func RecallSwapinHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	txid := vars["txid"]
	res, err := swapapi.RecallSwapin(&txid)
	writeResponse(w, res, err)
}
