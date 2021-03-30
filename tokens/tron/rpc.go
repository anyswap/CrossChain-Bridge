package tron

import (
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/client"

	"github.com/anyswap/CrossChain-Bridge/log"
)

var GRPC_TIMEOUT = time.Second * 15

func (b *Bridge) getClients() []*client.GrpcClient {
	endpoints := b.GatewayConfig.APIAddress
	clis = make([]*client.GrpcClient, 0)
	for _, endpoint := range endpoints {
		cli := client.NewGrpcClientWithTimeout(endpoint, GRPC_TIMEOUT)
		if cli != nil {
			clis = append(clis, cli)
		}
	}
}

type RPCError struct {
	errs   []error
	method string
}

func (e *RPCError) log(msg error) {
	log.Warn("[Solana RPC error]", "method", e.method, "msg", msg)
	if len(e.errs) < 1 {
		e.errs = make([]error, 1)
	}
	e.errs = append(e.errs, msg)
}

func (e *RPCError) Error() error {
	return fmt.Errorf("[Solana RPC error] method: %v errors:%+v", e.method, e.errs)
}

// GetLatestBlockNumber returns current finalized block height
func (b *Bridge) GetLatestBlockNumber() (height uint64, err error) {
	rpcError := &RPCError{[]error{}, "GetLatestBlockNumber"}
	for _, cli := range b.getClients() {
		res, err := cli.GetNowBlock()
		if err == nil {
			if res.BlockHeader.RawData.Number > 0 {
				return uint64(res.BlockHeader.RawData.Number), nil
			}
		} else {
			rpcError.log(err)
		}
	}
	return 0, rpcError.Error()
}
