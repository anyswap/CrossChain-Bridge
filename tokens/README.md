# How to add bridge support of new blockchain

## 1. create a new directory for each block chain

for example, `btc`,  `eth`,  `fsn`  etc.

## 2. implement methods in `CrossChainBridge` interface

```golang
IsSrcEndpoint() bool
```
`IsSrcEndpoint` returns `true` if this bridge is on the `source` block chain, otherwise returns `false`.

------

```golang
SetChainAndGateway(*ChainConfig, *GatewayConfig)
```
`SetChainAndGateway` set chain and gateway config.

------

```golang
GetChainConfig() *ChainConfig
```
`GetChainConfig` get chain config.

------

```golang
GetGatewayConfig() *GatewayConfig
```
`GetGatewayConfig` get gateway config.

------

```golang
GetTokenConfig(pairID string) *TokenConfig
```
`GetTokenConfig` get token config.

------

```golang
VerifyTokenConfig(*TokenConfig) error
```
`VerifyTokenConfig` verify token config.

------

```golang
IsValidAddress(address string) bool
```
`IsValidAddress` check if given address is valid.

------

```golang
GetTransaction(txHash string) (interface{}, error)
```
`GetTransaction` get transaction by hash.

------

```golang
GetTransactionStatus(txHash string) *TxStatus
```
`GetTransactionStatus` get transaction status by hash.

------

```golang
VerifyTransaction(pairID, txHash string, allowUnstable bool) (*TxSwapInfo, error)
```
`VerifyTransaction` verify transaction by hash.

------

```golang
VerifyMsgHash(rawTx interface{}, msgHash []string) error
```
`VerifyMsgHash` verify message hash of rawtx in `DCRM` signing.

------

```golang
BuildRawTransaction(args *BuildTxArgs) (rawTx interface{}, err error)
```
`BuildRawTransaction` build raw transaction for swapin/swapout.

------

```golang
SignTransaction(rawTx interface{}, pairID string) (signedTx interface{}, txHash string, err error)
```
`SignTransaction` sign transaction with configed private key.

------

```golang
DcrmSignTransaction(rawTx interface{}, args *BuildTxArgs) (signedTx interface{}, txHash string, err error)
```
`DcrmSignTransaction` sign transaction in `DCRM` way.

------

```golang
SendTransaction(signedTx interface{}) (txHash string, err error)
```
`SendTransaction` send/broadcast signed transaction.

------

```golang
GetLatestBlockNumber() (uint64, error)
```
`GetLatestBlockNumber` get the latest block number.

------

```golang
GetLatestBlockNumberOf(apiAddress string) (uint64, error)
```
`GetLatestBlockNumberOf` get the latest block number by connecting to specified full node RPC address.

------

```golang
StartChainTransactionScanJob()
```
`StartChainTransactionScanJob` start scan chain transaction and auto register found swapin or swapout of this bridge.

------

```golang
StartPoolTransactionScanJob()
```
`StartPoolTransactionScanJob` start scan pool transaction and auto register found swapin or swapout of this bridge.

------

```golang
GetBalance(accountAddress string) (*big.Int, error)
```
`GetBalance` get coin balance of given account address.

------

```golang
GetTokenBalance(tokenType, tokenAddress, accountAddress string) (*big.Int, error)
```
`GetTokenBalance` get token balance of given token and account.

------

```golang
GetTokenSupply(tokenType, tokenAddress string) (*big.Int, error)
```
`GetTokenSupply` get token total supply of give token.

------

## 3. other possible way

Because some chain are forked from already implemented blockchain, we can derive from this implemented bridge and update some interface implement.

For example, `fsn` is forked from `eth`. So we derive `fsn` bridge from `eth` bridge and update some chain verify, and then `fsn` is also supported now as it has implememted all the required methods in `CrossChainBridge` interface.
