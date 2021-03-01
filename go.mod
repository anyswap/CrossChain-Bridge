module github.com/anyswap/CrossChain-Bridge

go 1.14

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/btcsuite/btcd v0.21.0-beta
	github.com/btcsuite/btcutil v1.0.2
	github.com/btcsuite/btcwallet/wallet/txauthor v1.0.0
	github.com/btcsuite/btcwallet/wallet/txrules v1.0.0
	github.com/btcsuite/btcwallet/wallet/txsizes v1.0.0
	github.com/cosmos/cosmos-sdk v0.39.2
	github.com/fastly/go-utils v0.0.0-20180712184237-d95a45783239 // indirect
	github.com/fsn-dev/fsn-go-sdk v0.0.0-20201127063150-d66d045799f9
	github.com/fsnotify/fsnotify v1.4.9
	github.com/go-resty/resty/v2 v2.5.0
	github.com/golang/snappy v0.0.3-0.20201103224600-674baa8c7fc3 // indirect
	github.com/google/gofuzz v1.1.1-0.20200604201612-c04b05f3adfa // indirect
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/rpc v1.2.0
	github.com/jehiah/go-strftime v0.0.0-20171201141054-1d33003b3869 // indirect
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/jordan-wright/email v0.0.0-20200917010138-e1c00e156980
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible
	github.com/lestrrat-go/strftime v1.0.3 // indirect
	github.com/ltcsuite/ltcd v0.20.1-beta
	github.com/ltcsuite/ltcutil v1.0.2-beta
	github.com/ltcsuite/ltcwallet/wallet/txauthor v1.0.0
	github.com/ltcsuite/ltcwallet/wallet/txrules v1.0.0
	github.com/ltcsuite/ltcwallet/wallet/txsizes v1.0.0
	github.com/mattn/go-runewidth v0.0.4 // indirect
	github.com/pborman/uuid v1.2.1
	github.com/peterh/liner v1.1.1-0.20190123174540-a2c9a5303de7 // indirect
	github.com/shopspring/decimal v1.2.0
	github.com/sirupsen/logrus v1.7.0
	github.com/stretchr/testify v1.7.0
	github.com/tebeka/strftime v0.1.5 // indirect
	github.com/tendermint/go-amino v0.16.0
	github.com/tendermint/tendermint v0.33.9
	github.com/terra-project/core v0.4.2
	github.com/urfave/cli/v2 v2.3.0
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22
)

replace github.com/gogo/protobuf v1.3.3 => github.com/regen-network/protobuf v1.3.3-alpha.regen.1

replace github.com/tendermint/tendermin => github.com/tendermint/tendermint v0.33.9

replace github.com/cosmos/cosmos-sdk => github.com/cosmos/cosmos-sdk v0.39.2 // indirect
