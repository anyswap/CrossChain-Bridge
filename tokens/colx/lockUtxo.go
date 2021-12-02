package colx

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
)

var lockedUtxos = make(map[utxokey]int)
var unlockConds = make(map[utxokey]func() bool)
var utxoLock sync.RWMutex

type utxokey struct {
	txhash string
	vout   int
}

var (
	errUtxoLocked = fmt.Errorf("[Locked utxo] Utxo is already locked")
	errNotLocked  = fmt.Errorf("[Locked utxo] Utxo is not locked")
)

// IsUtxoLocked is utxo locked
func (b *Bridge) IsUtxoLocked(txhash string, vout int) bool {
	txhash = strings.ToUpper(txhash)
	key := utxokey{txhash: txhash, vout: vout}
	return b.isUtxoLocked(key)
}

func (b *Bridge) isUtxoLocked(key utxokey) bool {
	utxoLock.RLock()
	defer utxoLock.RUnlock()
	return (lockedUtxos[key] == 1)
}

// defaultUnlockCond unlock utxo after 5 days
var defaultUnlockCond = after(3600 * 120)

// LockUtxo lock utxo
func (b *Bridge) LockUtxo(txhash string, vout int) error {
	// Use default unlock cond
	return b.LockUtxoWithCond(txhash, vout, defaultUnlockCond)
}

// LockUtxoWithCond lock utxo with condition
func (b *Bridge) LockUtxoWithCond(txhash string, vout int, cond func() bool) error {
	txhash = strings.ToUpper(txhash)
	key := utxokey{txhash: txhash, vout: vout}
	return b.lockUtxo(key, cond)
}

func after(seconds int64) func() bool {
	deadline := time.Now().Unix() + seconds
	return func() bool {
		return time.Now().Unix() >= deadline
	}
}

func (b *Bridge) lockUtxo(key utxokey, cond func() bool) error {
	if b.isUtxoLocked(key) {
		return errUtxoLocked
	}
	utxoLock.Lock()
	defer utxoLock.Unlock()
	lockedUtxos[key] = 1
	unlockConds[key] = cond
	return nil
}

// UnlockUtxo unlock utxo
func (b *Bridge) UnlockUtxo(txhash string, vout int) {
	txhash = strings.ToUpper(txhash)
	key := utxokey{txhash: txhash, vout: vout}
	b.unlockUtxo(key)
}

func (b *Bridge) unlockUtxo(key utxokey) {
	utxoLock.Lock()
	defer utxoLock.Unlock()
	delete(lockedUtxos, key)
	delete(unlockConds, key)
}

// SetUnlockUtxoCond set unlock utxo condition
func (b *Bridge) SetUnlockUtxoCond(txhash string, vout int, cond func() bool) error {
	txhash = strings.ToUpper(txhash)
	key := utxokey{txhash: txhash, vout: vout}
	return b.setUnlockUtxoCond(key, cond)
}

func (b *Bridge) setUnlockUtxoCond(key utxokey, cond func() bool) error {
	utxoLock.Lock()
	defer utxoLock.Unlock()
	if lockedUtxos[key] != 1 {
		return errNotLocked
	}
	unlockConds[key] = cond
	return nil
}

// StartMonitLockedUtxo start monitor locked utxo
func (b *Bridge) StartMonitLockedUtxo() {
	log.Info("Start monit locked utxo")
	for {
		log.Debug("Check locked utxos", "number locked", len(lockedUtxos))
		for key, cond := range unlockConds {
			if cond() {
				log.Debug("Unlock utxo", "key", key)
				b.unlockUtxo(key)
			}
		}
		time.Sleep(time.Second * 60)
	}
}
