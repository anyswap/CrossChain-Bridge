# crossChain-Bridge
cross chain bridge using DCRM technology

# Install the Docker version
## 1. Install Docker. This depends on your platform, on Ubuntu this works:
```
sudo apt update
sudo apt install docker.io
```
## 2. Download the Docker image and create and run the container:
### swap
```
docker run -itd --name swap --network host --restart always -v /var/lib/docker/swap:/swap anywap/swap
```
### client
```
docker exec -d swap swaporacle ...
```
tools: `swapserver swaporacle swapscan riskctrl swapadmin swaptools` (/usr/local/bin)  
conf: `config-example.toml config-tokenpair-example.toml` (/usr/local/bin)  

# Install the Source version
## Building

```shell
git clone https://github.com/anyswap/CrossChain-Bridge.git
cd crossChain-Bridge
make all
```

after building, the following files will be generated in `./build/bin` directory:

```text
swapserver	# server provide api service, and trigger swap processing
swaporacle      # oracle take part in dcrm signing (can disagree illegal transaction)
config-example.toml
config-tokenpair-example.toml
```

## Modify config file

copy the example config file `config-example.toml` in `./build/bin` directory, and modify it accordingly.

see more, please refer [config file example](https://github.com/anyswap/CrossChain-Bridge/blob/master/params/config-example.toml)

#### Identifier

Identifier should be a short string to identify the bridge (eg. `BTC2ETH`, `BTC2FSN`)

#### MongoDB

MongoDB is used by the server to store swap status and history, you should config according to your modgodb database setting.
(the swap oracle don't need it)

#### APIServer

APIServer is used by the server to provide API service to register swap and to provide history retrieving.
(the swap oracle don't need it)

#### Oracle

Oracle is needed by the swap oracle to post swap register RPC requests to swap server
(the swap server don't need `Oracle`).

#### BtcExtra

BtcExtra is used to customize fees when build transaction on Bitcoin blockchain

#### SrcChain

SrcChain is used to config the chain of source endpoint of the cross chain bridge.

#### SrcGateway

SrcGateway is used to do RPC request to verify transactions on source blockchain, and to broadcast signed transaction.

#### DestChain

DestChain is used to config the chain of dest endpoint of the cross chain bridge.

#### DestGateway

DestGateway is used to do RPC request to verify transactions on dest blockchain, and to broadcast signed transaction.

#### Dcrm

Dcrm is used to config DCRM node info and group info.

`Initiators` is used to specify the server dcrm user (initiators of dcrm sign)

`Dcrm.DefaultNode` is used to specify default dcrm node to connect.

`Dcrm.OtherNodes` is an array used by server to specify other initiators of dcrm node.

for the swap server, `SignGroups` is needed for dcrm signing.

Notice:
If in test enviroment you may run more than one program of swap servers on one machine,
Please specify `different log file name` to clarify the outputs.
And please specify `different config files` for each server,
and assgin `KeystoreFile`, `PasswordFile` and `RPCAddress` etc. separatly.


## Modify token pair config files

copy the example config file `config-tokenpair-example.toml` in `./build/bin` directory, and modify it accordingly.

see more, please refer [token pair config file example](https://github.com/anyswap/CrossChain-Bridge/blob/master/params/config-tokenpair-example.toml)

#### PairID

pair ID of this token pair, must be unique.

#### SrcToken

SrcToken is used to config the token of source endpoint of the cross chain bridge.

#### DestToken

DestToken is used to config the token of dest endpoint of the cross chain bridge.


## Run swap server

```shell
setsid ./build/bin/swapserver --verbosity 6 --config build/bin/config.toml --pairsdir build/bin/tokenpairs --log build/bin/logs/server.log
```

## Run swap oracle

```shell
setsid ./build/bin/swaporacle --verbosity 6 --config build/bin/config.toml --pairsdir build/bin/tokenpairs --log build/bin/logs/oracle.log
```

## Others

`swapserver` and `swaporacle` has the following subcommands:

```text
help       - to see hep info.
version - to show the version.
license - to show the license
```

## Preparations

Running  `swapserver` and `swaporacle` to provide cross chain bridge service, we must prepare the following things firstly and config them rightly. Otherwise the program will not run or run rightly. To ensure this, we have add many checkings to the config items.

For the config file, please refer [config file example](https://github.com/anyswap/CrossChain-Bridge/blob/master/params/config-example.toml)

1. create Mongodb database (shared by all swap servers of the bridge provider)

    config `[MongoDB]` section accordingly (eg. `DbURL`,`DbName`,`UserName`,`Password`)

    For security reason, we suggest:
    1. change the mongodb `port` ( defaults to `27017`)
    2. enable `auth`
    3. create user with passord to access the database

2. create DCRM group

    For example, we take creating `2/3 theshold` jonitly managed DCRM group as an example.

    we have 3 users in the DCRM group, each user is running a `gdcrm` node to perform DCRM functions.

    1. `user1` -  build raw tx and trigger DCRM signing
    2. `user2`, `user3` - verify tx and accept the DCRM signing with `AGREE/DISAGREE` result

    After created DCRM group,

    We can get the corresponding `DCRM addresses` on supported blockchains. Then we should config `DcrmAddress` in `[SrcToken]` and `[DestToken]` section according to the blockchain of them.

    We should config the `[Dcrm]` section accordingly（ eg. `GroupID`, `TotalOracles`，`Mode`, `DefaultNode`, etc.）

    And we should config the following `[Dcrm]` section items sparately for each user in the DCRM group:

    1. `KeystoreFile`
    2. `PasswordFile`
    3. `RPCAddress`

    For example,

    we are configing `user1` now, we should config `KeystoreFile` and `PasswordFile` use `user1`'s keystore and password file (we will get `user1`'s private key to sign a DCRM requesting).

    And we should config `RPCAddress` to the RPC address of the running `gdcrm` node of `user1` (we will do RPC calls to this address to complete DCRM signing or accepting, etc.)

3. create DCRM sub-groups for signing

    In the above step we have created a `2/3 threshold` DCRM group.

    In signing we only need 2 members to agree.

    So we prepared the sub-groups (`2/2`) for signing. (eg. user1 + user2, user1 + user3)
    please see more detail about DCRM [here](https://github.com/fsn-dev/dcrm-walletService)

    After created, we should config `SignGroups` in `[Dcrm]` accordingly.

4. create mapping asset (eg. mBTC) contract **`with DCRM account`**

    mBTC is an smart contract inherit from `ERC20` and add two methods: `Swapin` and `Swapout`.

    please see more here about [mBTC](https://github.com/anyswap/mBTC)

    After created mBTC, we should config `ContractAddress` in `[DestToken]` section.

5. config `[APIServer]` section

    The swap server provides RPC service to query swap status and swap history. etc.

    Please see more here about [crossChain-Bridge-API](https://github.com/anyswap/CrossChain-Bridge/wiki/crossChain-Bridge-API)

    We should config `Port` (defaults to `11556`), and `AllowedOrigins` (CORS defaults to empty array. `["*"]` for allow any)

6. config `[SrcChain]`,`[SrcGateway]`

    We should config `APIAddress` in `[SrcGateway]` section,
    to post RPC request to the running full node to get transaction, broadcat transaction etc.

    Config `[SrcChain]`, ref. to the following example:

    ```toml
    BlockChain = "Bitcoin" # required
    NetID = "TestNet3"  # required
    Confirmations = 0 # suggest >= 6 for Mainnet # required
    InitialHeight = 0
    EnableScan = false
    ```

7. config `[DestChain]`, `[DestGateway]`

    We should config `APIAddress` in `[DestGateway]` section,
    to post RPC request to the running full node to get transaction, broadcat transaction etc.

    Config `[DestChain]` like `[SrcChain]`.

8. config `Identifier` to identify your crosschain bridge

    This should be a short string to identify the bridge (eg. `BTC2ETH`, `BTC2FSN`)

    Different DCRM group can have same Identifier.

    It only matter if you want use a DCRM group to provide multiple crosschain bridge service, and config each bridge with a different identifier (`Notice: This way is not suggested`)

9. config `[BtcExtra]`

    When build and sign transaction on Bitcoin blockchain, we can customize the following items:

    ```toml
    MinRelayFee   = 400
    RelayFeePerKb = 2000
    UtxoAggregateMinCount = 20
    UtxoAggregateMinValue = 1000000
    ```

    If not configed, the default vlaue will be used (in fact, the above values are the defaults)

10. config `[SrcToken]`

    Config `[SrcToken]`, ref. to the following example:

    ```toml
    ID = ""
    Name = "Bitcoin Coin"
    Symbol = "BTC"
    Decimals = 8  # required
    Description = "Bitcoin Coin"
    ContractAddress = ""
    DepositAddress = "mq6XaNvFWiSJtfGYiGakkRdXNrqH6V4Jpu" # required
    DcrmAddress = "mfwPnCuht2b4Lvb5XTds4Rvzy3jZ2ZWrBL" # required
    DcrmPubkey = "045c8648793e4867af465691685000ae841dccab0b011283139d2eae454b569d5789f01632e13a75a5aad8480140e895dd671cae3639f935750bea7ae4b5a25122"
    MaximumSwap = 1000.0 # required
    MinimumSwap = 0.00001 # required
    SwapFeeRate = 0.001 # required
    MaximumSwapFee = 0.01
    MinimumSwapFee = 0.00001
    PlusGasPricePercentage = 15
    BigValueThreshold = 5.0
    DisableSwap = false
    ```

    For ERC20 token, we should config `ID = "ERC20"` and `ContractAddress` to the token's contract address.

11. config `[DestToken]`

    Config `[DestToken]` like `[SrcToken]`.

    Don't forget to config  `ContractAddress` in `[DestToken]` section  (see step 4)

12. config `PairID` to identify token pair

```text
repeat step 10, 11, 12 to prepare multiple token pairs config,
put them in a directory (will be referenced by --pairsdir command line option)
```
