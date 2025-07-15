package cache

import (
	"sync"
	"time"

	"pansou/config"
)

// EnhancedTwoLevelCache 改进的两级缓存
type EnhancedTwoLevelCache struct {
	memory     *MemoryCache
	disk       *ShardedDiskCache
	mutex      sync.RWMutex
	serializer Serializer
}

// NewEnhancedTwoLevelCache 创建新的改进两级缓存
func NewEnhancedTwoLevelCache() (*EnhancedTwoLevelCache, error) {
	// 内存缓存大小为磁盘缓存的60%
	memCacheMaxItems := 5000
	memCacheSizeMB := config.AppConfig.CacheMaxSizeMB * 3 / 5
	
	memCache := NewMemoryCache(memCacheMaxItems, memCacheSizeMB)
	memCache.StartCleanupTask()

	// 创建分片磁盘缓存，默认使用8个分片
	shardCount := 8
	diskCache, err := NewShardedDiskCache(config.AppConfig.CachePath, shardCount, config.AppConfig.CacheMaxSizeMB)
	if err != nil {
		return nil, err
	}

	// 创建序列化器
	serializer := NewGobSerializer()

	return &EnhancedTwoLevelCache{
		memory:     memCache,
		disk:       diskCache,
		serializer: serializer,
	}, nil
}

// Set 设置缓存
func (c *EnhancedTwoLevelCache) Set(key string, data []byte, ttl time.Duration) error {
	// 先设置内存缓存（这是快速操作，直接在当前goroutine中执行）
	c.memory.Set(key, data, ttl)
	
	// 异步设置磁盘缓存（这是IO操作，可能较慢）
	go func(k string, d []byte, t time.Duration) {
		// 使用独立的goroutine写入磁盘，避免阻塞调用者
		_ = c.disk.Set(k, d, t)
	}(key, data, ttl)
	
	return nil
}

// Get 获取缓存
func (c *EnhancedTwoLevelCache) Get(key string) ([]byte, bool, error) {
	// 优先检查内存缓存
	if data, found := c.memory.Get(key); found {
		return data, true, nil
	}
	
	// 内存未命中，检查磁盘缓存
	data, found, err := c.disk.Get(key)
	if err != nil {
		return nil, false, err
	}
	
	if found {
		// 磁盘命中，更新内存缓存
		ttl := time.Duration(config.AppConfig.CacheTTLMinutes) * time.Minute
		c.memory.Set(key, data, ttl)
		return data, true, nil
	}
	
	return nil, false, nil
}

// Delete 删除缓存
func (c *EnhancedTwoLevelCache) Delete(key string) error {
	// 从内存缓存删除
	c.memory.mutex.Lock()
	if item, exists := c.memory.items[key]; exists {
		c.memory.currSize -= int64(item.size)
		delete(c.memory.items, key)
	}
	c.memory.mutex.Unlock()
	
	// 从磁盘缓存删除
	return c.disk.Delete(key)
}

// Clear 清空所有缓存
func (c *EnhancedTwoLevelCache) Clear() error {
	// 清空内存缓存
	c.memory.mutex.Lock()
	c.memory.items = make(map[string]*memoryCacheItem)
	c.memory.currSize = 0
	c.memory.mutex.Unlock()
	
	// 清空磁盘缓存
	return c.disk.Clear()
}

// 设置序列化器
func (c *EnhancedTwoLevelCache) SetSerializer(serializer Serializer) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.serializer = serializer
}

// 获取序列化器
func (c *EnhancedTwoLevelCache) GetSerializer() Serializer {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.serializer
} 