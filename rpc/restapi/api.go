package restapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/internal/swapapi"
	"github.com/gorilla/mux"
)

func writeResponse(w http.ResponseWriter, resp interface{}, err error) {
	// Note: must set header before write header
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(http.StatusOK)
	if err == nil {
		jsonData, _ := json.Marshal(resp)
		_, _ = w.Write(jsonData)
	} else {
		fmt.Fprintln(w, err.Error())
	}
}

// SeverInfoHandler handler
func SeverInfoHandler(w http.ResponseWriter, r *http.Request) {
	res, err := swapapi.GetServerInfo()
	writeResponse(w, res, err)
}

// StatisticsHandler handler
func StatisticsHandler(w http.ResponseWriter, r *http.Request) {
	res, err := swapapi.GetSwapStatistics()
	writeResponse(w, res, err)
}

// GetRawSwapinHandler handler
func GetRawSwapinHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txid := vars["txid"]
	res, err := swapapi.GetRawSwapin(&txid)
	writeResponse(w, res, err)
}

// GetRawSwapinResultHandler handler
func GetRawSwapinResultHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txid := vars["txid"]
	res, err := swapapi.GetRawSwapinResult(&txid)
	writeResponse(w, res, err)
}

// GetSwapinHandler handler
func GetSwapinHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txid := vars["txid"]
	res, err := swapapi.GetSwapin(&txid)
	writeResponse(w, res, err)
}

// GetRawSwapoutHandler handler
func GetRawSwapoutHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txid := vars["txid"]
	res, err := swapapi.GetRawSwapout(&txid)
	writeResponse(w, res, err)
}

// GetRawSwapoutResultHandler handler
func GetRawSwapoutResultHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txid := vars["txid"]
	res, err := swapapi.GetRawSwapoutResult(&txid)
	writeResponse(w, res, err)
}

// GetSwapoutHandler handler
func GetSwapoutHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txid := vars["txid"]
	res, err := swapapi.GetSwapout(&txid)
	writeResponse(w, res, err)
}

func getHistoryParams(r *http.Request) (address string, offset, limit int, err error) {
	vars := mux.Vars(r)
	vals := r.URL.Query()

	address = vars["address"]

	offsetStr, exist := vals["offset"]
	if exist {
		offset, err = common.GetIntFromStr(offsetStr[0])
		if err != nil {
			return address, offset, limit, err
		}
	}

	limitStr, exist := vals["limit"]
	if exist {
		limit, err = common.GetIntFromStr(limitStr[0])
		if err != nil {
			return address, offset, limit, err
		}
	}

	return address, offset, limit, nil
}

// SwapinHistoryHandler handler
func SwapinHistoryHandler(w http.ResponseWriter, r *http.Request) {
	address, offset, limit, err := getHistoryParams(r)
	if err != nil {
		writeResponse(w, nil, err)
	} else {
		res, err := swapapi.GetSwapinHistory(address, offset, limit)
		writeResponse(w, res, err)
	}
}

// SwapoutHistoryHandler handler
func SwapoutHistoryHandler(w http.ResponseWriter, r *http.Request) {
	address, offset, limit, err := getHistoryParams(r)
	if err != nil {
		writeResponse(w, nil, err)
	} else {
		res, err := swapapi.GetSwapoutHistory(address, offset, limit)
		writeResponse(w, res, err)
	}
}

// PostSwapinHandler handler
func PostSwapinHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txid := vars["txid"]
	res, err := swapapi.Swapin(&txid)
	writeResponse(w, res, err)
}

// RetrySwapinHandler handler
func RetrySwapinHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txid := vars["txid"]
	res, err := swapapi.RetrySwapin(&txid)
	writeResponse(w, res, err)
}

// PostP2shSwapinHandler handler
func PostP2shSwapinHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txid := vars["txid"]
	bind := vars["bind"]
	res, err := swapapi.P2shSwapin(&txid, &bind)
	writeResponse(w, res, err)
}

// PostSwapoutHandler handler
func PostSwapoutHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txid := vars["txid"]
	res, err := swapapi.Swapout(&txid)
	writeResponse(w, res, err)
}

// RegisterP2shAddress handler
func RegisterP2shAddress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]
	res, err := swapapi.RegisterP2shAddress(address)
	writeResponse(w, res, err)
}

// GetP2shAddressInfo handler
func GetP2shAddressInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]
	res, err := swapapi.GetP2shAddressInfo(address)
	writeResponse(w, res, err)
}

// RegisterAddress handler
func RegisterAddress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]
	res, err := swapapi.RegisterAddress(address)
	writeResponse(w, res, err)
}

// GetRegisteredAddress handler
func GetRegisteredAddress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]
	res, err := swapapi.GetRegisteredAddress(address)
	writeResponse(w, res, err)
}
