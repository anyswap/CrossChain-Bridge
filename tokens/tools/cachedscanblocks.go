package tools

// NewCachedScannedBlocks new cached scanned blocks
func NewCachedScannedBlocks(capacity int) *CachedScannedBlocks {
	return &CachedScannedBlocks{
		nextIndex: 0,
		capacity:  capacity,
		blocks:    make([]cachedScannedBlockRecord, capacity),
	}
}

// CachedScannedBlocks cached scanned blocks
type CachedScannedBlocks struct {
	nextIndex int
	capacity  int
	blocks    []cachedScannedBlockRecord
}

type cachedScannedBlockRecord struct {
	hash   string
	height uint64
}

// CacheScannedBlock add cache block
func (c *CachedScannedBlocks) CacheScannedBlock(hash string, height uint64) {
	c.blocks[c.nextIndex] = cachedScannedBlockRecord{
		hash:   hash,
		height: height,
	}
	c.nextIndex = (c.nextIndex + 1) % c.capacity
}

// IsBlockScanned return if cache block exist
func (c *CachedScannedBlocks) IsBlockScanned(blockHash string) bool {
	for _, block := range c.blocks {
		if block.hash == blockHash {
			return true
		}
	}
	return false
}
