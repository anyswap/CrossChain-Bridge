// Package server provides JSON/RESTful RPC service.
package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/didip/tollbooth/v6"
	"github.com/didip/tollbooth/v6/limiter"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/rpc/v2"
	rpcjson "github.com/gorilla/rpc/v2/json2"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/rpc/restapi"
	"github.com/anyswap/CrossChain-Bridge/rpc/rpcapi"
)

// StartAPIServer start api server
func StartAPIServer() {
	router := mux.NewRouter()
	initRouter(router)

	apiPort := params.GetAPIPort()
	apiServer := params.GetServerConfig().APIServer
	allowedOrigins := apiServer.AllowedOrigins
	maxRequestsLimit := apiServer.MaxRequestsLimit
	if maxRequestsLimit <= 0 {
		maxRequestsLimit = 10 // default value
	}

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
	lmt := tollbooth.NewLimiter(float64(maxRequestsLimit),
		&limiter.ExpirableOptions{
			DefaultExpirationTTL: 600 * time.Second,
		},
	)
	handler := tollbooth.LimitHandler(lmt, handlers.CORS(corsOptions...)(router))
	svr := http.Server{
		Addr:         fmt.Sprintf(":%v", apiPort),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 300 * time.Second,
		Handler:      handler,
	}
	go func() {
		if err := svr.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed && utils.IsCleanuping() {
				return
			}
			log.Fatal("ListenAndServe error", "err", err)
		}
	}()

	utils.TopWaitGroup.Add(1)
	go utils.WaitAndCleanup(func() { doCleanup(&svr) })
}

func doCleanup(svr *http.Server) {
	defer utils.TopWaitGroup.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := svr.Shutdown(ctx); err != nil {
		log.Error("Server Shutdown failed", "err", err)
	}
	log.Info("Close http server success")
}

// nolint:funlen // put together handle func
func initRouter(r *mux.Router) {
	rpcserver := rpc.NewServer()
	rpcserver.RegisterCodec(rpcjson.NewCodec(), "application/json")
	err := rpcserver.RegisterService(new(rpcapi.RPCAPI), "swap")
	if err != nil {
		log.Fatal("start rpc service failed", "err", err)
	}

	r.Handle("/rpc", rpcserver)

	r.HandleFunc("/serverinfo", restapi.ServerInfoHandler).Methods("GET")
	r.HandleFunc("/versioninfo", restapi.VersionInfoHandler).Methods("GET")
	r.HandleFunc("/oracleinfo", restapi.OracleInfoHandler).Methods("GET")
	r.HandleFunc("/nonceinfo", restapi.NonceInfoHandler).Methods("GET")
	r.HandleFunc("/pairinfo/{pairid}", restapi.TokenPairInfoHandler).Methods("GET")
	r.HandleFunc("/pairsinfo/{pairids}", restapi.TokenPairsInfoHandler).Methods("GET")

	r.HandleFunc("/swapin/post/{pairid}/{txid}", restapi.PostSwapinHandler).Methods("POST")
	r.HandleFunc("/swapout/post/{pairid}/{txid}", restapi.PostSwapoutHandler).Methods("POST")
	r.HandleFunc("/swapin/p2sh/{txid}/{bind}", restapi.PostP2shSwapinHandler).Methods("POST")
	r.HandleFunc("/swapin/retry/{pairid}/{txid}", restapi.RetrySwapinHandler).Methods("POST")

	r.HandleFunc("/swapin/{pairid}/{txid}", restapi.GetSwapinHandler).Methods("GET")
	r.HandleFunc("/swapout/{pairid}/{txid}", restapi.GetSwapoutHandler).Methods("GET")
	r.HandleFunc("/swapin/{pairid}/{txid}/raw", restapi.GetRawSwapinHandler).Methods("GET")
	r.HandleFunc("/swapout/{pairid}/{txid}/raw", restapi.GetRawSwapoutHandler).Methods("GET")
	r.HandleFunc("/swapin/{pairid}/{txid}/rawresult", restapi.GetRawSwapinResultHandler).Methods("GET")
	r.HandleFunc("/swapout/{pairid}/{txid}/rawresult", restapi.GetRawSwapoutResultHandler).Methods("GET")
	r.HandleFunc("/swapin/history/{pairid}/{address}", restapi.SwapinHistoryHandler).Methods("GET")
	r.HandleFunc("/swapout/history/{pairid}/{address}", restapi.SwapoutHistoryHandler).Methods("GET")

	r.HandleFunc("/p2sh/{address}", restapi.GetP2shAddressInfo).Methods("GET")
	r.HandleFunc("/p2sh/bind/{address}", restapi.RegisterP2shAddress).Methods("POST")

	r.HandleFunc("/registered/{address}", restapi.GetRegisteredAddress).Methods("GET")
	r.HandleFunc("/register/{address}", restapi.RegisterAddress).Methods("POST")
}
