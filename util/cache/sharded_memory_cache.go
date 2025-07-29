package cache

import (
	"hash/fnv"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// 分片内存缓存项
type shardedMemoryCacheItem struct {
	data         []byte
	expiry       time.Time
	lastUsed     int64 // 使用原子操作的时间戳
	lastModified time.Time
	size         int
}

// 单个分片
type memoryCacheShard struct {
	items    map[string]*shardedMemoryCacheItem
	mutex    sync.RWMutex
	currSize int64
}

// 分片内存缓存
type ShardedMemoryCache struct {
	shards    []*memoryCacheShard
	shardMask uint32 // 用于快速取模的掩码
	maxItems  int
	maxSize   int64
	itemsPerShard int
	sizePerShard  int64
}

// 创建新的分片内存缓存
func NewShardedMemoryCache(maxItems int, maxSizeMB int) *ShardedMemoryCache {
	// 动态确定分片数量：基于CPU核心数，但至少4个，最多64个
	shardCount := runtime.NumCPU() * 2
	if shardCount < 4 {
		shardCount = 4
	}
	if shardCount > 64 {
		shardCount = 64
	}
	
	// 确保分片数是2的幂，便于使用掩码进行快速取模
	shardCount = nextPowerOfTwo(shardCount)
	
	totalSize := int64(maxSizeMB) * 1024 * 1024
	itemsPerShard := maxItems / shardCount
	sizePerShard := totalSize / int64(shardCount)
	
	shards := make([]*memoryCacheShard, shardCount)
	for i := 0; i < shardCount; i++ {
		shards[i] = &memoryCacheShard{
			items: make(map[string]*shardedMemoryCacheItem),
		}
	}
	
	return &ShardedMemoryCache{
		shards:        shards,
		shardMask:     uint32(shardCount - 1), // 用于快速取模
		maxItems:      maxItems,
		maxSize:       totalSize,
		itemsPerShard: itemsPerShard,
		sizePerShard:  sizePerShard,
	}
}

// 获取下一个2的幂
func nextPowerOfTwo(n int) int {
	if n <= 1 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	return n + 1
}

// 获取分片
func (c *ShardedMemoryCache) getShard(key string) *memoryCacheShard {
	h := fnv.New32a()
	h.Write([]byte(key))
	shardIndex := h.Sum32() & c.shardMask // 使用掩码进行快速取模
	return c.shards[shardIndex]
}

// 设置缓存
func (c *ShardedMemoryCache) Set(key string, data []byte, ttl time.Duration) {
	c.SetWithTimestamp(key, data, ttl, time.Now())
}

// SetWithTimestamp 设置缓存，并指定最后修改时间
func (c *ShardedMemoryCache) SetWithTimestamp(key string, data []byte, ttl time.Duration, lastModified time.Time) {
	shard := c.getShard(key)
	shard.mutex.Lock()
	defer shard.mutex.Unlock()
	
	// 如果已存在，先减去旧项的大小
	if item, exists := shard.items[key]; exists {
		atomic.AddInt64(&shard.currSize, -int64(item.size))
	}
	
	// 创建新的缓存项
	now := time.Now()
	item := &shardedMemoryCacheItem{
		data:         data,
		expiry:       now.Add(ttl),
		lastUsed:     now.UnixNano(),
		lastModified: lastModified,
		size:         len(data),
	}
	
	// 检查是否需要清理空间
	if len(shard.items) >= c.itemsPerShard || shard.currSize+int64(len(data)) > c.sizePerShard {
		c.evictFromShard(shard)
	}
	
	// 存储新项
	shard.items[key] = item
	atomic.AddInt64(&shard.currSize, int64(len(data)))
}

// 获取缓存
func (c *ShardedMemoryCache) Get(key string) ([]byte, bool) {
	shard := c.getShard(key)
	shard.mutex.RLock()
	item, exists := shard.items[key]
	shard.mutex.RUnlock()
	
	if !exists {
		return nil, false
	}
	
	// 检查是否过期
	if time.Now().After(item.expiry) {
		shard.mutex.Lock()
		delete(shard.items, key)
		atomic.AddInt64(&shard.currSize, -int64(item.size))
		shard.mutex.Unlock()
		return nil, false
	}
	
	// 原子操作更新最后使用时间，避免额外的锁
	atomic.StoreInt64(&item.lastUsed, time.Now().UnixNano())
	
	return item.data, true
}

// GetWithTimestamp 获取缓存及其最后修改时间
func (c *ShardedMemoryCache) GetWithTimestamp(key string) ([]byte, time.Time, bool) {
	shard := c.getShard(key)
	shard.mutex.RLock()
	item, exists := shard.items[key]
	shard.mutex.RUnlock()
	
	if !exists {
		return nil, time.Time{}, false
	}
	
	// 检查是否过期
	if time.Now().After(item.expiry) {
		shard.mutex.Lock()
		delete(shard.items, key)
		atomic.AddInt64(&shard.currSize, -int64(item.size))
		shard.mutex.Unlock()
		return nil, time.Time{}, false
	}
	
	// 原子操作更新最后使用时间
	atomic.StoreInt64(&item.lastUsed, time.Now().UnixNano())
	
	return item.data, item.lastModified, true
}

// GetLastModified 获取缓存项的最后修改时间
func (c *ShardedMemoryCache) GetLastModified(key string) (time.Time, bool) {
	shard := c.getShard(key)
	shard.mutex.RLock()
	defer shard.mutex.RUnlock()
	
	item, exists := shard.items[key]
	if !exists {
		return time.Time{}, false
	}
	
	// 检查是否过期
	if time.Now().After(item.expiry) {
		return time.Time{}, false
	}
	
	return item.lastModified, true
}

// 从指定分片中驱逐最久未使用的项
func (c *ShardedMemoryCache) evictFromShard(shard *memoryCacheShard) {
	var oldestKey string
	var oldestTime int64 = 9223372036854775807 // int64最大值
	
	for k, v := range shard.items {
		lastUsed := atomic.LoadInt64(&v.lastUsed)
		if lastUsed < oldestTime {
			oldestKey = k
			oldestTime = lastUsed
		}
	}
	
	// 如果找到了最久未使用的项，删除它
	if oldestKey != "" {
		item := shard.items[oldestKey]
		atomic.AddInt64(&shard.currSize, -int64(item.size))
		delete(shard.items, oldestKey)
	}
}

// 清理过期项
func (c *ShardedMemoryCache) CleanExpired() {
	now := time.Now()
	
	// 并行清理所有分片
	var wg sync.WaitGroup
	for _, shard := range c.shards {
		wg.Add(1)
		go func(s *memoryCacheShard) {
			defer wg.Done()
			s.mutex.Lock()
			defer s.mutex.Unlock()
			
			for k, v := range s.items {
				if now.After(v.expiry) {
					atomic.AddInt64(&s.currSize, -int64(v.size))
					delete(s.items, k)
				}
			}
		}(shard)
	}
	wg.Wait()
}

// Delete 删除指定键的缓存项
func (c *ShardedMemoryCache) Delete(key string) {
	shard := c.getShard(key)
	shard.mutex.Lock()
	defer shard.mutex.Unlock()
	
	if item, exists := shard.items[key]; exists {
		atomic.AddInt64(&shard.currSize, -int64(item.size))
		delete(shard.items, key)
	}
}

// Clear 清空所有缓存项
func (c *ShardedMemoryCache) Clear() {
	// 并行清理所有分片
	var wg sync.WaitGroup
	for _, shard := range c.shards {
		wg.Add(1)
		go func(s *memoryCacheShard) {
			defer wg.Done()
			s.mutex.Lock()
			defer s.mutex.Unlock()
			
			s.items = make(map[string]*shardedMemoryCacheItem)
			atomic.StoreInt64(&s.currSize, 0)
		}(shard)
	}
	wg.Wait()
}

// 启动定期清理
func (c *ShardedMemoryCache) StartCleanupTask() {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for range ticker.C {
			c.CleanExpired()
		}
	}()
}