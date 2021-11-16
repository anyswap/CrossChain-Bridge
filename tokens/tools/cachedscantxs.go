package tools

// NewCachedScannedTxs new cached scanned txs
func NewCachedScannedTxs(capacity int) *CachedScannedTxs {
	return &CachedScannedTxs{
		nextIndex: 0,
		capacity:  capacity,
		txs:       make([]cachedScannedTxRecord, capacity),
	}
}

// CachedScannedTxs cached scanned txs
type CachedScannedTxs struct {
	nextIndex int
	capacity  int
	txs       []cachedScannedTxRecord
}

type cachedScannedTxRecord struct {
	hash string
}

// CacheScannedTx add cache tx
func (c *CachedScannedTxs) CacheScannedTx(hash string) {
	c.txs[c.nextIndex] = cachedScannedTxRecord{
		hash: hash,
	}
	c.nextIndex = (c.nextIndex + 1) % c.capacity
}

// IsTxScanned return if cache tx exist
func (c *CachedScannedTxs) IsTxScanned(txHash string) bool {
	for _, tx := range c.txs {
		if tx.hash == txHash {
			return true
		}
	}
	return false
}
