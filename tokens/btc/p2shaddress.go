package btc

import (
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
)

// GetP2shAddressWithMemo common
func GetP2shAddressWithMemo(memo, pubKeyHash []byte, net *chaincfg.Params) (p2shAddress string, redeemScript []byte, err error) {
	redeemScript, err = txscript.NewScriptBuilder().
		AddData(memo).AddOp(txscript.OP_DROP).
		AddOp(txscript.OP_DUP).AddOp(txscript.OP_HASH160).AddData(pubKeyHash).
		AddOp(txscript.OP_EQUALVERIFY).AddOp(txscript.OP_CHECKSIG).
		Script()
	if err != nil {
		return
	}
	var addressScriptHash *btcutil.AddressScriptHash
	addressScriptHash, err = btcutil.NewAddressScriptHash(redeemScript, net)
	if err != nil {
		return
	}
	p2shAddress = addressScriptHash.EncodeAddress()
	return
}

// GetP2shAddress get p2sh address from bind address
func (b *Bridge) GetP2shAddress(bindAddr string) (p2shAddress string, redeemScript []byte, err error) {
	if !tokens.GetCrossChainBridge(!b.IsSrc).IsValidAddress(bindAddr) {
		return "", nil, fmt.Errorf("invalid bind address %v", bindAddr)
	}
	memo := common.FromHex(bindAddr)
	net := b.GetChainParams()
	pairID := PairID
	tokenCfg := b.GetTokenConfig(pairID)
	if tokenCfg == nil {
		return "", nil, tokens.ErrUnknownPairID
	}

	dcrmAddress := tokenCfg.DcrmAddress
	address, err := btcutil.DecodeAddress(dcrmAddress, net)
	if err != nil {
		return "", nil, fmt.Errorf("invalid dcrm address %v, %v", dcrmAddress, err)
	}
	pubKeyHash := address.ScriptAddress()
	return GetP2shAddressWithMemo(memo, pubKeyHash, net)
}

func (b *Bridge) getRedeemScriptByOutputScrpit(preScript []byte) ([]byte, error) {
	pkScript, err := txscript.ParsePkScript(preScript)
	if err != nil {
		return nil, err
	}
	p2shAddress, err := pkScript.Address(b.GetChainParams())
	if err != nil {
		return nil, err
	}
	p2shAddr := p2shAddress.String()
	bindAddr := tools.GetP2shBindAddress(p2shAddr)
	if bindAddr == "" {
		return nil, fmt.Errorf("ps2h address %v is registered", p2shAddr)
	}
	var address string
	address, redeemScript, _ := b.GetP2shAddress(bindAddr)
	if address != p2shAddr {
		return nil, fmt.Errorf("ps2h address mismatch for bind address %v, have %v want %v", bindAddr, p2shAddr, address)
	}
	return redeemScript, nil
}

// GetP2shAddressByRedeemScript get p2sh address by redeem script
func (b *Bridge) GetP2shAddressByRedeemScript(redeemScript []byte) (string, error) {
	net := b.GetChainParams()
	addressScriptHash, err := btcutil.NewAddressScriptHash(redeemScript, net)
	if err != nil {
		return "", err
	}
	return addressScriptHash.EncodeAddress(), nil
}

// GetP2shSigScript get p2sh signature script
func (b *Bridge) GetP2shSigScript(redeemScript []byte) ([]byte, error) {
	p2shAddr, err := b.GetP2shAddressByRedeemScript(redeemScript)
	if err != nil {
		return nil, err
	}
	return b.getPayToAddrScript(p2shAddr)
}
