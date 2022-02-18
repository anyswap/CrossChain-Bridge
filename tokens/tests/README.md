# Testmode suppor

test mode get rids of business related components: MPC, DB, etc.

use config examples in `tokens/tests` directory.

```text
~/temp/bridge-test/
├── bridge-eth2bsc.toml
└── tokenpairs
    └── bridge-eth2bsc-test-token.toml
```

start the program

```shell
./build/bin/swapserver --config ~/temp/bridge-test/bridge-eth2bsc.toml --pairsdir ~/temp/bridge-test/tokenpairs
```

trigger the test

```shell
# format "/swap/test/{swaptype}/{pairid}/{txid}"
curl -sS http://127.0.0.1:11556/swap/test/swapout/USDT/0xc241bcd7176aef7c0f4bc331c55c7e282f03aed663eef7b55705b0bd67d402b8
```
