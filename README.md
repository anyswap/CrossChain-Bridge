# crossChain-Bridge
cross chain bridge using DCRM technology

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
config.toml
```

## Modify config.toml

modify the example `config.toml` in `./build/bin` directory

see more, please refer [config file example](https://github.com/anyswap/CrossChain-Bridge/blob/master/params/config.toml)

### Identifier

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

### BtcExtra

BtcExtra is used to customize fees when build transaction on Bitcoin blockchain

#### SrcToken

SrcToken is used to config the source endpoint of the cross chain bridge.

#### SrcGateway

SrcGateway is used to do RPC request to verify transactions on source blockchain, and to broadcast signed transaction.

#### DestToken

DestToken is used to config the dest endpoint of the cross chain bridge.

#### DestGateway

DestGateway is used to do RPC request to verify transactions on dest blockchain, and to broadcast signed transaction.

#### Dcrm

Dcrm is used to config DCRM node info and group info.

`ServerAccount` is used to specify the server dcrm user (initiator of dcrm sign)

for the swap server, `Pubkey` and `SignGroups` is needed for dcrm signing.

Notice:
If in test enviroment you may run more than one program of swap servers on one machine,
Please specify `different log file name` to clarify the outputs.
And please specify `different config files` for each server,
and assgin `KeystoreFile`, `PasswordFile` and `RPCAddress` etc. separatly.

## Run swap server

```shell
setsid ./build/bin/swapserver --verbosity 6 --config build/bin/config.toml --log build/bin/logs/server.log
```

## Run swap oracle

```shell
setsid ./build/bin/swaporacle --verbosity 6 --config build/bin/config.toml --log build/bin/logs/oracle.log
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

For the config file, please refer [config file example](https://github.com/anyswap/CrossChain-Bridge/blob/master/params/config.toml)

1. create Mongodb database (shared by all swap servers of the bridge provider)

    config `[MongoDB]` section accordingly (eg. `DbURL`,`DbName`,`UserName`,`Password`)

    For security reason, we suggest:
    1. change the mongod `port` ( deafults to `27017`)
    2. enable `auth`
    3. create user with passord to access the database

2. create DCRM group

    For example, we take creating `2/3 theshold` jonitly managed DCRM group as an example.

    we have 3 users in the DCRM group, each user is running a `gdcrm` node to perform DCRM functions.

    1. `user1` -  build raw tx and trigger DCRM signing
    2. `user2`, `user3` - verify tx and accept the DCRM signing with `AGREE/DISAGREE` result

    After created DCRM group,

    We can get the corresponding `DCRM addresses` on supported blockchains. Then we should config `DcrmAddress` in `[SrcToken]` and `[DestToken]` section according to the blockchain of them.

    We should config the `[Dcrm]` section accordingly（ eg. `ServerAccount`, `GroupID`, `Pubkey`，`NeededOracles`， `TotalOracles`，`Mode`）

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

6. config `[SrcToken]`,`[SrcGateway]`

    We should config `APIAddress` in `[SrcGateway]` section,
    to post RPC request to the running full node to get transaction, broadcat transaction etc.

    Config `[SrcToken]`, ref. to the following example:

    ```toml
    BlockChain = "Bitcoin" # required
    NetID = "TestNet3"  # required
    ID = ""
    Name = "Bitcoin Coin"
    Symbol = "BTC"
    Decimals = 8  # required
    Description = "Bitcoin Coin"
    ContractAddress = ""
    DcrmAddress = "mfwPnCuht2b4Lvb5XTds4Rvzy3jZ2ZWrBL" # required
    Confirmations = 0 # suggest >= 6 for Mainnet # required
    MaximumSwap = 1000.0 # required
    MinimumSwap = 0.00001 # required
    SwapFeeRate = 0.001 # required
    ```

    For ERC20 token, we should config `ID = "ERC20"` and `ContractAddress` to the token's contract address.

7. config `[DestToken]`, `[DestGateway]`

    We should config `APIAddress` in `[DestGateway]` section,
    to post RPC request to the running full node to get transaction, broadcat transaction etc.

    Config `[DestToken]` like `[SrcToken]`.

    Don't forget to config  `ContractAddress` in `[DestToken]` section  (see step 4)

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
