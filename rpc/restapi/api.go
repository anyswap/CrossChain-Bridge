package restapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/internal/swapapi"
	"github.com/anyswap/CrossChain-Bridge/params"
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

// VersionInfoHandler handler
func VersionInfoHandler(w http.ResponseWriter, r *http.Request) {
	version := params.VersionWithMeta
	writeResponse(w, version, nil)
}

// ServerInfoHandler handler
func ServerInfoHandler(w http.ResponseWriter, r *http.Request) {
	res, err := swapapi.GetServerInfo()
	writeResponse(w, res, err)
}

// TokenPairInfoHandler handler
func TokenPairInfoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pairID := vars["pairid"]
	res, err := swapapi.GetTokenPairInfo(pairID)
	writeResponse(w, res, err)
}

// StatisticsHandler handler
func StatisticsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pairID := vars["pairid"]
	res, err := swapapi.GetSwapStatistics(pairID)
	writeResponse(w, res, err)
}

func getBindParam(r *http.Request) string {
	vals := r.URL.Query()
	bindVals, exist := vals["bind"]
	if exist {
		return bindVals[0]
	}
	return ""
}

// GetRawSwapinHandler handler
func GetRawSwapinHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txid := vars["txid"]
	pairID := vars["pairid"]
	bind := getBindParam(r)
	res, err := swapapi.GetRawSwapin(&txid, &pairID, &bind)
	writeResponse(w, res, err)
}

// GetRawSwapinResultHandler handler
func GetRawSwapinResultHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txid := vars["txid"]
	pairID := vars["pairid"]
	bind := getBindParam(r)
	res, err := swapapi.GetRawSwapinResult(&txid, &pairID, &bind)
	writeResponse(w, res, err)
}

// GetSwapinHandler handler
func GetSwapinHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txid := vars["txid"]
	pairID := vars["pairid"]
	bind := getBindParam(r)
	res, err := swapapi.GetSwapin(&txid, &pairID, &bind)
	writeResponse(w, res, err)
}

// GetRawSwapoutHandler handler
func GetRawSwapoutHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txid := vars["txid"]
	pairID := vars["pairid"]
	bind := getBindParam(r)
	res, err := swapapi.GetRawSwapout(&txid, &pairID, &bind)
	writeResponse(w, res, err)
}

// GetRawSwapoutResultHandler handler
func GetRawSwapoutResultHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txid := vars["txid"]
	pairID := vars["pairid"]
	bind := getBindParam(r)
	res, err := swapapi.GetRawSwapoutResult(&txid, &pairID, &bind)
	writeResponse(w, res, err)
}

// GetSwapoutHandler handler
func GetSwapoutHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txid := vars["txid"]
	pairID := vars["pairid"]
	bind := getBindParam(r)
	res, err := swapapi.GetSwapout(&txid, &pairID, &bind)
	writeResponse(w, res, err)
}

func getHistoryParams(r *http.Request) (address, pairID string, offset, limit int, err error) {
	vars := mux.Vars(r)
	vals := r.URL.Query()

	address = vars["address"]
	pairID = vars["pairid"]

	offsetStr, exist := vals["offset"]
	if exist {
		offset, err = common.GetIntFromStr(offsetStr[0])
		if err != nil {
			return address, pairID, offset, limit, err
		}
	}

	limitStr, exist := vals["limit"]
	if exist {
		limit, err = common.GetIntFromStr(limitStr[0])
		if err != nil {
			return address, pairID, offset, limit, err
		}
	}

	return address, pairID, offset, limit, nil
}

// SwapinHistoryHandler handler
func SwapinHistoryHandler(w http.ResponseWriter, r *http.Request) {
	address, pairID, offset, limit, err := getHistoryParams(r)
	if err != nil {
		writeResponse(w, nil, err)
	} else {
		res, err := swapapi.GetSwapinHistory(address, pairID, offset, limit)
		writeResponse(w, res, err)
	}
}

// SwapoutHistoryHandler handler
func SwapoutHistoryHandler(w http.ResponseWriter, r *http.Request) {
	address, pairID, offset, limit, err := getHistoryParams(r)
	if err != nil {
		writeResponse(w, nil, err)
	} else {
		res, err := swapapi.GetSwapoutHistory(address, pairID, offset, limit)
		writeResponse(w, res, err)
	}
}

// PostSwapinHandler handler
func PostSwapinHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txid := vars["txid"]
	pairID := vars["pairid"]
	res, err := swapapi.Swapin(&txid, &pairID)
	writeResponse(w, res, err)
}

// RetrySwapinHandler handler
func RetrySwapinHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txid := vars["txid"]
	pairID := vars["pairid"]
	res, err := swapapi.RetrySwapin(&txid, &pairID)
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
	pairID := vars["pairid"]
	res, err := swapapi.Swapout(&txid, &pairID)
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
