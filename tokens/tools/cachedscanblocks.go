package tools

// NewCachedScannedBlocks new cached scanned blocks
func NewCachedScannedBlocks(capacity int) *cachedScannedBlocks {
	return &cachedScannedBlocks{
		nextIndex: 0,
		capacity:  capacity,
		blocks:    make([]cachedScannedBlockRecord, capacity),
	}
}

type cachedScannedBlocks struct {
	nextIndex int
	capacity  int
	blocks    []cachedScannedBlockRecord
}

type cachedScannedBlockRecord struct {
	hash   string
	height uint64
}

func (c *cachedScannedBlocks) CacheScannedBlock(hash string, height uint64) {
	c.blocks[c.nextIndex] = cachedScannedBlockRecord{
		hash:   hash,
		height: height,
	}
	c.nextIndex = (c.nextIndex + 1) % c.capacity
}

func (c *cachedScannedBlocks) IsBlockScanned(blockHash string) bool {
	for _, block := range c.blocks {
		if block.hash == blockHash {
			return true
		}
	}
	return false
}
