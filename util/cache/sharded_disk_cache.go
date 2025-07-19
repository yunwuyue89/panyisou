package cache

import (
	"fmt"
	"hash/fnv"
	"path/filepath"
	"sync"
	"time"
)

// ShardedDiskCache 分片磁盘缓存
type ShardedDiskCache struct {
	baseDir     string
	shardCount  int
	shards      []*DiskCache
	maxSizeMB   int
	mutex       sync.RWMutex
}

// NewShardedDiskCache 创建新的分片磁盘缓存
func NewShardedDiskCache(baseDir string, shardCount, maxSizeMB int) (*ShardedDiskCache, error) {
	// 确保每个分片的大小合理
	shardSize := maxSizeMB / shardCount
	if shardSize < 1 {
		shardSize = 1
	}
	
	cache := &ShardedDiskCache{
		baseDir:    baseDir,
		shardCount: shardCount,
		shards:     make([]*DiskCache, shardCount),
		maxSizeMB:  maxSizeMB,
	}
	
	// 初始化每个分片
	for i := 0; i < shardCount; i++ {
		shardPath := filepath.Join(baseDir, fmt.Sprintf("shard_%d", i))
		diskCache, err := NewDiskCache(shardPath, shardSize)
		if err != nil {
			return nil, err
		}
		cache.shards[i] = diskCache
	}
	
	return cache, nil
}

// 获取键对应的分片
func (c *ShardedDiskCache) getShard(key string) *DiskCache {
	// 计算哈希值决定分片
	h := fnv.New32a()
	h.Write([]byte(key))
	shardIndex := int(h.Sum32()) % c.shardCount
	return c.shards[shardIndex]
}

// Set 设置缓存
func (c *ShardedDiskCache) Set(key string, data []byte, ttl time.Duration) error {
	shard := c.getShard(key)
	return shard.Set(key, data, ttl)
}

// Get 获取缓存
func (c *ShardedDiskCache) Get(key string) ([]byte, bool, error) {
	shard := c.getShard(key)
	return shard.Get(key)
}

// Delete 删除缓存
func (c *ShardedDiskCache) Delete(key string) error {
	shard := c.getShard(key)
	return shard.Delete(key)
}

// Has 检查缓存是否存在
func (c *ShardedDiskCache) Has(key string) bool {
	shard := c.getShard(key)
	return shard.Has(key)
}

// Clear 清空所有缓存
func (c *ShardedDiskCache) Clear() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	var lastErr error
	for _, shard := range c.shards {
		if err := shard.Clear(); err != nil {
			lastErr = err
		}
	}
	
	return lastErr
} 

// GetLastModified 获取缓存项的最后修改时间
func (c *ShardedDiskCache) GetLastModified(key string) (time.Time, bool) {
	shard := c.getShard(key)
	return shard.GetLastModified(key)
} 