module github.com/gaozhengxin/CrossChain-Bridge/tokens/tron/trondev

go 1.15

replace github.com/anyswap/CrossChain-Bridge => ../../..

require (
	github.com/anyswap/CrossChain-Bridge v0.0.0-00010101000000-000000000000
	github.com/fbsobreira/gotron-sdk v0.0.0-20210316163828-8cb47d581197
	github.com/golang/protobuf v1.5.2
	google.golang.org/grpc v1.37.0
)
