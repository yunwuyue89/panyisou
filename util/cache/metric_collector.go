package cache

import (
	"runtime"
	"sync/atomic"
	"time"
)

// NewMetricCollector 创建指标收集器
func NewMetricCollector() *MetricCollector {
	return &MetricCollector{
		systemMetrics:      &SystemMetrics{},
		applicationMetrics: &ApplicationMetrics{},
		cacheMetrics:       &CacheMetrics{},
		metricsHistory:     make([]MetricSnapshot, 0),
		maxHistorySize:     1000, // 保留1000个历史快照
		collectionChan:     make(chan struct{}),
	}
}

// Start 启动指标收集
func (m *MetricCollector) Start(interval time.Duration) error {
	if !atomic.CompareAndSwapInt32(&m.isCollecting, 0, 1) {
		return nil // 已经在收集中
	}
	
	go m.collectionLoop(interval)
	return nil
}

// Stop 停止指标收集
func (m *MetricCollector) Stop() {
	if atomic.CompareAndSwapInt32(&m.isCollecting, 1, 0) {
		close(m.collectionChan)
	}
}

// collectionLoop 收集循环
func (m *MetricCollector) collectionLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			m.collectMetrics()
			
		case <-m.collectionChan:
			return
		}
	}
}

// collectMetrics 收集指标
func (m *MetricCollector) collectMetrics() {
	now := time.Now()
	
	// 收集系统指标
	systemMetrics := m.collectSystemMetrics(now)
	
	// 收集应用指标
	applicationMetrics := m.collectApplicationMetrics(now)
	
	// 收集缓存指标
	cacheMetrics := m.collectCacheMetrics(now)
	
	// 创建快照
	snapshot := MetricSnapshot{
		Timestamp:   now,
		System:      *systemMetrics,
		Application: *applicationMetrics,
		Cache:       *cacheMetrics,
	}
	
	// 计算综合指标
	snapshot.OverallPerformance = m.calculateOverallPerformance(&snapshot)
	snapshot.Efficiency = m.calculateEfficiency(&snapshot)
	snapshot.Stability = m.calculateStability(&snapshot)
	
	// 保存快照
	m.saveSnapshot(snapshot)
}

// collectSystemMetrics 收集系统指标
func (m *MetricCollector) collectSystemMetrics(timestamp time.Time) *SystemMetrics {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	return &SystemMetrics{
		Timestamp:       timestamp,
		CPUUsage:        float64(memStats.GCCPUFraction),
		MemoryUsage:     int64(memStats.Alloc),
		MemoryTotal:     int64(memStats.Sys),
		DiskIORate:      0, // 简化实现
		NetworkIORate:   0, // 简化实现
		GoroutineCount:  runtime.NumGoroutine(),
		GCPauseDuration: time.Duration(memStats.PauseTotalNs),
		HeapSize:        int64(memStats.HeapSys),
		AllocRate:       float64(memStats.Mallocs - memStats.Frees),
	}
}

// collectApplicationMetrics 收集应用指标
func (m *MetricCollector) collectApplicationMetrics(timestamp time.Time) *ApplicationMetrics {
	// 简化实现，实际应用中应该从应用监控系统获取
	return &ApplicationMetrics{
		Timestamp:       timestamp,
		RequestRate:     100.0, // 模拟值
		ResponseTime:    50 * time.Millisecond,
		ErrorRate:       0.01,
		ThroughputMBps:  10.5,
		ConcurrentUsers: 50,
		QueueDepth:      5,
		ProcessingRate:  95.5,
	}
}

// collectCacheMetrics 收集缓存指标
func (m *MetricCollector) collectCacheMetrics(timestamp time.Time) *CacheMetrics {
	// 简化实现，实际应用中应该从缓存系统获取
	return &CacheMetrics{
		Timestamp:         timestamp,
		HitRate:           0.85,
		WriteRate:         20.0,
		ReadRate:          80.0,
		EvictionRate:      2.0,
		CompressionRatio:  0.6,
		StorageUsage:      1024 * 1024 * 100, // 100MB
		BufferUtilization: 0.75,
		BatchEfficiency:   0.9,
	}
}

// calculateOverallPerformance 计算整体性能
func (m *MetricCollector) calculateOverallPerformance(snapshot *MetricSnapshot) float64 {
	// 性能评分算法
	cpuScore := (1.0 - snapshot.System.CPUUsage) * 30
	memoryScore := (1.0 - float64(snapshot.System.MemoryUsage)/float64(snapshot.System.MemoryTotal)) * 25
	responseScore := (1.0 - float64(snapshot.Application.ResponseTime)/float64(time.Second)) * 25
	cacheScore := snapshot.Cache.HitRate * 20
	
	return cpuScore + memoryScore + responseScore + cacheScore
}

// calculateEfficiency 计算效率
func (m *MetricCollector) calculateEfficiency(snapshot *MetricSnapshot) float64 {
	// 效率评分算法
	cacheEfficiency := snapshot.Cache.HitRate * 0.4
	batchEfficiency := snapshot.Cache.BatchEfficiency * 0.3
	compressionEfficiency := snapshot.Cache.CompressionRatio * 0.3
	
	return cacheEfficiency + batchEfficiency + compressionEfficiency
}

// calculateStability 计算稳定性
func (m *MetricCollector) calculateStability(snapshot *MetricSnapshot) float64 {
	// 稳定性评分算法（基于变化率）
	errorRateStability := (1.0 - snapshot.Application.ErrorRate) * 0.5
	responseTimeStability := 1.0 - (float64(snapshot.Application.ResponseTime) / float64(time.Second))
	if responseTimeStability < 0 {
		responseTimeStability = 0
	}
	responseTimeStability *= 0.5
	
	return errorRateStability + responseTimeStability
}

// saveSnapshot 保存快照
func (m *MetricCollector) saveSnapshot(snapshot MetricSnapshot) {
	m.historyMutex.Lock()
	defer m.historyMutex.Unlock()
	
	m.metricsHistory = append(m.metricsHistory, snapshot)
	
	// 限制历史记录大小
	if len(m.metricsHistory) > m.maxHistorySize {
		m.metricsHistory = m.metricsHistory[1:]
	}
}

// GetLatestMetrics 获取最新指标
func (m *MetricCollector) GetLatestMetrics() *MetricSnapshot {
	m.historyMutex.RLock()
	defer m.historyMutex.RUnlock()
	
	if len(m.metricsHistory) == 0 {
		return nil
	}
	
	latest := m.metricsHistory[len(m.metricsHistory)-1]
	return &latest
}

// GetMetricsHistory 获取指标历史
func (m *MetricCollector) GetMetricsHistory(limit int) []MetricSnapshot {
	m.historyMutex.RLock()
	defer m.historyMutex.RUnlock()
	
	if limit <= 0 || limit > len(m.metricsHistory) {
		limit = len(m.metricsHistory)
	}
	
	history := make([]MetricSnapshot, limit)
	startIndex := len(m.metricsHistory) - limit
	copy(history, m.metricsHistory[startIndex:])
	
	return history
}