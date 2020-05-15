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
