package cache

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	
	"pansou/util/json"
)

// BufferStatusMonitor 缓冲区状态监控器
type BufferStatusMonitor struct {
	// 监控配置
	monitorInterval    time.Duration
	alertThresholds    *AlertThresholds
	
	// 监控状态
	isMonitoring       int32
	shutdownChan       chan struct{}
	
	// 健康检查
	healthChecker      *HealthChecker
	
	// 报警系统
	alertManager       *AlertManager
	
	// 性能指标
	performanceMetrics *PerformanceMetrics
	
	// 监控数据
	monitoringData     *MonitoringData
	dataMutex          sync.RWMutex
	
	// 历史记录
	historyBuffer      []MonitorSnapshot
	historyMutex       sync.Mutex
	maxHistorySize     int
}

// AlertThresholds 报警阈值
type AlertThresholds struct {
	// 内存阈值
	MemoryUsageWarning   int64   // 内存使用警告阈值（字节）
	MemoryUsageCritical  int64   // 内存使用严重阈值（字节）
	
	// 缓冲区阈值
	BufferCountWarning   int     // 缓冲区数量警告阈值
	BufferCountCritical  int     // 缓冲区数量严重阈值
	
	// 操作阈值
	OperationQueueWarning  int   // 操作队列警告阈值
	OperationQueueCritical int   // 操作队列严重阈值
	
	// 时间阈值
	ProcessTimeWarning     time.Duration // 处理时间警告阈值
	ProcessTimeCritical    time.Duration // 处理时间严重阈值
	
	// 成功率阈值
	SuccessRateWarning     float64 // 成功率警告阈值
	SuccessRateCritical    float64 // 成功率严重阈值
}

// HealthChecker 健康检查器
type HealthChecker struct {
	lastHealthCheck    time.Time
	healthCheckInterval time.Duration
	healthStatus       HealthStatus
	healthHistory      []HealthCheckResult
	mutex              sync.RWMutex
}

// HealthStatus 健康状态
type HealthStatus struct {
	Overall            string    `json:"overall"`             // healthy, warning, critical
	LastCheck          time.Time `json:"last_check"`
	Components         map[string]ComponentHealth `json:"components"`
	Issues             []HealthIssue `json:"issues,omitempty"`
}

// ComponentHealth 组件健康状态
type ComponentHealth struct {
	Status      string                 `json:"status"`
	LastCheck   time.Time             `json:"last_check"`
	Metrics     map[string]interface{} `json:"metrics"`
	Message     string                `json:"message,omitempty"`
}

// HealthIssue 健康问题
type HealthIssue struct {
	Component   string    `json:"component"`
	Severity    string    `json:"severity"`    // warning, critical
	Message     string    `json:"message"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	Count       int       `json:"count"`
}

// HealthCheckResult 健康检查结果
type HealthCheckResult struct {
	Timestamp   time.Time     `json:"timestamp"`
	Status      string        `json:"status"`
	CheckTime   time.Duration `json:"check_time"`
	Issues      []HealthIssue `json:"issues"`
}

// AlertManager 报警管理器
type AlertManager struct {
	alerts          []Alert
	alertHistory    []Alert
	mutex           sync.RWMutex
	maxAlertHistory int
	
	// 报警配置
	alertCooldown   map[string]time.Time // 报警冷却时间
	cooldownPeriod  time.Duration        // 冷却期间
}

// Alert 报警
type Alert struct {
	ID          string                 `json:"id"`
	Level       string                 `json:"level"`       // info, warning, critical
	Component   string                 `json:"component"`
	Message     string                 `json:"message"`
	Timestamp   time.Time             `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Resolved    bool                  `json:"resolved"`
	ResolvedAt  *time.Time            `json:"resolved_at,omitempty"`
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	// CPU指标
	CPUUsage        float64   `json:"cpu_usage"`
	CPUHistory      []float64 `json:"cpu_history"`
	
	// 内存指标
	MemoryUsage     int64     `json:"memory_usage"`
	MemoryHistory   []int64   `json:"memory_history"`
	GCStats         runtime.MemStats `json:"gc_stats"`
	
	// 吞吐量指标
	OperationsPerSecond float64 `json:"operations_per_second"`
	ThroughputHistory   []float64 `json:"throughput_history"`
	
	// 延迟指标
	AverageLatency      time.Duration `json:"average_latency"`
	P95Latency          time.Duration `json:"p95_latency"`
	P99Latency          time.Duration `json:"p99_latency"`
	LatencyHistory      []time.Duration `json:"latency_history"`
	
	// 错误率指标
	ErrorRate           float64   `json:"error_rate"`
	ErrorHistory        []float64 `json:"error_history"`
	
	// 资源利用率
	DiskIORate          float64   `json:"disk_io_rate"`
	NetworkIORate       float64   `json:"network_io_rate"`
	
	// 更新时间
	LastUpdated         time.Time `json:"last_updated"`
}

// MonitoringData 监控数据
type MonitoringData struct {
	// 系统状态
	SystemHealth       HealthStatus      `json:"system_health"`
	PerformanceMetrics PerformanceMetrics `json:"performance_metrics"`
	
	// 缓冲区状态
	BufferStates       map[string]BufferState `json:"buffer_states"`
	GlobalBufferStats  *GlobalBufferStats     `json:"global_buffer_stats"`
	
	// 实时统计
	RealTimeStats      RealTimeStats     `json:"real_time_stats"`
	
	// 趋势分析
	TrendAnalysis      TrendAnalysis     `json:"trend_analysis"`
	
	// 预测数据
	Predictions        PredictionData    `json:"predictions"`
}

// BufferState 缓冲区状态
type BufferState struct {
	ID                 string        `json:"id"`
	Size               int           `json:"size"`
	Capacity           int           `json:"capacity"`
	UtilizationRate    float64       `json:"utilization_rate"`
	LastActivity       time.Time     `json:"last_activity"`
	OperationsPerMin   float64       `json:"operations_per_min"`
	AverageDataSize    int64         `json:"average_data_size"`
	CompressionRatio   float64       `json:"compression_ratio"`
	Health             string        `json:"health"`
}

// RealTimeStats 实时统计
type RealTimeStats struct {
	ActiveOperations   int     `json:"active_operations"`
	QueuedOperations   int     `json:"queued_operations"`
	ProcessingRate     float64 `json:"processing_rate"`
	ThroughputMBps     float64 `json:"throughput_mbps"`
	CacheHitRate       float64 `json:"cache_hit_rate"`
	CompressionRatio   float64 `json:"compression_ratio"`
	ErrorRate          float64 `json:"error_rate"`
	LastUpdated        time.Time `json:"last_updated"`
}

// TrendAnalysis 趋势分析
type TrendAnalysis struct {
	MemoryTrend        string    `json:"memory_trend"`        // increasing, decreasing, stable
	ThroughputTrend    string    `json:"throughput_trend"`
	ErrorRateTrend     string    `json:"error_rate_trend"`
	BufferUsageTrend   string    `json:"buffer_usage_trend"`
	AnalysisTime       time.Time `json:"analysis_time"`
	Confidence         float64   `json:"confidence"`
}

// PredictionData 预测数据
type PredictionData struct {
	MemoryUsageIn1Hour     int64     `json:"memory_usage_in_1hour"`
	MemoryUsageIn24Hours   int64     `json:"memory_usage_in_24hours"`
	BufferOverflowRisk     float64   `json:"buffer_overflow_risk"`
	SystemLoadPrediction   float64   `json:"system_load_prediction"`
	RecommendedActions     []string  `json:"recommended_actions"`
	ConfidenceLevel        float64   `json:"confidence_level"`
	PredictionTime         time.Time `json:"prediction_time"`
}

// MonitorSnapshot 监控快照
type MonitorSnapshot struct {
	Timestamp          time.Time          `json:"timestamp"`
	SystemHealth       HealthStatus       `json:"system_health"`
	BufferCount        int               `json:"buffer_count"`
	TotalMemoryUsage   int64             `json:"total_memory_usage"`
	OperationsPerSecond float64          `json:"operations_per_second"`
	ErrorRate          float64           `json:"error_rate"`
	CacheHitRate       float64           `json:"cache_hit_rate"`
}

// NewBufferStatusMonitor 创建缓冲区状态监控器
func NewBufferStatusMonitor() *BufferStatusMonitor {
	monitor := &BufferStatusMonitor{
		monitorInterval: 30 * time.Second, // 30秒监控间隔
		shutdownChan:    make(chan struct{}),
		maxHistorySize:  288, // 保存24小时历史（每30秒一个，24*60*2=2880，简化为288）
		alertThresholds: &AlertThresholds{
			MemoryUsageWarning:     50 * 1024 * 1024,  // 50MB
			MemoryUsageCritical:    100 * 1024 * 1024, // 100MB
			BufferCountWarning:     30,
			BufferCountCritical:    50,
			OperationQueueWarning:  500,
			OperationQueueCritical: 1000,
			ProcessTimeWarning:     5 * time.Second,
			ProcessTimeCritical:    15 * time.Second,
			SuccessRateWarning:     0.95, // 95%
			SuccessRateCritical:    0.90, // 90%
		},
		monitoringData: &MonitoringData{
			BufferStates:   make(map[string]BufferState),
			RealTimeStats:  RealTimeStats{},
			TrendAnalysis:  TrendAnalysis{},
			Predictions:    PredictionData{},
		},
	}
	
	// 初始化组件
	monitor.healthChecker = &HealthChecker{
		healthCheckInterval: 1 * time.Minute,
		healthStatus: HealthStatus{
			Overall:    "healthy",
			Components: make(map[string]ComponentHealth),
			Issues:     make([]HealthIssue, 0),
		},
		healthHistory: make([]HealthCheckResult, 0),
	}
	
	monitor.alertManager = &AlertManager{
		alerts:          make([]Alert, 0),
		alertHistory:    make([]Alert, 0),
		maxAlertHistory: 1000,
		alertCooldown:   make(map[string]time.Time),
		cooldownPeriod:  5 * time.Minute, // 5分钟冷却期
	}
	
	monitor.performanceMetrics = &PerformanceMetrics{
		CPUHistory:        make([]float64, 0),
		MemoryHistory:     make([]int64, 0),
		ThroughputHistory: make([]float64, 0),
		LatencyHistory:    make([]time.Duration, 0),
		ErrorHistory:      make([]float64, 0),
	}
	
	return monitor
}

// Start 启动监控器
func (b *BufferStatusMonitor) Start(globalManager *GlobalBufferManager) {
	if !atomic.CompareAndSwapInt32(&b.isMonitoring, 0, 1) {
		return // 已经在监控中
	}
	
	fmt.Printf("🔍 [缓冲区状态监控器] 启动监控，间隔: %v\n", b.monitorInterval)
	
	go b.monitoringLoop(globalManager)
	go b.healthCheckLoop()
	go b.alertProcessingLoop()
}

// monitoringLoop 监控循环
func (b *BufferStatusMonitor) monitoringLoop(globalManager *GlobalBufferManager) {
	ticker := time.NewTicker(b.monitorInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			b.collectMetrics(globalManager)
			b.analyzeData()
			b.checkAlerts()
			b.updatePredictions()
			b.saveSnapshot()
			
		case <-b.shutdownChan:
			return
		}
	}
}

// healthCheckLoop 健康检查循环
func (b *BufferStatusMonitor) healthCheckLoop() {
	ticker := time.NewTicker(b.healthChecker.healthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			b.performHealthCheck()
			
		case <-b.shutdownChan:
			return
		}
	}
}

// alertProcessingLoop 报警处理循环
func (b *BufferStatusMonitor) alertProcessingLoop() {
	ticker := time.NewTicker(1 * time.Minute) // 每分钟检查一次报警
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			b.processAlerts()
			
		case <-b.shutdownChan:
			return
		}
	}
}

// collectMetrics 收集指标
func (b *BufferStatusMonitor) collectMetrics(globalManager *GlobalBufferManager) {
	b.dataMutex.Lock()
	defer b.dataMutex.Unlock()
	
	// 收集全局缓冲区统计
	b.monitoringData.GlobalBufferStats = globalManager.GetStats()
	
	// 收集缓冲区状态
	bufferInfo := globalManager.GetBufferInfo()
	for id, info := range bufferInfo {
		if infoMap, ok := info.(map[string]interface{}); ok {
			bufferState := BufferState{
				ID:           id,
				LastActivity: time.Now(),
				Health:       "healthy",
			}
			
			// 提取缓冲区信息
			if size, ok := infoMap["total_operations"].(int64); ok {
				bufferState.Size = int(size)
			}
			if dataSize, ok := infoMap["total_data_size"].(int64); ok {
				bufferState.AverageDataSize = dataSize
			}
			if ratio, ok := infoMap["compress_ratio"].(float64); ok {
				bufferState.CompressionRatio = ratio
			}
			
			b.monitoringData.BufferStates[id] = bufferState
		}
	}
	
	// 收集性能指标
	b.collectPerformanceMetrics()
	
	// 更新实时统计
	b.updateRealTimeStats()
}

// collectPerformanceMetrics 收集性能指标
func (b *BufferStatusMonitor) collectPerformanceMetrics() {
	// 收集内存统计
	runtime.ReadMemStats(&b.performanceMetrics.GCStats)
	
	currentMemory := int64(b.performanceMetrics.GCStats.Alloc)
	b.performanceMetrics.MemoryUsage = currentMemory
	
	// 更新内存历史
	b.performanceMetrics.MemoryHistory = append(b.performanceMetrics.MemoryHistory, currentMemory)
	if len(b.performanceMetrics.MemoryHistory) > 100 { // 保留最近100个数据点
		b.performanceMetrics.MemoryHistory = b.performanceMetrics.MemoryHistory[1:]
	}
	
	// 简化的CPU使用率估算（基于GC统计）
	gcCPUPercent := float64(b.performanceMetrics.GCStats.GCCPUFraction) * 100
	b.performanceMetrics.CPUUsage = gcCPUPercent
	
	// 更新CPU历史
	b.performanceMetrics.CPUHistory = append(b.performanceMetrics.CPUHistory, gcCPUPercent)
	if len(b.performanceMetrics.CPUHistory) > 100 {
		b.performanceMetrics.CPUHistory = b.performanceMetrics.CPUHistory[1:]
	}
	
	b.performanceMetrics.LastUpdated = time.Now()
}

// updateRealTimeStats 更新实时统计
func (b *BufferStatusMonitor) updateRealTimeStats() {
	stats := &b.monitoringData.RealTimeStats
	
	if b.monitoringData.GlobalBufferStats != nil {
		globalStats := b.monitoringData.GlobalBufferStats
		
		// 活跃操作数
		stats.ActiveOperations = int(globalStats.ActiveBuffers)
		
		// 处理速率（操作/秒）
		if globalStats.TotalOperationsBuffered > 0 {
			stats.ProcessingRate = float64(globalStats.TotalOperationsBuffered) / 
				time.Since(globalStats.LastCleanupTime).Seconds()
		}
		
		// 压缩比例
		stats.CompressionRatio = globalStats.AverageCompressionRatio
		
		// 缓存命中率
		stats.CacheHitRate = globalStats.HitRate
	}
	
	// 内存使用（MB/s）
	if b.performanceMetrics.MemoryUsage > 0 {
		stats.ThroughputMBps = float64(b.performanceMetrics.MemoryUsage) / 1024 / 1024
	}
	
	stats.LastUpdated = time.Now()
}

// analyzeData 分析数据
func (b *BufferStatusMonitor) analyzeData() {
	b.analyzeTrends()
	b.detectAnomalies()
}

// analyzeTrends 分析趋势
func (b *BufferStatusMonitor) analyzeTrends() {
	trends := &b.monitoringData.TrendAnalysis
	
	// 内存趋势分析
	if len(b.performanceMetrics.MemoryHistory) >= 3 {
		recent := b.performanceMetrics.MemoryHistory[len(b.performanceMetrics.MemoryHistory)-3:]
		if recent[2] > recent[1] && recent[1] > recent[0] {
			trends.MemoryTrend = "increasing"
		} else if recent[2] < recent[1] && recent[1] < recent[0] {
			trends.MemoryTrend = "decreasing"
		} else {
			trends.MemoryTrend = "stable"
		}
	}
	
	// 缓冲区使用趋势
	bufferCount := len(b.monitoringData.BufferStates)
	if bufferCount > b.alertThresholds.BufferCountWarning {
		trends.BufferUsageTrend = "increasing"
	} else {
		trends.BufferUsageTrend = "stable"
	}
	
	trends.AnalysisTime = time.Now()
	trends.Confidence = 0.8 // 简化的置信度
}

// detectAnomalies 检测异常
func (b *BufferStatusMonitor) detectAnomalies() {
	// 内存异常检测
	if b.performanceMetrics.MemoryUsage > b.alertThresholds.MemoryUsageCritical {
		b.triggerAlert("memory", "critical", 
			fmt.Sprintf("内存使用过高: %d bytes", b.performanceMetrics.MemoryUsage))
	} else if b.performanceMetrics.MemoryUsage > b.alertThresholds.MemoryUsageWarning {
		b.triggerAlert("memory", "warning", 
			fmt.Sprintf("内存使用警告: %d bytes", b.performanceMetrics.MemoryUsage))
	}
	
	// 缓冲区数量异常检测
	bufferCount := len(b.monitoringData.BufferStates)
	if bufferCount > b.alertThresholds.BufferCountCritical {
		b.triggerAlert("buffer_count", "critical", 
			fmt.Sprintf("缓冲区数量过多: %d", bufferCount))
	} else if bufferCount > b.alertThresholds.BufferCountWarning {
		b.triggerAlert("buffer_count", "warning", 
			fmt.Sprintf("缓冲区数量警告: %d", bufferCount))
	}
}

// checkAlerts 检查报警
func (b *BufferStatusMonitor) checkAlerts() {
	// 检查系统健康状态
	if b.healthChecker.healthStatus.Overall == "critical" {
		b.triggerAlert("system_health", "critical", "系统健康状态严重")
	} else if b.healthChecker.healthStatus.Overall == "warning" {
		b.triggerAlert("system_health", "warning", "系统健康状态警告")
	}
}

// triggerAlert 触发报警
func (b *BufferStatusMonitor) triggerAlert(component, level, message string) {
	alertKey := fmt.Sprintf("%s_%s", component, level)
	
	// 检查冷却期
	b.alertManager.mutex.Lock()
	if lastAlert, exists := b.alertManager.alertCooldown[alertKey]; exists {
		if time.Since(lastAlert) < b.alertManager.cooldownPeriod {
			b.alertManager.mutex.Unlock()
			return // 还在冷却期内
		}
	}
	
	// 创建新报警
	alert := Alert{
		ID:        fmt.Sprintf("%s_%d", alertKey, time.Now().Unix()),
		Level:     level,
		Component: component,
		Message:   message,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
		Resolved:  false,
	}
	
	// 添加相关指标作为元数据
	alert.Metadata["memory_usage"] = b.performanceMetrics.MemoryUsage
	alert.Metadata["buffer_count"] = len(b.monitoringData.BufferStates)
	alert.Metadata["cpu_usage"] = b.performanceMetrics.CPUUsage
	
	b.alertManager.alerts = append(b.alertManager.alerts, alert)
	b.alertManager.alertCooldown[alertKey] = time.Now()
	
	b.alertManager.mutex.Unlock()
	
	// 输出报警日志
	fmt.Printf("🚨 [报警] %s - %s: %s\n", level, component, message)
}

// updatePredictions 更新预测
func (b *BufferStatusMonitor) updatePredictions() {
	predictions := &b.monitoringData.Predictions
	
	// 简化的内存使用预测
	if len(b.performanceMetrics.MemoryHistory) >= 5 {
		history := b.performanceMetrics.MemoryHistory
		recent := history[len(history)-5:]
		
		// 简单线性预测
		growth := float64(recent[4]-recent[0]) / 4
		predictions.MemoryUsageIn1Hour = recent[4] + int64(growth*120)     // 2小时数据点预测1小时
		predictions.MemoryUsageIn24Hours = recent[4] + int64(growth*2880) // 预测24小时
	}
	
	// 缓冲区溢出风险评估
	bufferCount := len(b.monitoringData.BufferStates)
	if bufferCount > b.alertThresholds.BufferCountWarning {
		predictions.BufferOverflowRisk = float64(bufferCount) / float64(b.alertThresholds.BufferCountCritical)
	} else {
		predictions.BufferOverflowRisk = 0.1
	}
	
	// 推荐行动
	predictions.RecommendedActions = b.generateRecommendations()
	predictions.ConfidenceLevel = 0.7
	predictions.PredictionTime = time.Now()
}

// generateRecommendations 生成推荐
func (b *BufferStatusMonitor) generateRecommendations() []string {
	recommendations := make([]string, 0)
	
	// 基于内存使用推荐
	if b.performanceMetrics.MemoryUsage > b.alertThresholds.MemoryUsageWarning {
		recommendations = append(recommendations, "考虑增加内存或减少缓冲区大小")
	}
	
	// 基于缓冲区数量推荐
	bufferCount := len(b.monitoringData.BufferStates)
	if bufferCount > b.alertThresholds.BufferCountWarning {
		recommendations = append(recommendations, "考虑调整缓冲区清理频率")
	}
	
	// 基于趋势推荐
	if b.monitoringData.TrendAnalysis.MemoryTrend == "increasing" {
		recommendations = append(recommendations, "内存使用呈增长趋势，建议监控和优化")
	}
	
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "系统运行正常，继续监控")
	}
	
	return recommendations
}

// performHealthCheck 执行健康检查
func (b *BufferStatusMonitor) performHealthCheck() {
	startTime := time.Now()
	
	b.healthChecker.mutex.Lock()
	defer b.healthChecker.mutex.Unlock()
	
	health := &b.healthChecker.healthStatus
	health.LastCheck = time.Now()
	health.Issues = make([]HealthIssue, 0)
	
	// 检查内存健康
	memoryHealth := b.checkMemoryHealth()
	health.Components["memory"] = memoryHealth
	
	// 检查缓冲区健康
	bufferHealth := b.checkBufferHealth()
	health.Components["buffers"] = bufferHealth
	
	// 检查性能健康
	performanceHealth := b.checkPerformanceHealth()
	health.Components["performance"] = performanceHealth
	
	// 确定整体健康状态
	health.Overall = b.determineOverallHealth()
	
	// 记录健康检查结果
	checkResult := HealthCheckResult{
		Timestamp: time.Now(),
		Status:    health.Overall,
		CheckTime: time.Since(startTime),
		Issues:    health.Issues,
	}
	
	b.healthChecker.healthHistory = append(b.healthChecker.healthHistory, checkResult)
	if len(b.healthChecker.healthHistory) > 100 { // 保留最近100次检查
		b.healthChecker.healthHistory = b.healthChecker.healthHistory[1:]
	}
	
	b.healthChecker.lastHealthCheck = time.Now()
}

// checkMemoryHealth 检查内存健康
func (b *BufferStatusMonitor) checkMemoryHealth() ComponentHealth {
	health := ComponentHealth{
		Status:    "healthy",
		LastCheck: time.Now(),
		Metrics:   make(map[string]interface{}),
	}
	
	memUsage := b.performanceMetrics.MemoryUsage
	health.Metrics["usage_bytes"] = memUsage
	health.Metrics["usage_mb"] = memUsage / 1024 / 1024
	
	if memUsage > b.alertThresholds.MemoryUsageCritical {
		health.Status = "critical"
		health.Message = "内存使用严重过高"
	} else if memUsage > b.alertThresholds.MemoryUsageWarning {
		health.Status = "warning"
		health.Message = "内存使用偏高"
	} else {
		health.Message = "内存使用正常"
	}
	
	return health
}

// checkBufferHealth 检查缓冲区健康
func (b *BufferStatusMonitor) checkBufferHealth() ComponentHealth {
	health := ComponentHealth{
		Status:    "healthy",
		LastCheck: time.Now(),
		Metrics:   make(map[string]interface{}),
	}
	
	bufferCount := len(b.monitoringData.BufferStates)
	health.Metrics["buffer_count"] = bufferCount
	health.Metrics["max_buffers"] = b.alertThresholds.BufferCountCritical
	
	if bufferCount > b.alertThresholds.BufferCountCritical {
		health.Status = "critical"
		health.Message = "缓冲区数量过多"
	} else if bufferCount > b.alertThresholds.BufferCountWarning {
		health.Status = "warning"
		health.Message = "缓冲区数量偏高"
	} else {
		health.Message = "缓冲区状态正常"
	}
	
	return health
}

// checkPerformanceHealth 检查性能健康
func (b *BufferStatusMonitor) checkPerformanceHealth() ComponentHealth {
	health := ComponentHealth{
		Status:    "healthy",
		LastCheck: time.Now(),
		Metrics:   make(map[string]interface{}),
	}
	
	cpuUsage := b.performanceMetrics.CPUUsage
	health.Metrics["cpu_usage"] = cpuUsage
	health.Metrics["gc_cpu_fraction"] = b.performanceMetrics.GCStats.GCCPUFraction
	
	if cpuUsage > 80 {
		health.Status = "warning"
		health.Message = "CPU使用率偏高"
	} else {
		health.Message = "性能状态正常"
	}
	
	return health
}

// determineOverallHealth 确定整体健康状态
func (b *BufferStatusMonitor) determineOverallHealth() string {
	hasCritical := false
	hasWarning := false
	
	for _, component := range b.healthChecker.healthStatus.Components {
		switch component.Status {
		case "critical":
			hasCritical = true
		case "warning":
			hasWarning = true
		}
	}
	
	if hasCritical {
		return "critical"
	} else if hasWarning {
		return "warning"
	}
	
	return "healthy"
}

// processAlerts 处理报警
func (b *BufferStatusMonitor) processAlerts() {
	b.alertManager.mutex.Lock()
	defer b.alertManager.mutex.Unlock()
	
	// 检查是否有报警需要自动解决
	for i := range b.alertManager.alerts {
		alert := &b.alertManager.alerts[i]
		if !alert.Resolved {
			if b.shouldResolveAlert(alert) {
				now := time.Now()
				alert.Resolved = true
				alert.ResolvedAt = &now
				
				fmt.Printf("✅ [报警解决] %s - %s: %s\n", 
					alert.Level, alert.Component, alert.Message)
			}
		}
	}
	
	// 移动已解决的报警到历史记录
	activeAlerts := make([]Alert, 0)
	for _, alert := range b.alertManager.alerts {
		if !alert.Resolved {
			activeAlerts = append(activeAlerts, alert)
		} else {
			b.alertManager.alertHistory = append(b.alertManager.alertHistory, alert)
		}
	}
	
	b.alertManager.alerts = activeAlerts
	
	// 限制历史记录大小
	if len(b.alertManager.alertHistory) > b.alertManager.maxAlertHistory {
		excess := len(b.alertManager.alertHistory) - b.alertManager.maxAlertHistory
		b.alertManager.alertHistory = b.alertManager.alertHistory[excess:]
	}
}

// shouldResolveAlert 检查是否应该解决报警
func (b *BufferStatusMonitor) shouldResolveAlert(alert *Alert) bool {
	switch alert.Component {
	case "memory":
		return b.performanceMetrics.MemoryUsage < b.alertThresholds.MemoryUsageWarning
	case "buffer_count":
		return len(b.monitoringData.BufferStates) < b.alertThresholds.BufferCountWarning
	case "system_health":
		return b.healthChecker.healthStatus.Overall == "healthy"
	}
	
	return false
}

// saveSnapshot 保存监控快照
func (b *BufferStatusMonitor) saveSnapshot() {
	b.historyMutex.Lock()
	defer b.historyMutex.Unlock()
	
	snapshot := MonitorSnapshot{
		Timestamp:           time.Now(),
		SystemHealth:        b.healthChecker.healthStatus,
		BufferCount:         len(b.monitoringData.BufferStates),
		TotalMemoryUsage:    b.performanceMetrics.MemoryUsage,
		OperationsPerSecond: b.monitoringData.RealTimeStats.ProcessingRate,
		ErrorRate:           b.monitoringData.RealTimeStats.ErrorRate,
		CacheHitRate:        b.monitoringData.RealTimeStats.CacheHitRate,
	}
	
	b.historyBuffer = append(b.historyBuffer, snapshot)
	
	// 限制历史记录大小
	if len(b.historyBuffer) > b.maxHistorySize {
		b.historyBuffer = b.historyBuffer[1:]
	}
}

// Stop 停止监控器
func (b *BufferStatusMonitor) Stop() {
	if !atomic.CompareAndSwapInt32(&b.isMonitoring, 1, 0) {
		return
	}
	
	close(b.shutdownChan)
	fmt.Printf("🔍 [缓冲区状态监控器] 已停止监控\n")
}

// GetMonitoringData 获取监控数据
func (b *BufferStatusMonitor) GetMonitoringData() *MonitoringData {
	b.dataMutex.RLock()
	defer b.dataMutex.RUnlock()
	
	// 深拷贝监控数据
	dataCopy := *b.monitoringData
	return &dataCopy
}

// GetHealthStatus 获取健康状态
func (b *BufferStatusMonitor) GetHealthStatus() HealthStatus {
	b.healthChecker.mutex.RLock()
	defer b.healthChecker.mutex.RUnlock()
	
	return b.healthChecker.healthStatus
}

// GetActiveAlerts 获取活跃报警
func (b *BufferStatusMonitor) GetActiveAlerts() []Alert {
	b.alertManager.mutex.RLock()
	defer b.alertManager.mutex.RUnlock()
	
	alerts := make([]Alert, len(b.alertManager.alerts))
	copy(alerts, b.alertManager.alerts)
	return alerts
}

// GetMonitorHistory 获取监控历史
func (b *BufferStatusMonitor) GetMonitorHistory(limit int) []MonitorSnapshot {
	b.historyMutex.Lock()
	defer b.historyMutex.Unlock()
	
	if limit <= 0 || limit > len(b.historyBuffer) {
		limit = len(b.historyBuffer)
	}
	
	history := make([]MonitorSnapshot, limit)
	startIndex := len(b.historyBuffer) - limit
	copy(history, b.historyBuffer[startIndex:])
	
	return history
}

// ExportMonitoringReport 导出监控报告
func (b *BufferStatusMonitor) ExportMonitoringReport() (string, error) {
	report := map[string]interface{}{
		"timestamp":        time.Now(),
		"monitoring_data":  b.GetMonitoringData(),
		"health_status":    b.GetHealthStatus(),
		"active_alerts":    b.GetActiveAlerts(),
		"performance_metrics": b.performanceMetrics,
		"recent_history":   b.GetMonitorHistory(50), // 最近50个快照
	}
	
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("导出监控报告失败: %v", err)
	}
	
	return string(jsonData), nil
}