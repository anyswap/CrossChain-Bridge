# crossChain-Bridge
cross chain bridge using DCRM technology

## Building

```shell
git clone https://github.com/fsn-dev/crossChain-Bridge.git
cd crossChain-Bridge
make all
```

after building, the following 3 files will be generated in `./build/bin` directory:

```text
swapserver	# server provide api service, and trigger swap processing
swaporacle	# oracle take part in dcrm signing (can disagree illegal transaction)
config.toml
```

## Modify config.toml

modify the example `config.toml` in `./build/bin` directory

see more, please refer [config file example](https://github.com/fsn-dev/crossChain-Bridge/blob/master/params/config.toml)

#### MongoDB

MongoDB is used by the server to store swap status and history, you should config according to your modgodb database setting (the swap oracle don't need it).

#### ApiServer

ApiServer is used by the server to provide API service to register swap and to provide history retrieving (the swap oracle don't need it).

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

for the swap server, `Pubkey` and `SignGroups` is needed for dcrm signing (the swap oracle don't need it).

#### Oracle

Oracle is needed by the swap oracle (the swap server don't need it).

Notice:
If in test enviroment you may run more than one program of swap server and oracles on the same machine,
please specify `different log file name` to clarify the outputs.
And please specify `different config files` to specify `KeystoreFile`, `PasswordFile` and `RpcAddress` etc. separatly.

## Run swap server

```shell
setsid ./build/bin/swapserver -v 6 -c build/bin/config.toml --log build/bin/logs/server.log
```

## Run swap oracle

```shell
setsid ./build/bin/swaporacle -v 6 -c build/bin/config.toml --log build/bin/logs/oracle.log
```

## Others

both the `swapserver` and `swaporacle` provide the following subcommands:

```text
help       - to see hep info.
version - to show the version.
license - to show the license
```

## Preparations

Running  `swapserver` and `swaporacle` to provide cross chain bridge service, we must prepare the following things firstly and config them rightly. Otherwise the program will not run or run rightly. To ensure this, we have add many checkings to the config items.

For the config file, please refer [config file example](https://github.com/fsn-dev/crossChain-Bridge/blob/master/params/config.toml)

1. create Mongodb database (**`only swapserver need`**)

    config `[MongoDB]` section accordingly (eg. `DbURL`,`DbName`,`UserName`,`Password`)

    For security reason, we suggest:
    1. change the mongod `port` ( deafults to `27017`)
    2. enable `auth`
    3. forbid remote connection.
    4. create user with passord to access the database

2. create DCRM group

    For example, we take creating `2/3 theshold` jonitly managed DCRM group as an example.

    we have 3 users in the DCRM group, each user is running a `gdcrm` node to perform DCRM functions.

    1. `user1` is the swap server 

        build raw tx and trigger DCRM signing

    2. `user2`, `user3` is the swap oracles
        
        verify tx and accept the DCRM signing with `AGREE/DISAGREE` result

    After created DCRM group,  
    
    We can get the corresponding `DCRM addresses` on supported blockchains. Then we should config `DcrmAddress` in `[SrcToken]` and `[DestToken]` section according to the blockchain of them.
    
    We should config the `[Dcrm]` section accordingly（ eg. `GroupID`, `Pubkey`，`NeededOracles`， `TotalOracles`，`Mode`）

    And we should config the following `[Dcrm]` section items sparately for each user in the DCRM group:

    1. `KeystoreFile`
    2. `PasswordFile`
    3. `RpcAddress`

    For example, 
    
    we are configing `user1` now, we should config `KeystoreFile` and `PasswordFile` use `user1`'s keystore and password file (we will get `user1`'s private key to sign a DCRM requesting). 

    And we should config `RpcAddress` to the RPC address of the running `gdcrm` node of `user1` (we will do RPC calls to this address to complete DCRM signing or accepting, etc.)

3. create DCRM sub-groups for signing (**`only swapserver need`**)

    In the above step we have created a `2/3 threshold` DCRM group. 
    
    In signing we only need 2 members to agree. 
    
    So we prepared the sub-groups for signing. (please see more detail about DCRM [here](https://github.com/fsn-dev/dcrm-walletService))

    After created, we should config `SignGroups` in `[Dcrm]` accordingly.

4. create mapping asset (eg. mBTC) contract **`with DCRM account`**

    mBTC is an smart contract inherit from `ERC20` and add two methods: `Swapin` and `Swapout`.

    please see more here about [mBTC](https://github.com/fsn-dev/mBTC)

    After created mBTC, we should config `ContractAddress` in `[DestToken]` section.

5. config `[ApiServer]` section (**`only swapserver need`**)

    The swap server provides RPC service to query swap status and swap history. etc. Please see more here about [crossChain-Bridge-API](https://github.com/fsn-dev/crossChain-Bridge/wiki/crossChain-Bridge-API)

    We should config `Port` (defaults to `11556`), and `AllowedOrigins` (deafults to empty array)

6. config `[SrcToken]`,`[SrcGateway]`

    We should config `ApiAddress` in `[SrcGateway]` section, to post RPC request to the running full node to get transaction, broadcat transaction etc.

    Config `[SrcToken]`, ref. to the following example:

    ```toml
    BlockChain = "Bitcoin" # required
    NetID = "TestNet3"  # required
    Name = "Bitcoin Coin"
    Symbol = "BTC"
    Decimals = 8  # required
    Description = "Bitcoin Coin"
    DcrmAddress = "mfwPnCuht2b4Lvb5XTds4Rvzy3jZ2ZWrBL" # required
    Confirmations = 0 # suggest >= 6 for Mainnet # required
    MaximumSwap = 1000.0 # required
    MinimumSwap = 0.00001 # required
    SwapFeeRate = 0.001 # required
    ```

7. config `[DestToken]`, `[DestGateway]` 

    we should config `ApiAddress` in `[DestGateway]` section, to post RPC request to the running full node to get transaction, broadcat transaction etc.

    Config `[DestToken]` like `[SrcToken]`. 
    
    Don't forget to config  `ContractAddress` in `[DestToken]` section  (see step 4)

8. config `Identifier` to identify your crosschain bridge

    This should be a short string to identify the bridge (eg. `BTC2ETH`, `BTC2FSN`)
    
    Different DCRM group can have same Identifier. 
    
    It only matter if you want use a DCRM group to provide multiple crosschain bridge service, and config each bridge with a different identifier (`Notice: This way is not suggested`)

9. config `[Oracle]` (**`only swaporacle need`**)

    For Oracle, It should config `ServerApiAddress` to post swap registers the oracle finds.

10. config `[BtcExtra]` (**`only swapserver need`**)

    When build and sign transaction on Bitcoin blockchain, we can specify the following items:

    ```toml
    MinRelayFee   = 400
    RelayFeePerKb = 2000
    FromPublicKey = ""
    ```

    If not configed, the default vlaue will be used (in fact, the above values are the defaults)
