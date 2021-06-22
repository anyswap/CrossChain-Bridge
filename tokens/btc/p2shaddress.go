package btc

import (
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

func (b *Bridge) getP2shAddressWithMemo(memo, pubKeyHash []byte) (p2shAddress string, redeemScript []byte, err error) {
	redeemScript, err = b.GetP2shRedeemScript(memo, pubKeyHash)
	if err != nil {
		return
	}
	addressScriptHash, err := b.NewAddressScriptHash(redeemScript)
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
	pairID := PairID
	tokenCfg := b.GetTokenConfig(pairID)
	if tokenCfg == nil {
		return "", nil, tokens.ErrUnknownPairID
	}

	dcrmAddress := tokenCfg.DcrmAddress
	address, err := b.DecodeAddress(dcrmAddress)
	if err != nil {
		return "", nil, fmt.Errorf("invalid dcrm address %v, %w", dcrmAddress, err)
	}
	pubKeyHash := address.ScriptAddress()
	return b.getP2shAddressWithMemo(memo, pubKeyHash)
}

func (b *Bridge) getRedeemScriptByOutputScrpit(preScript []byte) ([]byte, error) {
	pkScript, err := b.ParsePkScript(preScript)
	if err != nil {
		return nil, err
	}
	p2shAddress, err := pkScript.Address(b.Inherit.GetChainParams())
	if err != nil {
		return nil, err
	}
	p2shAddr := p2shAddress.String()
	bindAddr := tools.GetP2shBindAddress(p2shAddr)
	if bindAddr == "" {
		return nil, fmt.Errorf("p2sh address %v is not registered", p2shAddr)
	}
	var address string
	address, redeemScript, _ := b.GetP2shAddress(bindAddr)
	if address != p2shAddr {
		return nil, fmt.Errorf("p2sh address mismatch for bind address %v, have %v want %v", bindAddr, p2shAddr, address)
	}
	return redeemScript, nil
}

// GetP2shAddressByRedeemScript get p2sh address by redeem script
func (b *Bridge) GetP2shAddressByRedeemScript(redeemScript []byte) (string, error) {
	addressScriptHash, err := b.NewAddressScriptHash(redeemScript)
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
	return b.GetPayToAddrScript(p2shAddr)
}
