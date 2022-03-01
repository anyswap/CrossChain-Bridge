# How to test Non-EVM chain bridge

One of the difficulty in the development and debug of the non-evm chain bridge is that how to perform verification and threshold signatures on the MPC network.

We provide a private test mode, by replacing the complexity of the mpc network test environment with private key signature, developer can focus on the api test of non-evm chain.

## Private testmode support

### 1. Change the chain and token config

First, config the private test mode with private key, copy the config examples in `tokens/tests` directory, change the `DcrmAddressPriKey` to your test private key(Only for test, don't use the key which has real fund).

chain A: non-evm src chain

chain B: evm dest chain

```text
~/temp/bridge-test/
├── bridge-nonevm2eth.toml
└── tokenpairs
    └── bridge-nonevm2eth-USDT.toml
```

### 2. Start the bridge

```shell
./build/bin/swapserver --config ~/temp/bridge-test/bridge-nonevm2eth.toml --pairsdir ~/temp/bridge-test/tokenpairs
```

### 3. Trigger the bridge test

1. Bridge USDT token from chain A to chain B

```shell
# format "/swap/test/{swaptype}/{pairid}/{txid}"
curl -sS http://127.0.0.1:11556/swap/test/swapin/USDT/c241bcd7176aef7c0f4bc331c55c7e282f03aed663eef7b55705b0bd67d402b8
```

2. Bridge USDT token from chain B to chain A

```shell
# format "/swap/test/{swaptype}/{pairid}/{txid}"
curl -sS http://127.0.0.1:11556/swap/test/swapout/USDT/0xc241bcd7176aef7c0f4bc331c55c7e282f03aed663eef7b55705b0bd67d402b8
```
