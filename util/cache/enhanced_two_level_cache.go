package cache

import (
	"sync"
	"time"

	"pansou/config"
)

// EnhancedTwoLevelCache 改进的两级缓存
type EnhancedTwoLevelCache struct {
	memory     *ShardedMemoryCache
	disk       *ShardedDiskCache
	mutex      sync.RWMutex
	serializer Serializer
}

// NewEnhancedTwoLevelCache 创建新的改进两级缓存
func NewEnhancedTwoLevelCache() (*EnhancedTwoLevelCache, error) {
	// 内存缓存大小为磁盘缓存的60%
	memCacheMaxItems := 5000
	memCacheSizeMB := config.AppConfig.CacheMaxSizeMB * 3 / 5
	
	memCache := NewShardedMemoryCache(memCacheMaxItems, memCacheSizeMB)
	memCache.StartCleanupTask()

	// 创建优化的分片磁盘缓存，使用动态分片数量
	diskCache, err := NewOptimizedShardedDiskCache(config.AppConfig.CachePath, config.AppConfig.CacheMaxSizeMB)
	if err != nil {
		return nil, err
	}

	// 创建序列化器
	serializer := NewGobSerializer()

	// 🔥 设置内存缓存的磁盘缓存引用，用于LRU淘汰时的备份
	memCache.SetDiskCacheReference(diskCache)

	return &EnhancedTwoLevelCache{
		memory:     memCache,
		disk:       diskCache,
		serializer: serializer,
	}, nil
}

// Set 设置缓存
func (c *EnhancedTwoLevelCache) Set(key string, data []byte, ttl time.Duration) error {
	// 获取当前时间作为最后修改时间
	now := time.Now()
	
	// 先设置内存缓存（这是快速操作，直接在当前goroutine中执行）
	c.memory.SetWithTimestamp(key, data, ttl, now)
	
	// 异步设置磁盘缓存（这是IO操作，可能较慢）
	go func(k string, d []byte, t time.Duration) {
		// 使用独立的goroutine写入磁盘，避免阻塞调用者
		_ = c.disk.Set(k, d, t)
	}(key, data, ttl)
	
	return nil
}

// SetMemoryOnly 仅更新内存缓存
func (c *EnhancedTwoLevelCache) SetMemoryOnly(key string, data []byte, ttl time.Duration) error {
	now := time.Now()
	
	// 🔥 只更新内存缓存，不触发磁盘写入
	c.memory.SetWithTimestamp(key, data, ttl, now)
	
	return nil
}

// SetBothLevels 更新内存和磁盘缓存
func (c *EnhancedTwoLevelCache) SetBothLevels(key string, data []byte, ttl time.Duration) error {
	now := time.Now()
	
	// 同步更新内存缓存
	c.memory.SetWithTimestamp(key, data, ttl, now)
	
	// 同步更新磁盘缓存（确保数据持久化）
	return c.disk.Set(key, data, ttl)
}

// SetWithFinalFlag 根据结果状态选择更新策略
func (c *EnhancedTwoLevelCache) SetWithFinalFlag(key string, data []byte, ttl time.Duration, isFinal bool) error {
	if isFinal {
		return c.SetBothLevels(key, data, ttl)
	} else {
		return c.SetMemoryOnly(key, data, ttl)
	}
}

// Get 获取缓存
func (c *EnhancedTwoLevelCache) Get(key string) ([]byte, bool, error) {
	
	// 检查内存缓存
	data, _, memHit := c.memory.GetWithTimestamp(key)
	if memHit {
		return data, true, nil
	}

    // 尝试从磁盘读取数据
	diskData, diskHit, diskErr := c.disk.Get(key)
	if diskErr == nil && diskHit {
		// 磁盘缓存命中，更新内存缓存
		diskLastModified, _ := c.disk.GetLastModified(key)
		ttl := time.Duration(config.AppConfig.CacheTTLMinutes) * time.Minute
		c.memory.SetWithTimestamp(key, diskData, ttl, diskLastModified)
		return diskData, true, nil
	}
	
	return nil, false, nil
}

// Delete 删除缓存
func (c *EnhancedTwoLevelCache) Delete(key string) error {
	// 从内存缓存删除
	c.memory.Delete(key)
	
	// 从磁盘缓存删除
	return c.disk.Delete(key)
}

// Clear 清空所有缓存
func (c *EnhancedTwoLevelCache) Clear() error {
	// 清空内存缓存
	c.memory.Clear()
	
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

// FlushMemoryToDisk 将内存中的所有缓存项同步到磁盘
func (c *EnhancedTwoLevelCache) FlushMemoryToDisk() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	// 获取所有内存缓存项
	items := c.memory.GetAllItems()
	
	if len(items) == 0 {
		return
	}
	
	for key, item := range items {
		ttl := time.Until(item.Expiry)
		if ttl > 0 {
			_ = c.disk.Set(key, item.Data, ttl)
		}
	}
} 