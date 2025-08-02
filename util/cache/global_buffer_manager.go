package cache

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// GlobalBufferStrategy 全局缓冲策略
type GlobalBufferStrategy string

const (
	// BufferByKeyword 按关键词缓冲
	BufferByKeyword GlobalBufferStrategy = "keyword"
	
	// BufferByPlugin 按插件缓冲
	BufferByPlugin GlobalBufferStrategy = "plugin"
	
	// BufferByPattern 按搜索模式缓冲
	BufferByPattern GlobalBufferStrategy = "pattern"
	
	// BufferHybrid 混合缓冲策略
	BufferHybrid GlobalBufferStrategy = "hybrid"
)

// SearchPattern 搜索模式
type SearchPattern struct {
	KeywordPattern   string            // 关键词模式
	PluginSet        []string          // 插件集合
	TimeWindow       time.Duration     // 时间窗口
	Frequency        int               // 频率
	LastAccessTime   time.Time         // 最后访问时间
	Metadata         map[string]interface{} // 元数据
}

// GlobalBuffer 全局缓冲区
type GlobalBuffer struct {
	// 基础信息
	ID               string                   // 缓冲区ID
	Strategy         GlobalBufferStrategy     // 缓冲策略
	CreatedAt        time.Time               // 创建时间
	LastUpdatedAt    time.Time               // 最后更新时间
	
	// 数据存储
	Operations       []*CacheOperation       // 操作列表
	KeywordGroups    map[string][]*CacheOperation // 按关键词分组
	PluginGroups     map[string][]*CacheOperation // 按插件分组
	
	// 统计信息
	TotalOperations  int64                   // 总操作数
	TotalDataSize    int64                   // 总数据大小
	CompressRatio    float64                 // 压缩比例
	
	// 控制参数
	MaxOperations    int                     // 最大操作数
	MaxDataSize      int64                   // 最大数据大小
	MaxAge           time.Duration           // 最大存活时间
	
	mutex            sync.RWMutex            // 读写锁
}

// GlobalBufferManager 全局缓冲区管理器
type GlobalBufferManager struct {
	// 配置
	strategy          GlobalBufferStrategy
	maxBuffers        int                    // 最大缓冲区数量
	defaultBufferSize int                    // 默认缓冲区大小
	
	// 缓冲区管理
	buffers          map[string]*GlobalBuffer // 缓冲区映射
	buffersMutex     sync.RWMutex            // 缓冲区锁
	
	// 搜索模式分析
	patternAnalyzer  *SearchPatternAnalyzer
	
	// 数据合并器
	dataMerger       *AdvancedDataMerger
	
	// 状态监控
	statusMonitor    *BufferStatusMonitor
	
	// 统计信息
	stats            *GlobalBufferStats
	
	// 控制通道
	cleanupTicker    *time.Ticker
	shutdownChan     chan struct{}
	
	// 初始化状态
	initialized      int32
}

// GlobalBufferStats 全局缓冲区统计
type GlobalBufferStats struct {
	// 缓冲区统计
	ActiveBuffers        int64     // 活跃缓冲区数量
	TotalBuffersCreated  int64     // 总创建缓冲区数量
	TotalBuffersDestroyed int64    // 总销毁缓冲区数量
	
	// 操作统计
	TotalOperationsBuffered int64  // 总缓冲操作数
	TotalOperationsMerged   int64  // 总合并操作数
	TotalDataMerged         int64  // 总合并数据大小
	
	// 效率统计
	AverageCompressionRatio float64 // 平均压缩比例
	AverageBufferLifetime   time.Duration // 平均缓冲区生命周期
	HitRate                 float64 // 命中率
	
	// 性能统计
	LastCleanupTime     time.Time     // 最后清理时间
	CleanupFrequency    time.Duration // 清理频率
	MemoryUsage         int64         // 内存使用量
}

// NewGlobalBufferManager 创建全局缓冲区管理器
func NewGlobalBufferManager(strategy GlobalBufferStrategy) *GlobalBufferManager {
	manager := &GlobalBufferManager{
		strategy:          strategy,
		maxBuffers:        50,  // 最大50个缓冲区
		defaultBufferSize: 100, // 默认100个操作
		buffers:           make(map[string]*GlobalBuffer),
		shutdownChan:      make(chan struct{}),
		stats: &GlobalBufferStats{
			LastCleanupTime: time.Now(),
		},
	}
	
	// 初始化组件
	manager.patternAnalyzer = NewSearchPatternAnalyzer()
	manager.dataMerger = NewAdvancedDataMerger()
	manager.statusMonitor = NewBufferStatusMonitor()
	
	return manager
}

// Initialize 初始化管理器
func (g *GlobalBufferManager) Initialize() error {
	if !atomic.CompareAndSwapInt32(&g.initialized, 0, 1) {
		return nil // 已经初始化
	}
	
	// 启动定期清理
	g.cleanupTicker = time.NewTicker(5 * time.Minute) // 每5分钟清理一次
	go g.cleanupRoutine()
	
	// 启动状态监控
	go g.statusMonitor.Start(g)
	
	fmt.Printf("🚀 [全局缓冲区管理器] 初始化完成，策略: %s\n", g.strategy)
	return nil
}

// AddOperation 添加操作到全局缓冲区
func (g *GlobalBufferManager) AddOperation(op *CacheOperation) (*GlobalBuffer, bool, error) {
	if err := g.Initialize(); err != nil {
		return nil, false, err
	}
	
	// 🎯 根据策略确定缓冲区ID
	bufferID := g.determineBufferID(op)
	
	g.buffersMutex.Lock()
	defer g.buffersMutex.Unlock()
	
	// 🔧 获取或创建缓冲区
	buffer, exists := g.buffers[bufferID]
	if !exists {
		buffer = g.createNewBuffer(bufferID, op)
		g.buffers[bufferID] = buffer
		atomic.AddInt64(&g.stats.TotalBuffersCreated, 1)
		atomic.AddInt64(&g.stats.ActiveBuffers, 1)
	}
	
	// 🚀 添加操作到缓冲区
	shouldFlush := g.addOperationToBuffer(buffer, op)
	
	// 📊 更新统计
	atomic.AddInt64(&g.stats.TotalOperationsBuffered, 1)
	
	return buffer, shouldFlush, nil
}

// determineBufferID 确定缓冲区ID
func (g *GlobalBufferManager) determineBufferID(op *CacheOperation) string {
	switch g.strategy {
	case BufferByKeyword:
		return fmt.Sprintf("keyword_%s", op.Keyword)
		
	case BufferByPlugin:
		return fmt.Sprintf("plugin_%s", op.PluginName)
		
	case BufferByPattern:
		pattern := g.patternAnalyzer.AnalyzePattern(op)
		return fmt.Sprintf("pattern_%s", pattern.KeywordPattern)
		
	case BufferHybrid:
		// 混合策略：关键词+插件+时间窗口
		timeWindow := op.Timestamp.Truncate(time.Minute) // 1分钟时间窗口
		return fmt.Sprintf("hybrid_%s_%s_%d", 
			op.Keyword, op.PluginName, timeWindow.Unix())
			
	default:
		return fmt.Sprintf("default_%s", op.Key)
	}
}

// createNewBuffer 创建新缓冲区
func (g *GlobalBufferManager) createNewBuffer(bufferID string, firstOp *CacheOperation) *GlobalBuffer {
	now := time.Now()
	
	buffer := &GlobalBuffer{
		ID:               bufferID,
		Strategy:         g.strategy,
		CreatedAt:        now,
		LastUpdatedAt:    now,
		Operations:       make([]*CacheOperation, 0, g.defaultBufferSize),
		KeywordGroups:    make(map[string][]*CacheOperation),
		PluginGroups:     make(map[string][]*CacheOperation),
		MaxOperations:    g.defaultBufferSize,
		MaxDataSize:      int64(g.defaultBufferSize * 1000), // 估算100KB
		MaxAge:           10 * time.Minute, // 10分钟最大存活时间
	}
	
	return buffer
}

// addOperationToBuffer 添加操作到缓冲区
func (g *GlobalBufferManager) addOperationToBuffer(buffer *GlobalBuffer, op *CacheOperation) bool {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()
	
	// 🔧 数据合并优化
	merged := g.dataMerger.TryMergeOperation(buffer, op)
	if merged {
		atomic.AddInt64(&g.stats.TotalOperationsMerged, 1)
		atomic.AddInt64(&g.stats.TotalDataMerged, int64(op.DataSize))
	} else {
		// 添加新操作
		buffer.Operations = append(buffer.Operations, op)
		buffer.TotalOperations++
		buffer.TotalDataSize += int64(op.DataSize)
		
		// 按关键词分组
		if buffer.KeywordGroups[op.Keyword] == nil {
			buffer.KeywordGroups[op.Keyword] = make([]*CacheOperation, 0)
		}
		buffer.KeywordGroups[op.Keyword] = append(buffer.KeywordGroups[op.Keyword], op)
		
		// 按插件分组
		if buffer.PluginGroups[op.PluginName] == nil {
			buffer.PluginGroups[op.PluginName] = make([]*CacheOperation, 0)
		}
		buffer.PluginGroups[op.PluginName] = append(buffer.PluginGroups[op.PluginName], op)
	}
	
	buffer.LastUpdatedAt = time.Now()
	
	// 🎯 检查是否应该刷新
	return g.shouldFlushBuffer(buffer)
}

// shouldFlushBuffer 检查是否应该刷新缓冲区
func (g *GlobalBufferManager) shouldFlushBuffer(buffer *GlobalBuffer) bool {
	now := time.Now()
	
	// 条件1：操作数量达到阈值
	if len(buffer.Operations) >= buffer.MaxOperations {
		return true
	}
	
	// 条件2：数据大小达到阈值
	if buffer.TotalDataSize >= buffer.MaxDataSize {
		return true
	}
	
	// 条件3：缓冲区存活时间过长
	if now.Sub(buffer.CreatedAt) >= buffer.MaxAge {
		return true
	}
	
	// 条件4：内存压力（基于全局统计）
	totalMemory := atomic.LoadInt64(&g.stats.MemoryUsage)
	if totalMemory > 50*1024*1024 { // 50MB内存阈值
		return true
	}
	
	// 条件5：高优先级操作比例达到阈值
	highPriorityRatio := g.calculateHighPriorityRatio(buffer)
	if highPriorityRatio > 0.6 { // 60%高优先级阈值
		return true
	}
	
	return false
}

// calculateHighPriorityRatio 计算高优先级操作比例
func (g *GlobalBufferManager) calculateHighPriorityRatio(buffer *GlobalBuffer) float64 {
	if len(buffer.Operations) == 0 {
		return 0
	}
	
	highPriorityCount := 0
	for _, op := range buffer.Operations {
		if op.Priority <= 2 { // 等级1和等级2插件
			highPriorityCount++
		}
	}
	
	return float64(highPriorityCount) / float64(len(buffer.Operations))
}

// FlushBuffer 刷新指定缓冲区
func (g *GlobalBufferManager) FlushBuffer(bufferID string) ([]*CacheOperation, error) {
	g.buffersMutex.Lock()
	defer g.buffersMutex.Unlock()
	
	buffer, exists := g.buffers[bufferID]
	if !exists {
		return nil, fmt.Errorf("缓冲区不存在: %s", bufferID)
	}
	
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()
	
	// 获取所有操作
	operations := make([]*CacheOperation, len(buffer.Operations))
	copy(operations, buffer.Operations)
	
	// 清空缓冲区
	buffer.Operations = buffer.Operations[:0]
	buffer.KeywordGroups = make(map[string][]*CacheOperation)
	buffer.PluginGroups = make(map[string][]*CacheOperation)
	buffer.TotalOperations = 0
	buffer.TotalDataSize = 0
	
	// 更新压缩比例
	if len(operations) > 0 {
		buffer.CompressRatio = float64(len(operations)) / float64(buffer.TotalOperations)
	}
	
	return operations, nil
}

// FlushAllBuffers 刷新所有缓冲区
func (g *GlobalBufferManager) FlushAllBuffers() map[string][]*CacheOperation {
	g.buffersMutex.RLock()
	bufferIDs := make([]string, 0, len(g.buffers))
	for id := range g.buffers {
		bufferIDs = append(bufferIDs, id)
	}
	g.buffersMutex.RUnlock()
	
	result := make(map[string][]*CacheOperation)
	for _, id := range bufferIDs {
		if ops, err := g.FlushBuffer(id); err == nil && len(ops) > 0 {
			result[id] = ops
		}
	}
	
	return result
}

// cleanupRoutine 清理例程
func (g *GlobalBufferManager) cleanupRoutine() {
	for {
		select {
		case <-g.cleanupTicker.C:
			g.performCleanup()
			
		case <-g.shutdownChan:
			g.cleanupTicker.Stop()
			return
		}
	}
}

// performCleanup 执行清理
func (g *GlobalBufferManager) performCleanup() {
	now := time.Now()
	
	g.buffersMutex.Lock()
	defer g.buffersMutex.Unlock()
	
	toDelete := make([]string, 0)
	
	for id, buffer := range g.buffers {
		buffer.mutex.RLock()
		
		// 清理条件：空缓冲区且超过5分钟未活动
		if len(buffer.Operations) == 0 && now.Sub(buffer.LastUpdatedAt) > 5*time.Minute {
			toDelete = append(toDelete, id)
		}
		
		buffer.mutex.RUnlock()
	}
	
	// 删除过期缓冲区
	for _, id := range toDelete {
		delete(g.buffers, id)
		atomic.AddInt64(&g.stats.TotalBuffersDestroyed, 1)
		atomic.AddInt64(&g.stats.ActiveBuffers, -1)
	}
	
	// 更新清理统计
	g.stats.LastCleanupTime = now
	g.stats.CleanupFrequency = now.Sub(g.stats.LastCleanupTime)
	
	// 计算内存使用量
	g.updateMemoryUsage()
	
	if len(toDelete) > 0 {
		fmt.Printf("🧹 [全局缓冲区] 清理完成，删除%d个过期缓冲区\n", len(toDelete))
	}
}

// updateMemoryUsage 更新内存使用量估算
func (g *GlobalBufferManager) updateMemoryUsage() {
	totalMemory := int64(0)
	
	for _, buffer := range g.buffers {
		buffer.mutex.RLock()
		totalMemory += buffer.TotalDataSize
		buffer.mutex.RUnlock()
	}
	
	atomic.StoreInt64(&g.stats.MemoryUsage, totalMemory)
}

// Shutdown 优雅关闭
func (g *GlobalBufferManager) Shutdown() error {
	if !atomic.CompareAndSwapInt32(&g.initialized, 1, 0) {
		return nil // 已经关闭
	}
	
	// 停止后台任务
	close(g.shutdownChan)
	
	// 刷新所有缓冲区
	flushedBuffers := g.FlushAllBuffers()
	totalOperations := 0
	for _, ops := range flushedBuffers {
		totalOperations += len(ops)
	}
	
	fmt.Printf("🔄 [全局缓冲区管理器] 关闭完成，刷新%d个缓冲区，%d个操作\n", 
		len(flushedBuffers), totalOperations)
	
	return nil
}

// GetStats 获取统计信息
func (g *GlobalBufferManager) GetStats() *GlobalBufferStats {
	stats := *g.stats
	stats.ActiveBuffers = atomic.LoadInt64(&g.stats.ActiveBuffers)
	stats.MemoryUsage = atomic.LoadInt64(&g.stats.MemoryUsage)
	
	// 计算平均压缩比例
	if stats.TotalOperationsBuffered > 0 {
		stats.AverageCompressionRatio = float64(stats.TotalOperationsMerged) / float64(stats.TotalOperationsBuffered)
	}
	
	// 计算命中率
	if stats.TotalOperationsBuffered > 0 {
		stats.HitRate = float64(stats.TotalOperationsMerged) / float64(stats.TotalOperationsBuffered)
	}
	
	return &stats
}

// GetBufferInfo 获取缓冲区信息
func (g *GlobalBufferManager) GetBufferInfo() map[string]interface{} {
	g.buffersMutex.RLock()
	defer g.buffersMutex.RUnlock()
	
	info := make(map[string]interface{})
	
	for id, buffer := range g.buffers {
		buffer.mutex.RLock()
		bufferInfo := map[string]interface{}{
			"id":               id,
			"strategy":         buffer.Strategy,
			"created_at":       buffer.CreatedAt,
			"last_updated_at":  buffer.LastUpdatedAt,
			"total_operations": buffer.TotalOperations,
			"total_data_size":  buffer.TotalDataSize,
			"compress_ratio":   buffer.CompressRatio,
			"keyword_groups":   len(buffer.KeywordGroups),
			"plugin_groups":    len(buffer.PluginGroups),
		}
		buffer.mutex.RUnlock()
		
		info[id] = bufferInfo
	}
	
	return info
}