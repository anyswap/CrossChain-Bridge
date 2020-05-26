package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/rpc/v2"
	rpcjson "github.com/gorilla/rpc/v2/json2"

	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/params"
	"github.com/fsn-dev/crossChain-Bridge/rpc/restapi"
	"github.com/fsn-dev/crossChain-Bridge/rpc/rpcapi"
)

func StartAPIServer() {
	rpcserver := rpc.NewServer()
	rpcserver.RegisterCodec(rpcjson.NewCodec(), "application/json")
	rpcserver.RegisterService(new(rpcapi.RpcAPI), "swap")

	r := mux.NewRouter()
	r.Handle("/rpc", rpcserver)
	r.HandleFunc("/serverinfo", restapi.SeverInfoHandler).Methods("GET")
	r.HandleFunc("/statistics", restapi.StatisticsHandler).Methods("GET")
	r.HandleFunc("/swapin/post/{txid}", restapi.PostSwapinHandler).Methods("POST")
	r.HandleFunc("/swapout/post/{txid}", restapi.PostSwapoutHandler).Methods("POST")
	r.HandleFunc("/swapin/recall/{txid}", restapi.RecallSwapinHandler).Methods("POST")
	r.HandleFunc("/swapin/{txid}", restapi.GetSwapinHandler).Methods("GET")
	r.HandleFunc("/swapout/{txid}", restapi.GetSwapoutHandler).Methods("GET")
	r.HandleFunc("/swapin/{txid}/raw", restapi.GetRawSwapinHandler).Methods("GET")
	r.HandleFunc("/swapout/{txid}/raw", restapi.GetRawSwapoutHandler).Methods("GET")
	r.HandleFunc("/swapin/{txid}/rawresult", restapi.GetRawSwapinResultHandler).Methods("GET")
	r.HandleFunc("/swapout/{txid}/rawresult", restapi.GetRawSwapoutResultHandler).Methods("GET")
	r.HandleFunc("/swapin/history/{address}", restapi.SwapinHistoryHandler).Methods("GET")
	r.HandleFunc("/swapout/history/{address}", restapi.SwapoutHistoryHandler).Methods("GET")

	methodsExcluesGet := []string{"POST", "HEAD", "PUT", "DELETE", "CONNECT", "OPTIONS", "TRACE", "PATCH"}
	methodsExcluesPost := []string{"GET", "HEAD", "PUT", "DELETE", "CONNECT", "OPTIONS", "TRACE", "PATCH"}

	r.HandleFunc("/serverinfo", warnHandler).Methods(methodsExcluesGet...)
	r.HandleFunc("/statistics", warnHandler).Methods(methodsExcluesGet...)
	r.HandleFunc("/swapin/post/{txid}", warnHandler).Methods(methodsExcluesPost...)
	r.HandleFunc("/swapout/post/{txid}", warnHandler).Methods(methodsExcluesPost...)
	r.HandleFunc("/swapin/recall/{txid}", warnHandler).Methods(methodsExcluesPost...)
	r.HandleFunc("/swapin/{txid}", warnHandler).Methods(methodsExcluesGet...)
	r.HandleFunc("/swapout/{txid}", warnHandler).Methods(methodsExcluesGet...)
	r.HandleFunc("/swapin/{txid}/raw", warnHandler).Methods(methodsExcluesGet...)
	r.HandleFunc("/swapout/{txid}/raw", warnHandler).Methods(methodsExcluesGet...)
	r.HandleFunc("/swapin/{txid}/rawresult", warnHandler).Methods(methodsExcluesGet...)
	r.HandleFunc("/swapout/{txid}/rawresult", warnHandler).Methods(methodsExcluesGet...)
	r.HandleFunc("/swapin/history/{address}", warnHandler).Methods(methodsExcluesGet...)
	r.HandleFunc("/swapout/history/{address}", warnHandler).Methods(methodsExcluesGet...)

	apiPort := params.GetApiPort()
	apiServer := params.GetConfig().ApiServer
	allowedOrigins := apiServer.AllowedOrigins

	corsOptions := []handlers.CORSOption{
		handlers.AllowedMethods([]string{"GET", "POST"}),
	}
	if len(allowedOrigins) != 0 {
		corsOptions = append(corsOptions,
			handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type"}),
			handlers.AllowedOrigins(allowedOrigins),
		)
	}

	log.Info("JSON RPC service listen and serving", "port", apiPort, "allowedOrigins", allowedOrigins)
	svr := http.Server{
		Addr:         fmt.Sprintf(":%v", apiPort),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		Handler:      handlers.CORS(corsOptions...)(r),
	}
	go func() {
		if err := svr.ListenAndServe(); err != nil {
			log.Error("ListenAndServe error", "err", err)
		}
	}()
}

func warnHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, fmt.Sprintf("Forbid '%v' on '%v'", r.Method, r.RequestURI))
}
