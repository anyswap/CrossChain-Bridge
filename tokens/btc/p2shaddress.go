package btc

import (
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
)

// GetP2shAddressWithMemo common
func GetP2shAddressWithMemo(memo []byte, pubKeyHash []byte, net *chaincfg.Params) (address string, script []byte, err error) {
	script, err = txscript.NewScriptBuilder().
		AddData(memo).AddOp(txscript.OP_DROP).
		AddOp(txscript.OP_DUP).AddOp(txscript.OP_HASH160).AddData(pubKeyHash).
		AddOp(txscript.OP_EQUALVERIFY).AddOp(txscript.OP_CHECKSIG).
		Script()
	if err != nil {
		return
	}
	var addressScriptHash *btcutil.AddressScriptHash
	addressScriptHash, err = btcutil.NewAddressScriptHash(script, net)
	if err != nil {
		return
	}
	address = addressScriptHash.EncodeAddress()
	return
}

// GetP2shAddress get p2sh address from bind address
func (b *Bridge) GetP2shAddress(bindAddr string) (string, []byte, error) {
	if !tokens.GetCrossChainBridge(!b.IsSrc).IsValidAddress(bindAddr) {
		return "", nil, fmt.Errorf("invalid bind address %v", bindAddr)
	}
	memo := common.FromHex(bindAddr)
	net := b.GetChainConfig()
	dcrmAddress := b.TokenConfig.DcrmAddress
	address, _ := btcutil.DecodeAddress(dcrmAddress, net)
	pubKeyHash := address.ScriptAddress()
	return GetP2shAddressWithMemo(memo, pubKeyHash, net)
}
