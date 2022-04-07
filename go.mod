module github.com/anyswap/CrossChain-Bridge

go 1.14

require (
	github.com/BurntSushi/toml v0.4.1
	github.com/bits-and-blooms/bitset v1.2.2
	github.com/btcsuite/btcd v0.21.0-beta
	github.com/btcsuite/btcutil v1.0.3-0.20201208143702-a53e38424cce
	github.com/btcsuite/btcwallet/wallet/txauthor v1.0.0
	github.com/btcsuite/btcwallet/wallet/txrules v1.0.0
	github.com/btcsuite/btcwallet/wallet/txsizes v1.0.0
	github.com/cosmos/cosmos-sdk v0.39.2
	github.com/deckarep/golang-set v1.7.1
	github.com/didip/tollbooth/v6 v6.1.1
	github.com/fastly/go-utils v0.0.0-20180712184237-d95a45783239 // indirect
	github.com/fatih/color v1.7.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/giangnamnabka/btcd v0.21.0-beta.0.20210425211148-6b9fe474f6d7
	github.com/giangnamnabka/btcutil v1.0.3-0.20210422183801-1e4dd2b8f1ba
	github.com/giangnamnabka/btcwallet/wallet/txauthor v1.0.1-0.20210422185028-3bf95367dc20
	github.com/giangnamnabka/btcwallet/wallet/txrules v1.0.1-0.20210422185028-3bf95367dc20
	github.com/giangnamnabka/btcwallet/wallet/txsizes v1.0.1-0.20210422185028-3bf95367dc20
	github.com/go-resty/resty/v2 v2.6.0
	github.com/golang/snappy v0.0.3-0.20201103224600-674baa8c7fc3 // indirect
	github.com/google/gofuzz v1.1.1-0.20200604201612-c04b05f3adfa // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/rpc v1.2.0
	github.com/gorilla/websocket v1.4.2
	github.com/jehiah/go-strftime v0.0.0-20171201141054-1d33003b3869 // indirect
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/jordan-wright/email v0.0.0-20200917010138-e1c00e156980
	github.com/juju/loggo v0.0.0-20210728185423-eebad3a902c4 // indirect
	github.com/juju/testing v0.0.0-20190723135506-ce30eb24acd2
	github.com/kr/pretty v0.2.0 // indirect
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible
	github.com/lestrrat-go/strftime v1.0.3 // indirect
	github.com/ltcsuite/ltcd v0.20.1-beta
	github.com/ltcsuite/ltcutil v1.0.2-beta
	github.com/ltcsuite/ltcwallet/wallet/txauthor v1.0.0
	github.com/ltcsuite/ltcwallet/wallet/txrules v1.0.0
	github.com/ltcsuite/ltcwallet/wallet/txsizes v1.0.0
	github.com/pborman/uuid v1.2.1
	github.com/prometheus/procfs v0.0.10 // indirect
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cobra v1.1.1 // indirect
	github.com/spf13/viper v1.7.1 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20200815110645-5c35d600f0ca
	github.com/tebeka/strftime v0.1.5 // indirect
	github.com/tendermint/go-amino v0.16.0
	github.com/tendermint/tendermint v0.33.9
	github.com/tendermint/tm-db v0.5.2 // indirect
	github.com/terra-project/core v0.4.2
	github.com/tidwall/pretty v1.0.2 // indirect
	github.com/urfave/cli/v2 v2.3.0
	go.mongodb.org/mongo-driver v1.7.4
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
)

replace github.com/gogo/protobuf v1.3.3 => github.com/regen-network/protobuf v1.3.3-alpha.regen.1

replace github.com/tendermint/tendermint => github.com/tendermint/tendermint v0.33.9

replace github.com/cosmos/cosmos-sdk => github.com/cosmos/cosmos-sdk v0.39.2 // indirect
