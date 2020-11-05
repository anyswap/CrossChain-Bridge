package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/rpc/v2"
	rpcjson "github.com/gorilla/rpc/v2/json2"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/rpc/restapi"
	"github.com/anyswap/CrossChain-Bridge/rpc/rpcapi"
)

// StartAPIServer start api server
func StartAPIServer() {
	router := initRouter()

	apiPort := params.GetAPIPort()
	apiServer := params.GetConfig().APIServer
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
		Handler:      handlers.CORS(corsOptions...)(router),
	}
	go func() {
		if err := svr.ListenAndServe(); err != nil {
			log.Error("ListenAndServe error", "err", err)
		}
	}()
}

func initRouter() *mux.Router {
	r := mux.NewRouter()

	rpcserver := rpc.NewServer()
	rpcserver.RegisterCodec(rpcjson.NewCodec(), "application/json")
	_ = rpcserver.RegisterService(new(rpcapi.RPCAPI), "swap")

	r.Handle("/rpc", rpcserver)

	registerHandleFunc(r, "/serverinfo", restapi.ServerInfoHandler, "GET")
	registerHandleFunc(r, "/versioninfo", restapi.VersionInfoHandler, "GET")
	registerHandleFunc(r, "/pairinfo/{pairid}", restapi.TokenPairInfoHandler, "GET")
	registerHandleFunc(r, "/statistics/{pairid}", restapi.StatisticsHandler, "GET")

	registerHandleFunc(r, "/swapin/post/{pairid}/{txid}", restapi.PostSwapinHandler, "POST")
	registerHandleFunc(r, "/swapout/post/{pairid}/{txid}", restapi.PostSwapoutHandler, "POST")
	registerHandleFunc(r, "/swapin/p2sh/{txid}/{bind}", restapi.PostP2shSwapinHandler, "POST")
	registerHandleFunc(r, "/swapin/retry/{pairid}/{txid}", restapi.RetrySwapinHandler, "POST")

	registerHandleFunc(r, "/swapin/{pairid}/{txid}", restapi.GetSwapinHandler, "GET")
	registerHandleFunc(r, "/swapout/{pairid}/{txid}", restapi.GetSwapoutHandler, "GET")
	registerHandleFunc(r, "/swapin/{pairid}/{txid}/raw", restapi.GetRawSwapinHandler, "GET")
	registerHandleFunc(r, "/swapout/{pairid}/{txid}/raw", restapi.GetRawSwapoutHandler, "GET")
	registerHandleFunc(r, "/swapin/{pairid}/{txid}/rawresult", restapi.GetRawSwapinResultHandler, "GET")
	registerHandleFunc(r, "/swapout/{pairid}/{txid}/rawresult", restapi.GetRawSwapoutResultHandler, "GET")
	registerHandleFunc(r, "/swapin/history/{pairid}/{address}", restapi.SwapinHistoryHandler, "GET")
	registerHandleFunc(r, "/swapout/history/{pairid}/{address}", restapi.SwapoutHistoryHandler, "GET")

	registerHandleFunc(r, "/p2sh/{address}", restapi.GetP2shAddressInfo, "GET")
	registerHandleFunc(r, "/p2sh/bind/{address}", restapi.RegisterP2shAddress, "POST")

	registerHandleFunc(r, "/bip32/{address}/{pairid}", restapi.GetBip32AddressInfo, "GET")
	registerHandleFunc(r, "/registered/{address}/{pairid}", restapi.GetRegisteredAddress, "GET")
	registerHandleFunc(r, "/register/{address}/{pairid}", restapi.RegisterAddress, "POST")

	return r
}

type handleFuncType = func(w http.ResponseWriter, r *http.Request)

func registerHandleFunc(r *mux.Router, path string, handler handleFuncType, methods ...string) {
	for i := 0; i < len(methods); i++ {
		methods[i] = strings.ToUpper(methods[i])
	}
	isAcceptMethod := func(method string) bool {
		for _, acceptMethod := range methods {
			if method == acceptMethod {
				return true
			}
		}
		return false
	}
	allMethods := []string{"GET", "POST", "HEAD", "PUT", "DELETE", "CONNECT", "OPTIONS", "TRACE", "PATCH"}
	excludedMethods := make([]string, 0, len(allMethods))
	for _, method := range allMethods {
		if !isAcceptMethod(method) {
			excludedMethods = append(excludedMethods, method)
		}
	}
	if len(methods) > 0 {
		acceptMethods := strings.Join(methods, ",")
		r.HandleFunc(path, handler).Methods(acceptMethods)
	}
	if len(excludedMethods) > 0 {
		forbidMethods := strings.Join(excludedMethods, ",")
		r.HandleFunc(path, warnHandler).Methods(forbidMethods)
	}
}

func warnHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Forbid '%v' on '%v'\n", r.Method, r.RequestURI)
}
