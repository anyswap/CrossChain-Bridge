package btc

import (
	"time"

	"github.com/fsn-dev/crossChain-Bridge/dcrm"
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/mongodb"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
)

func (b *Bridge) processTransaction(txid string) {
	_ = b.processSwapin(txid)
	_ = b.processP2shSwapin(txid)
}

func (b *Bridge) processSwapin(txid string) error {
	swap, _ := mongodb.FindSwapin(txid)
	if swap != nil {
		return nil
	}
	swapInfo, err := b.VerifyTransaction(txid, true)
	if !tokens.ShouldRegisterSwapForError(err) {
		//log.Trace("[scan] processSwapin", "txid", txid, "err", err)
		return err
	}
	err = b.registerSwapin(txid, swapInfo.Bind)
	if err != nil {
		log.Trace("[scan] processSwapin", "txid", txid, "err", err)
	}
	return err
}

func (b *Bridge) registerSwapin(txid string, bind string) error {
	isServer := dcrm.IsSwapServer()
	log.Info("[scan] register swapin", "isServer", isServer, "tx", txid, "bind", bind)
	if isServer {
		swap := &mongodb.MgoSwap{
			Key:       txid,
			TxType:    uint32(tokens.SwapinTx),
			Bind:      bind,
			TxID:      txid,
			Status:    mongodb.TxNotStable,
			Timestamp: time.Now().Unix(),
		}
		return mongodb.AddSwapin(swap)
	}
	return nil
}

func (b *Bridge) processP2shSwapin(txid string) error {
	swap, _ := mongodb.FindSwapin(txid)
	if swap != nil {
		return nil
	}
	swapInfo, err := b.checkP2shTransaction(txid, true)
	if !tokens.ShouldRegisterSwapForError(err) {
		//log.Trace("[scan] processP2shSwapin", "txid", txid, "err", err)
		return err
	}
	err = b.registerP2shSwapin(txid, swapInfo.Bind)
	if err != nil {
		log.Trace("[scan] processP2shSwapin", "txid", txid, "err", err)
	}
	return err
}

func (b *Bridge) registerP2shSwapin(txid string, bind string) error {
	log.Info("[scan] register p2sh swapin", "tx", txid, "bind", bind)
	swap := &mongodb.MgoSwap{
		Key:       txid,
		TxID:      txid,
		TxType:    uint32(tokens.P2shSwapinTx),
		Bind:      bind,
		Status:    mongodb.TxNotStable,
		Timestamp: time.Now().Unix(),
	}
	return mongodb.AddSwapin(swap)
}

func (b *Bridge) checkP2shTransaction(txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	tx, err := b.GetTransactionByHash(txHash)
	if err != nil {
		log.Debug(b.TokenConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txHash, "err", err)
		return nil, tokens.ErrTxNotFound
	}
	var bindAddress, p2shAddress string
	for _, output := range tx.Vout {
		switch *output.ScriptpubkeyType {
		case "p2sh":
			p2shAddress = *output.ScriptpubkeyAddress
			bindAddress = getBindAddress(p2shAddress)
			if bindAddress != "" {
				break
			}
		}
	}
	if bindAddress == "" {
		return nil, tokens.ErrTxWithWrongReceiver
	}
	return b.VerifyP2shTransaction(txHash, bindAddress, allowUnstable)
}

func getBindAddress(p2shAddress string) (bindAddress string) {
	bindAddress, _ = mongodb.FindP2shBindAddress(p2shAddress)
	return bindAddress
}
