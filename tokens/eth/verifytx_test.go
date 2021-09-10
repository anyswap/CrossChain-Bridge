package eth

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/common/hexutil"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/types"
)

var (
	br = NewCrossChainBridge(true)
)

const (
	swapoutType  = "swapout"
	swapout2Type = "swapout2" // swapout to string address (eg. BTC)
	swapinType   = "swapin"
	nativeType   = "native"

	testPairID = "testpairid"
)

type consArgs struct {
	args    []string
	wantErr error
}

type verifyTxTest struct {
	txtype  string // swpout, swapin, native
	wantErr error

	// for all
	token *tokens.TokenConfig

	// for swpout, swapin
	receipt *types.RPCTxReceipt

	// for native
	tx *types.RPCTransaction
}

var consArgsSlice = []*consArgs{
	{ // 0
		args: []string{
			nativeType, // txtype
			"0x1111111111111111111111111111111111111111", // from
			"0x9999999999999999999999999999999999999999", // to
			"123000000000000000000",                      // value
			"",                                           // contractAddr
			"0x9999999999999999999999999999999999999999", // depositAddr
			"false", // allowCallFromContract
		},
		wantErr: nil,
	},
	{ // 1
		args: []string{
			nativeType, // txtype
			"0x1111111111111111111111111111111111111111", // from
			"0x2222222222222222222222222222222222222222", // to
			"123000000000000000000",                      // value
			"",                                           // contractAddr
			"0x9999999999999999999999999999999999999999", // depositAddr
			"false", // allowCallFromContract
		},
		wantErr: tokens.ErrTxWithWrongReceiver,
	},
	{ // 2
		args: []string{
			swapinType, // txtype
			"0x1111111111111111111111111111111111111111", // from
			"0x6666666666666666666666666666666666666666", // to
			"0x6666666666666666666666666666666666666666", // contractAddr
			"0x9999999999999999999999999999999999999999", // depositAddr
			"false", // allowCallFromContract
			"0x6666666666666666666666666666666666666666",                         // log address
			"0x00000000000000000000000000000000000000000000199ed685fa73de40f30c", // log data
			"false", // log removed
			// log topics
			"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
			"0x0000000000000000000000001111111111111111111111111111111111111111",
			"0x0000000000000000000000009999999999999999999999999999999999999999",
		},
		wantErr: nil,
	},
	{ //3
		args: []string{
			swapinType, // txtype
			"0x1111111111111111111111111111111111111111", // from
			"0x6666666666666666666666666666666666666666", // to
			"0x6666666666666666666666666666666666666666", // contractAddr
			"0x9999999999999999999999999999999999999999", // depositAddr
			"false", // allowCallFromContract
			"0x6666666666666666666666666666666666666666",                         // log address
			"0x00000000000000000000000000000000000000000000199ed685fa73de40f30c", // log data
			"false", // log removed
			// log topics
			"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
			"0x0000000000000000000000001111111111111111111111111111111111111111",
			"0x0000000000000000000000002222222222222222222222222222222222222222",
		},
		wantErr: tokens.ErrTxWithWrongReceiver, // receiver and deposit address mismatch
	},
	{ // 4
		args: []string{
			swapinType, // txtype
			"0x1111111111111111111111111111111111111111", // from
			"0x2222222222222222222222222222222222222222", // to
			"0x6666666666666666666666666666666666666666", // contractAddr
			"0x9999999999999999999999999999999999999999", // depositAddr
			"false", // allowCallFromContract
			"0x6666666666666666666666666666666666666666",                         // log address
			"0x00000000000000000000000000000000000000000000199ed685fa73de40f30c", // log data
			"false", // log removed
			// log topics
			"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
			"0x0000000000000000000000001111111111111111111111111111111111111111",
			"0x0000000000000000000000009999999999999999999999999999999999999999",
		},
		wantErr: tokens.ErrTxWithWrongContract, // to and contract address mismatch
	},
	{ // 5
		args: []string{
			swapinType, // txtype
			"0x1111111111111111111111111111111111111111", // from
			"0x2222222222222222222222222222222222222222", // to
			"0x6666666666666666666666666666666666666666", // contractAddr
			"0x9999999999999999999999999999999999999999", // depositAddr
			"true", // allowCallFromContract
			"0x6666666666666666666666666666666666666666",                         // log address
			"0x00000000000000000000000000000000000000000000199ed685fa73de40f30c", // log data
			"false", // log removed
			// log topics
			"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
			"0x0000000000000000000000001111111111111111111111111111111111111111",
			"0x0000000000000000000000009999999999999999999999999999999999999999",
		},
		wantErr: nil, // allowCallFromContract is true, compare log address with contract address
	},
	{ // 6
		args: []string{
			swapinType, // txtype
			"0x1111111111111111111111111111111111111111", // from
			"0x6666666666666666666666666666666666666666", // to
			"0x6666666666666666666666666666666666666666", // contractAddr
			"0x9999999999999999999999999999999999999999", // depositAddr
			"true", // allowCallFromContract
			"0x7777777777777777777777777777777777777777",                         // log address
			"0x00000000000000000000000000000000000000000000199ed685fa73de40f30c", // log data
			"false", // log removed
			// log topics
			"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
			"0x0000000000000000000000001111111111111111111111111111111111111111",
			"0x0000000000000000000000009999999999999999999999999999999999999999",
		},
		wantErr: tokens.ErrDepositLogNotFound, // log address mismatch
	},
	{ // 7
		args: []string{
			swapinType, // txtype
			"0x1111111111111111111111111111111111111111", // from
			"0x6666666666666666666666666666666666666666", // to
			"0x6666666666666666666666666666666666666666", // contractAddr
			"0x9999999999999999999999999999999999999999", // depositAddr
			"false", // allowCallFromContract
			"0x6666666666666666666666666666666666666666",                         // log address
			"0x00000000000000000000000000000000000000000000199ed685fa73de40f30c", // log data
			"false", // log removed
			// log topics
			"0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925",
			"0x0000000000000000000000001111111111111111111111111111111111111111",
			"0x0000000000000000000000009999999999999999999999999999999999999999",
		},
		wantErr: tokens.ErrDepositLogNotFound, // log topic mismatch
	},
	{ // 8
		args: []string{
			swapoutType, // txtype
			"0x1111111111111111111111111111111111111111", // from
			"0x6666666666666666666666666666666666666666", // to
			"0x6666666666666666666666666666666666666666", // contractAddr
			"",      // depositAddr
			"false", // allowCallFromContract
			"0x6666666666666666666666666666666666666666",                         // log address
			"0x00000000000000000000000000000000000000000000199ed685fa73de40f30c", // log data
			"false", // log removed
			// log topics
			"0x6b616089d04950dc06c45c6dd787d657980543f89651aec47924752c7d16c888",
			"0x0000000000000000000000005c71f679870b190c7545a26981fa1f38aaba839e",
			"0x0000000000000000000000005c71f679870b190c7545a26981fa1f38aaba839e",
		},
		wantErr: nil,
	},
	{ // 9
		args: []string{
			swapout2Type, // txtype
			"0x1111111111111111111111111111111111111111", // from
			"0x6666666666666666666666666666666666666666", // to
			"0x6666666666666666666666666666666666666666", // contractAddr
			"",      // depositAddr
			"false", // allowCallFromContract
			"0x6666666666666666666666666666666666666666", // log address
			"0x00000000000000000000000000000000000000000000199ed685fa73de40f30c0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000002a30783636363636363636363636363636363636363636363636363636363636363636363636363636363636360000000000000000000000000000000000000000", // log data
			"false", // log removed
			// log topics
			"0x9c92ad817e5474d30a4378deface765150479363a897b0590fbb12ae9d89396b",
			"0x0000000000000000000000005ff6c18d800ba845fc874e4ceb08c70c1a394cf8",
		},
		wantErr: nil,
	},
	{ // 10
		args: []string{
			swapout2Type, // txtype
			"0x1111111111111111111111111111111111111111", // from
			"0x6666666666666666666666666666666666666666", // to
			"0x6666666666666666666666666666666666666666", // contractAddr
			"",      // depositAddr
			"false", // allowCallFromContract
			"0x6666666666666666666666666666666666666666",                         // log address
			"0x00000000000000000000000000000000000000000000199ed685fa73de40f30c", // log data
			"false", // log removed
			// log topics
			"0x6b616089d04950dc06c45c6dd787d657980543f89651aec47924752c7d16c888",
			"0x0000000000000000000000005c71f679870b190c7545a26981fa1f38aaba839e",
			"0x0000000000000000000000005c71f679870b190c7545a26981fa1f38aaba839e",
		},
		wantErr: tokens.ErrSwapoutLogNotFound, // log topic mismatch
	},
	{ // 11
		args: []string{
			swapoutType, // txtype
			"0x1111111111111111111111111111111111111111", // from
			"0x6666666666666666666666666666666666666666", // to
			"0x6666666666666666666666666666666666666666", // contractAddr
			"",      // depositAddr
			"false", // allowCallFromContract
			"0x6666666666666666666666666666666666666666",                         // log address
			"0x00000000000000000000000000000000000000000000199ed685fa73de40f30c", // log data
			"false", // log removed
			// log topics
			"0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925",
			"0x0000000000000000000000005c71f679870b190c7545a26981fa1f38aaba839e",
			"0x0000000000000000000000005c71f679870b190c7545a26981fa1f38aaba839e",
		},
		wantErr: tokens.ErrSwapoutLogNotFound,
	},
	{ // 12
		args: []string{
			swapoutType, // txtype
			"0x1111111111111111111111111111111111111111", // from
			"0x6666666666666666666666666666666666666666", // to
			"0x2222222222222222222222222222222222222222", // contractAddr
			"",      // depositAddr
			"false", // allowCallFromContract
			"0x6666666666666666666666666666666666666666",                         // log address
			"0x00000000000000000000000000000000000000000000199ed685fa73de40f30c", // log data
			"false", // log removed
			// log topics
			"0x6b616089d04950dc06c45c6dd787d657980543f89651aec47924752c7d16c888",
			"0x0000000000000000000000005c71f679870b190c7545a26981fa1f38aaba839e",
			"0x0000000000000000000000005c71f679870b190c7545a26981fa1f38aaba839e",
		},
		wantErr: tokens.ErrTxWithWrongContract,
	},
	{ // 13
		args: []string{
			swapoutType, // txtype
			"0x1111111111111111111111111111111111111111", // from
			"0x2222222222222222222222222222222222222222", // to
			"0x6666666666666666666666666666666666666666", // contractAddr
			"",      // depositAddr
			"false", // allowCallFromContract
			"0x6666666666666666666666666666666666666666",                         // log address
			"0x00000000000000000000000000000000000000000000199ed685fa73de40f30c", // log data
			"false", // log removed
			// log topics
			"0x6b616089d04950dc06c45c6dd787d657980543f89651aec47924752c7d16c888",
			"0x0000000000000000000000005c71f679870b190c7545a26981fa1f38aaba839e",
			"0x0000000000000000000000005c71f679870b190c7545a26981fa1f38aaba839e",
		},
		wantErr: tokens.ErrTxWithWrongContract,
	},
	{ // 14
		args: []string{
			swapoutType, // txtype
			"0x1111111111111111111111111111111111111111", // from
			"0x2222222222222222222222222222222222222222", // to
			"0x6666666666666666666666666666666666666666", // contractAddr
			"",     // depositAddr
			"true", // allowCallFromContract
			"0x6666666666666666666666666666666666666666",                         // log address
			"0x00000000000000000000000000000000000000000000199ed685fa73de40f30c", // log data
			"false", // log removed
			// log topics
			"0x6b616089d04950dc06c45c6dd787d657980543f89651aec47924752c7d16c888",
			"0x0000000000000000000000005c71f679870b190c7545a26981fa1f38aaba839e",
			"0x0000000000000000000000005c71f679870b190c7545a26981fa1f38aaba839e",
		},
		wantErr: nil,
	},
	{ // 14
		args: []string{
			swapoutType, // txtype
			"0x1111111111111111111111111111111111111111", // from
			"0x2222222222222222222222222222222222222222", // to
			"0x6666666666666666666666666666666666666666", // contractAddr
			"",     // depositAddr
			"true", // allowCallFromContract
			"0x7777777777777777777777777777777777777777",                         // log address
			"0x00000000000000000000000000000000000000000000199ed685fa73de40f30c", // log data
			"false", // log removed
			// log topics
			"0x6b616089d04950dc06c45c6dd787d657980543f89651aec47924752c7d16c888",
			"0x0000000000000000000000005c71f679870b190c7545a26981fa1f38aaba839e",
			"0x0000000000000000000000005c71f679870b190c7545a26981fa1f38aaba839e",
		},
		wantErr: tokens.ErrSwapoutLogNotFound,
	},
}

// TestVerifyTx test verify tx, compare the verify error with the wanted error
func TestVerifyTx(t *testing.T) {
	br.ChainConfig = &tokens.ChainConfig{
		BlockChain: "testBlockChain",
	}
	tokens.SrcBridge = br
	tokens.DstBridge = br
	params.SetConfig(&params.BridgeConfig{
		Identifier: "testIdentifier",
		Extra: &params.ExtraConfig{
			MustRegisterAccount: false,
		},
	})

	allPassed := true
	tests := constructTests(t, consArgsSlice)
	for i, test := range tests {
		err := verifyTestTx(test)
		if !errors.Is(err, test.wantErr) {
			tokenJs, _ := json.Marshal(test.token)
			receiptJs, _ := json.Marshal(test.receipt)
			allPassed = false
			t.Errorf("verify tx failed, index %v, txtype %v, token %v, receipt %v, want error '%v', real error '%v'",
				i, test.txtype, string(tokenJs), string(receiptJs), test.wantErr, err)
		}
	}
	if allPassed {
		t.Logf("test verify tx all passed with %v test cases", len(tests))
	}
}

func verifyTestTx(test *verifyTxTest) (err error) {
	swapInfo := &tokens.TxSwapInfo{
		PairID: testPairID,
		Hash:   "0x0000000000000000000000000000000000000000000000000000000000000000",
	}
	allowUnstable := true
	completeTokenConfig(test.token)
	switch test.txtype {
	case swapoutType:
		ExtCodeParts = mETHExtCodeParts
		params.GetExtraConfig().IsSwapoutToStringAddress = false
		_, err = br.verifySwapoutTx(swapInfo, allowUnstable, test.token, test.receipt)
	case swapout2Type:
		ExtCodeParts = mBTCExtCodeParts
		params.GetExtraConfig().IsSwapoutToStringAddress = true
		_, err = br.verifySwapoutTx(swapInfo, allowUnstable, test.token, test.receipt)
	case swapinType:
		_, err = br.verifyErc20SwapinTx(swapInfo, allowUnstable, test.token, test.receipt)
	case nativeType:
		_, err = br.verifyNativeSwapinTx(swapInfo, allowUnstable, test.token, test.tx)
	default:
		err = fmt.Errorf("verifyTestTx: unknown swap type '%v'", test.txtype)
	}
	return err
}

func constructTests(t *testing.T, argsSlice []*consArgs) []*verifyTxTest {
	tests := make([]*verifyTxTest, 0, len(argsSlice))
	for _, args := range argsSlice {
		test := constructTest(t, args.args, args.wantErr)
		if test != nil {
			tests = append(tests, test)
		}
	}
	return tests
}

func constructTest(t *testing.T, args []string, wantErr error) *verifyTxTest {
	if len(args) == 0 {
		return nil
	}
	txtype := args[0]
	args = args[1:]
	switch txtype {
	case swapoutType:
		return constructSwapoutTxTest(t, args, wantErr, false)
	case swapout2Type:
		return constructSwapoutTxTest(t, args, wantErr, true)
	case swapinType:
		return constructSwapinTxTest(t, args, wantErr)
	case nativeType:
		return constructNativeTxTest(t, args, wantErr)
	default:
		t.Errorf("constructTest: unknown swap type '%v'", txtype)
	}
	return nil
}

func constructNativeTxTest(t *testing.T, args []string, wantErr error) *verifyTxTest {
	test := &verifyTxTest{
		txtype:  nativeType,
		wantErr: wantErr,
	}

	from, to, amount := getFromToAndValue(t, args)
	contractAddr, depositAddr, allowCallFromContract := getTokenInfo(t, args[3:])

	test.token = &tokens.TokenConfig{
		ContractAddress:         contractAddr,
		DepositAddress:          depositAddr,
		AllowSwapinFromContract: allowCallFromContract,
	}

	test.tx = &types.RPCTransaction{
		From:      from,
		Recipient: to,
		Amount:    amount,
	}

	return test
}

func constructSwapinTxTest(t *testing.T, args []string, wantErr error) *verifyTxTest {
	test := &verifyTxTest{
		txtype:  swapinType,
		wantErr: wantErr,
	}

	from, to := getFromToAddress(t, args)
	contractAddr, depositAddr, allowCallFromContract := getTokenInfo(t, args[2:])
	logAddr, logData, removed, topics := getLogInfo(t, args[5:])

	test.token = &tokens.TokenConfig{
		ContractAddress:         contractAddr,
		DepositAddress:          depositAddr,
		AllowSwapinFromContract: allowCallFromContract,
	}

	log := &types.RPCLog{
		Address: logAddr,
		Topics:  topics,
		Data:    &logData,
		Removed: &removed,
	}

	test.receipt = &types.RPCTxReceipt{
		From:      from,
		Recipient: to,
		Logs:      []*types.RPCLog{log},
	}
	return test
}

func constructSwapoutTxTest(t *testing.T, args []string, wantErr error, isSwapout2Str bool) *verifyTxTest {
	test := &verifyTxTest{
		txtype:  swapoutType,
		wantErr: wantErr,
	}
	if isSwapout2Str {
		test.txtype = swapout2Type
	}

	from, to := getFromToAddress(t, args)
	contractAddr, depositAddr, allowCallFromContract := getTokenInfo(t, args[2:])
	logAddr, logData, removed, topics := getLogInfo(t, args[5:])

	test.token = &tokens.TokenConfig{
		ContractAddress:          contractAddr,
		DepositAddress:           depositAddr,
		AllowSwapoutFromContract: allowCallFromContract,
	}

	log := &types.RPCLog{
		Address: logAddr,
		Topics:  topics,
		Data:    &logData,
		Removed: &removed,
	}

	test.receipt = &types.RPCTxReceipt{
		From:      from,
		Recipient: to,
		Logs:      []*types.RPCLog{log},
	}
	return test
}

func getFromToAddress(t *testing.T, args []string) (from, to *common.Address) {
	t.Helper()
	if len(args) < 2 {
		t.Errorf("getFromToAddress with less args: %v", args)
		return
	}
	fromAddr := common.HexToAddress(args[0])
	toAddr := common.HexToAddress(args[1])
	return &fromAddr, &toAddr
}

func getFromToAndValue(t *testing.T, args []string) (from, to *common.Address, amount *hexutil.Big) {
	t.Helper()
	if len(args) < 3 {
		t.Errorf("getFromToAndValue with less args: %v", args)
		return
	}
	from, to = getFromToAddress(t, args)
	value, err := common.GetBigIntFromStr(args[2])
	if err != nil {
		t.Errorf("getFromToAndValue with error: %v", err)
	}
	amount = (*hexutil.Big)(value)
	return from, to, amount
}

func getTokenInfo(t *testing.T, args []string) (contractAddr, depositAddr string, allowCallFromContract bool) {
	t.Helper()
	if len(args) < 3 {
		t.Errorf("getTokenInfo with less args: %v", args)
		return
	}
	contractAddr = args[0]
	depositAddr = args[1]
	allowCallFromContract = strings.EqualFold(args[2], "true")
	return contractAddr, depositAddr, allowCallFromContract
}

func getLogInfo(t *testing.T, args []string) (logAddr *common.Address, logData hexutil.Bytes, removed bool, topics []common.Hash) {
	t.Helper()
	if len(args) < 4 {
		t.Errorf("getLogInfo with less args: %v", args)
		return
	}
	addr := common.HexToAddress(args[0])
	logData = common.FromHex(args[1])
	removed = strings.EqualFold(args[2], "true")

	topics = make([]common.Hash, len(args)-3)
	for i := 3; i < len(args); i++ {
		topics[i-3] = common.HexToHash(args[i])
	}
	return &addr, logData, removed, topics
}

func completeTokenConfig(token *tokens.TokenConfig) {
	token.ID = "testID"
	token.Name = "testName"
	token.Symbol = "testSymbol"
	testDecimals := uint8(18)
	token.Decimals = &testDecimals

	maximumSwap := 1000000.0
	minimumSwap := 100.0
	bigValueThreshold := 300000.0
	swapFeeRate := 0.001
	maximumSwapFee := 50.0
	minimumSwapFee := 10.0

	token.MaximumSwap = &maximumSwap
	token.MinimumSwap = &minimumSwap
	token.BigValueThreshold = &bigValueThreshold
	token.SwapFeeRate = &swapFeeRate
	token.MaximumSwapFee = &maximumSwapFee
	token.MinimumSwapFee = &minimumSwapFee

	token.CalcAndStoreValue()

	pairsConfig := make(map[string]*tokens.TokenPairConfig)
	pairsConfig[testPairID] = &tokens.TokenPairConfig{
		PairID:    testPairID,
		SrcToken:  token,
		DestToken: token,
	}

	tokens.SetTokenPairsConfig(pairsConfig, false)
}
