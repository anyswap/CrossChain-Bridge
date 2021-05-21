module github.com/anyswap/CrossChain-Bridge

go 1.14

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/btcsuite/btcd v0.21.0-beta
	github.com/btcsuite/btcutil v1.0.2
	github.com/btcsuite/btcwallet/wallet/txauthor v1.0.0
	github.com/btcsuite/btcwallet/wallet/txrules v1.0.0
	github.com/btcsuite/btcwallet/wallet/txsizes v1.0.0
	github.com/fsn-dev/fsn-go-sdk v0.0.0-20210430081410-a6b17c99c3ea
	github.com/fsnotify/fsnotify v1.4.9
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/rpc v1.2.0
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/jordan-wright/email v0.0.0-20200917010138-e1c00e156980
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible
	github.com/lestrrat-go/strftime v1.0.3 // indirect
	github.com/ltcsuite/ltcd v0.20.1-beta
	github.com/ltcsuite/ltcutil v1.0.2-beta
	github.com/ltcsuite/ltcwallet/wallet/txauthor v1.0.0
	github.com/ltcsuite/ltcwallet/wallet/txrules v1.0.0
	github.com/ltcsuite/ltcwallet/wallet/txsizes v1.0.0
	github.com/okex/exchain v0.18.6
	github.com/pborman/uuid v1.2.1
	github.com/shopspring/decimal v1.2.0
	github.com/sirupsen/logrus v1.7.0
	github.com/stretchr/testify v1.7.0
	github.com/tebeka/strftime v0.1.5 // indirect
	github.com/urfave/cli/v2 v2.3.0
	golang.org/x/crypto v0.0.0-20201203163018-be400aefbc4c
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22
)

replace (
	github.com/cosmos/cosmos-sdk => github.com/okex/cosmos-sdk v0.39.2-exchain5
	github.com/tendermint/tendermint => github.com/okex/tendermint v0.33.9-exchain4
)
