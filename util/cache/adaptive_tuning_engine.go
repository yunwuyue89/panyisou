package cache

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// AdaptiveTuningEngine 自适应调优引擎
type AdaptiveTuningEngine struct {
	// 核心组件
	metricCollector     *MetricCollector
	performanceAnalyzer *PerformanceAnalyzer
	predictiveModel     *PredictiveModel
	tuningStrategy      *TuningStrategy
	
	// 配置参数
	config              *AdaptiveTuningConfig
	
	// 运行状态
	isRunning           int32
	shutdownChan        chan struct{}
	
	// 调优历史
	tuningHistory       []*TuningRecord
	historyMutex        sync.RWMutex
	maxHistorySize      int
	
	// 学习数据
	learningData        *LearningDataset
	
	// 统计信息
	stats               *TuningEngineStats
	
	mutex               sync.RWMutex
}

// AdaptiveTuningConfig 自适应调优配置
type AdaptiveTuningConfig struct {
	// 调优间隔
	TuningInterval      time.Duration
	MetricInterval      time.Duration
	
	// 性能阈值
	CPUUsageThreshold   float64
	MemoryThreshold     int64
	LatencyThreshold    time.Duration
	
	// 学习参数
	LearningRate        float64
	AdaptationSpeed     float64
	StabilityFactor     float64
	
	// 调优范围
	MinBatchInterval    time.Duration
	MaxBatchInterval    time.Duration
	MinBatchSize        int
	MaxBatchSize        int
	
	// 安全参数
	MaxAdjustmentRatio  float64  // 最大调整幅度
	RollbackThreshold   float64  // 回滚阈值
	
	// 预测参数
	PredictionWindow    time.Duration
	ConfidenceThreshold float64
}

// MetricCollector 指标收集器
type MetricCollector struct {
	// 系统指标
	systemMetrics       *SystemMetrics
	
	// 应用指标
	applicationMetrics  *ApplicationMetrics
	
	// 缓存指标
	cacheMetrics        *CacheMetrics
	
	// 历史数据
	metricsHistory      []MetricSnapshot
	historyMutex        sync.RWMutex
	maxHistorySize      int
	
	// 采集状态
	isCollecting        int32
	collectionChan      chan struct{}
}

// SystemMetrics 系统指标
type SystemMetrics struct {
	Timestamp           time.Time
	CPUUsage            float64
	MemoryUsage         int64
	MemoryTotal         int64
	DiskIORate          float64
	NetworkIORate       float64
	GoroutineCount      int
	GCPauseDuration     time.Duration
	HeapSize            int64
	AllocRate           float64
}

// ApplicationMetrics 应用指标
type ApplicationMetrics struct {
	Timestamp           time.Time
	RequestRate         float64
	ResponseTime        time.Duration
	ErrorRate           float64
	ThroughputMBps      float64
	ConcurrentUsers     int
	QueueDepth          int
	ProcessingRate      float64
}

// CacheMetrics 缓存指标
type CacheMetrics struct {
	Timestamp           time.Time
	HitRate             float64
	WriteRate           float64
	ReadRate            float64
	EvictionRate        float64
	CompressionRatio    float64
	StorageUsage        int64
	BufferUtilization   float64
	BatchEfficiency     float64
}

// MetricSnapshot 指标快照
type MetricSnapshot struct {
	Timestamp           time.Time
	System              SystemMetrics
	Application         ApplicationMetrics
	Cache               CacheMetrics
	
	// 综合指标
	OverallPerformance  float64
	Efficiency          float64
	Stability           float64
}

// PerformanceAnalyzer 性能分析器
type PerformanceAnalyzer struct {
	// 分析算法
	trendAnalyzer       *TrendAnalyzer
	anomalyDetector     *AnomalyDetector
	correlationAnalyzer *CorrelationAnalyzer
	
	// 分析结果
	currentTrends       map[string]Trend
	detectedAnomalies   []Anomaly
	correlations        map[string]float64
	
	mutex               sync.RWMutex
}

// Trend 趋势
type Trend struct {
	Metric              string
	Direction           string  // increasing, decreasing, stable
	Slope               float64
	Confidence          float64
	Duration            time.Duration
	Prediction          float64
}

// Anomaly 异常
type Anomaly struct {
	Metric              string
	Timestamp           time.Time
	Severity            string  // low, medium, high
	Value               float64
	ExpectedRange       [2]float64
	Description         string
	Impact              float64
}

// PredictiveModel 预测模型
type PredictiveModel struct {
	// 模型类型
	modelType           string // linear_regression, exponential_smoothing, arima
	
	// 模型参数
	coefficients        []float64
	seasonalFactors     []float64
	trendComponent      float64
	
	// 训练数据
	trainingData        []DataPoint
	testData            []DataPoint
	
	// 模型性能
	accuracy            float64
	rmse                float64
	mae                 float64
	
	// 预测结果
	predictions         map[string]Prediction
	
	mutex               sync.RWMutex
}

// DataPoint 数据点
type DataPoint struct {
	Timestamp           time.Time
	Values              map[string]float64
	Label               string
}

// Prediction 预测
type Prediction struct {
	Metric              string
	FutureValue         float64
	Confidence          float64
	TimeHorizon         time.Duration
	PredictedAt         time.Time
	ActualValue         *float64  // 用于验证预测准确性
}

// TuningStrategy 调优策略
type TuningStrategy struct {
	// 策略类型
	strategyType        string // conservative, aggressive, balanced
	
	// 调优规则
	rules               []*TuningRule
	
	// 参数调整
	parameterAdjustments map[string]ParameterAdjustment
	
	// 执行历史
	executionHistory    []*StrategyExecution
	
	mutex               sync.RWMutex
}

// TuningRule 调优规则
type TuningRule struct {
	Name                string
	Condition           func(*MetricSnapshot) bool
	Action              func(*AdaptiveTuningEngine) (*TuningDecision, error)
	Priority            int
	Enabled             bool
	LastTriggered       time.Time
	TriggerCount        int64
}

// ParameterAdjustment 参数调整
type ParameterAdjustment struct {
	ParameterName       string
	CurrentValue        interface{}
	ProposedValue       interface{}
	AdjustmentRatio     float64
	Reason              string
	ExpectedImpact      string
	Risk                string
}

// TuningDecision 调优决策
type TuningDecision struct {
	Timestamp           time.Time
	Trigger             string
	Adjustments         []ParameterAdjustment
	Confidence          float64
	ExpectedImprovement float64
	Risk                float64
	AutoExecute         bool
}

// StrategyExecution 策略执行
type StrategyExecution struct {
	Timestamp           time.Time
	Decision            *TuningDecision
	Executed            bool
	Result              *ExecutionResult
}

// ExecutionResult 执行结果
type ExecutionResult struct {
	Success             bool
	Error               error
	PerformanceBefore   float64
	PerformanceAfter    float64
	Improvement         float64
	SideEffects         []string
}

// TuningRecord 调优记录
type TuningRecord struct {
	Timestamp           time.Time
	Type                string  // automatic, manual, rollback
	Parameters          map[string]interface{}
	Reason              string
	Result              *TuningResult
}

// TuningResult 调优结果
type TuningResult struct {
	Success             bool
	PerformanceGain     float64
	ResourceUsageChange float64
	StabilityImpact     float64
	UserExperienceChange float64
	Duration            time.Duration
}

// LearningDataset 学习数据集
type LearningDataset struct {
	Features            [][]float64
	Labels              []float64
	Weights             []float64
	
	// 数据统计
	FeatureStats        []FeatureStatistics
	LabelStats          LabelStatistics
	
	// 数据划分
	TrainingSplit       float64
	ValidationSplit     float64
	TestSplit           float64
	
	mutex               sync.RWMutex
}

// FeatureStatistics 特征统计
type FeatureStatistics struct {
	Name                string
	Mean                float64
	Std                 float64
	Min                 float64
	Max                 float64
	Correlation         float64
}

// LabelStatistics 标签统计
type LabelStatistics struct {
	Mean                float64
	Std                 float64
	Min                 float64
	Max                 float64
	Distribution        map[string]int
}

// TuningEngineStats 调优引擎统计
type TuningEngineStats struct {
	// 基础统计
	TotalAdjustments    int64
	SuccessfulAdjustments int64
	FailedAdjustments   int64
	RollbackCount       int64
	
	// 性能统计
	AverageImprovement  float64
	MaxImprovement      float64
	TotalImprovement    float64
	
	// 学习统计
	ModelAccuracy       float64
	PredictionAccuracy  float64
	LearningIterations  int64
	
	// 时间统计
	AverageDecisionTime time.Duration
	TotalTuningTime     time.Duration
	LastTuningTime      time.Time
	
	// 系统影响
	CPUOverhead         float64
	MemoryOverhead      int64
	
	mutex               sync.RWMutex
}

// NewAdaptiveTuningEngine 创建自适应调优引擎
func NewAdaptiveTuningEngine() *AdaptiveTuningEngine {
	config := &AdaptiveTuningConfig{
		TuningInterval:      5 * time.Minute,
		MetricInterval:      30 * time.Second,
		CPUUsageThreshold:   0.8,
		MemoryThreshold:     500 * 1024 * 1024, // 500MB
		LatencyThreshold:    10 * time.Second,
		LearningRate:        0.01,
		AdaptationSpeed:     0.1,
		StabilityFactor:     0.9,
		MinBatchInterval:    10 * time.Second,
		MaxBatchInterval:    10 * time.Minute,
		MinBatchSize:        10,
		MaxBatchSize:        1000,
		MaxAdjustmentRatio:  0.3, // 最大30%调整
		RollbackThreshold:   0.1, // 性能下降10%触发回滚
		PredictionWindow:    1 * time.Hour,
		ConfidenceThreshold: 0.7,
	}
	
	engine := &AdaptiveTuningEngine{
		config:           config,
		shutdownChan:     make(chan struct{}),
		maxHistorySize:   1000,
		tuningHistory:    make([]*TuningRecord, 0),
		stats: &TuningEngineStats{
			LastTuningTime: time.Now(),
		},
	}
	
	// 初始化组件
	engine.metricCollector = NewMetricCollector()
	engine.performanceAnalyzer = NewPerformanceAnalyzer()
	engine.predictiveModel = NewPredictiveModel()
	engine.tuningStrategy = NewTuningStrategy()
	engine.learningData = NewLearningDataset()
	
	return engine
}

// Start 启动自适应调优引擎
func (a *AdaptiveTuningEngine) Start() error {
	if !atomic.CompareAndSwapInt32(&a.isRunning, 0, 1) {
		return fmt.Errorf("调优引擎已在运行中")
	}
	
	// 启动指标收集
	if err := a.metricCollector.Start(a.config.MetricInterval); err != nil {
		return fmt.Errorf("启动指标收集失败: %v", err)
	}
	
	// 启动主调优循环
	go a.tuningLoop()
	
	// 启动性能分析循环
	go a.analysisLoop()
	
	// 启动模型训练循环
	go a.learningLoop()
	
	fmt.Printf("🧠 [自适应调优引擎] 启动完成，调优间隔: %v\n", a.config.TuningInterval)
	return nil
}

// tuningLoop 调优循环
func (a *AdaptiveTuningEngine) tuningLoop() {
	ticker := time.NewTicker(a.config.TuningInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			a.performTuning()
			
		case <-a.shutdownChan:
			return
		}
	}
}

// analysisLoop 分析循环
func (a *AdaptiveTuningEngine) analysisLoop() {
	ticker := time.NewTicker(a.config.MetricInterval * 2) // 分析频率低于采集频率
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			a.performAnalysis()
			
		case <-a.shutdownChan:
			return
		}
	}
}

// learningLoop 学习循环
func (a *AdaptiveTuningEngine) learningLoop() {
	ticker := time.NewTicker(15 * time.Minute) // 每15分钟学习一次
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			a.performLearning()
			
		case <-a.shutdownChan:
			return
		}
	}
}

// performTuning 执行调优
func (a *AdaptiveTuningEngine) performTuning() {
	startTime := time.Now()
	
	// 获取当前指标
	currentMetrics := a.metricCollector.GetLatestMetrics()
	if currentMetrics == nil {
		return
	}
	
	// 分析性能状态
	performanceIssues := a.performanceAnalyzer.AnalyzeIssues(currentMetrics)
	
	// 生成调优决策
	decision := a.tuningStrategy.GenerateDecision(currentMetrics, performanceIssues)
	if decision == nil {
		return
	}
	
	// 验证决策合理性
	if !a.validateDecision(decision) {
		fmt.Printf("⚠️ [调优引擎] 决策验证失败，跳过执行\n")
		return
	}
	
	// 执行调优
	result := a.executeDecision(decision)
	
	// 记录调优历史
	record := &TuningRecord{
		Timestamp:  time.Now(),
		Type:       "automatic",
		Parameters: a.extractParameters(decision),
		Reason:     decision.Trigger,
		Result:     result,
	}
	
	a.addTuningRecord(record)
	
	// 更新统计
	a.updateTuningStats(result, time.Since(startTime))
	
	if result.Success {
		fmt.Printf("✅ [调优引擎] 自动调优完成，性能提升: %.2f%%\n", result.PerformanceGain*100)
	} else {
		fmt.Printf("❌ [调优引擎] 调优失败，考虑回滚\n")
		a.considerRollback(decision, result)
	}
}

// performAnalysis 执行性能分析
func (a *AdaptiveTuningEngine) performAnalysis() {
	// 趋势分析
	a.performanceAnalyzer.AnalyzeTrends(a.metricCollector.GetMetricsHistory(100))
	
	// 异常检测
	a.performanceAnalyzer.DetectAnomalies(a.metricCollector.GetLatestMetrics())
	
	// 相关性分析
	a.performanceAnalyzer.AnalyzeCorrelations(a.metricCollector.GetMetricsHistory(50))
}

// performLearning 执行机器学习
func (a *AdaptiveTuningEngine) performLearning() {
	// 收集训练数据
	a.collectTrainingData()
	
	// 训练预测模型
	if err := a.predictiveModel.Train(a.learningData); err != nil {
		fmt.Printf("⚠️ [调优引擎] 模型训练失败: %v\n", err)
		return
	}
	
	// 验证模型性能
	accuracy := a.predictiveModel.Validate()
	
	// 更新统计
	a.mutex.Lock()
	a.stats.ModelAccuracy = accuracy
	a.stats.LearningIterations++
	a.mutex.Unlock()
	
	fmt.Printf("🎓 [调优引擎] 模型训练完成，准确率: %.2f%%\n", accuracy*100)
}

// validateDecision 验证调优决策
func (a *AdaptiveTuningEngine) validateDecision(decision *TuningDecision) bool {
	// 检查置信度
	if decision.Confidence < a.config.ConfidenceThreshold {
		return false
	}
	
	// 检查风险级别
	if decision.Risk > 0.7 { // 风险过高
		return false
	}
	
	// 检查调整幅度
	for _, adj := range decision.Adjustments {
		if math.Abs(adj.AdjustmentRatio) > a.config.MaxAdjustmentRatio {
			return false
		}
	}
	
	return true
}

// executeDecision 执行调优决策
func (a *AdaptiveTuningEngine) executeDecision(decision *TuningDecision) *TuningResult {
	startTime := time.Now()
	
	// 获取执行前性能基线
	beforeMetrics := a.metricCollector.GetLatestMetrics()
	performanceBefore := a.calculateOverallPerformance(beforeMetrics)
	
	// 执行参数调整
	success := true
	
	for _, adjustment := range decision.Adjustments {
		if err := a.applyParameterAdjustment(adjustment); err != nil {
			success = false
			break
		}
	}
	
	if !success {
		return &TuningResult{
			Success:             false,
			PerformanceGain:     0,
			ResourceUsageChange: 0,
			StabilityImpact:     0,
			Duration:           time.Since(startTime),
		}
	}
	
	// 等待一段时间观察效果
	time.Sleep(30 * time.Second)
	
	// 获取执行后性能
	afterMetrics := a.metricCollector.GetLatestMetrics()
	performanceAfter := a.calculateOverallPerformance(afterMetrics)
	
	performanceGain := (performanceAfter - performanceBefore) / performanceBefore
	
	// 计算资源使用变化
	resourceBefore := float64(beforeMetrics.System.MemoryUsage + int64(beforeMetrics.System.CPUUsage*1000))
	resourceAfter := float64(afterMetrics.System.MemoryUsage + int64(afterMetrics.System.CPUUsage*1000))
	resourceChange := (resourceAfter - resourceBefore) / resourceBefore
	
	return &TuningResult{
		Success:             true,
		PerformanceGain:     performanceGain,
		ResourceUsageChange: resourceChange,
		StabilityImpact:     a.calculateStabilityImpact(beforeMetrics, afterMetrics),
		UserExperienceChange: performanceGain, // 简化假设
		Duration:           time.Since(startTime),
	}
}

// calculateOverallPerformance 计算整体性能分数
func (a *AdaptiveTuningEngine) calculateOverallPerformance(metrics *MetricSnapshot) float64 {
	if metrics == nil {
		return 0
	}
	
	// 性能分数计算（0-100分）
	cpuScore := math.Max(0, (1.0-metrics.System.CPUUsage)*40)  // CPU使用率越低越好，最高40分
	memoryScore := math.Max(0, (1.0-float64(metrics.System.MemoryUsage)/float64(metrics.System.MemoryTotal))*30) // 内存使用率越低越好，最高30分
	responseScore := math.Max(0, (1.0-math.Min(1.0, float64(metrics.Application.ResponseTime)/float64(time.Second)))*20) // 响应时间越短越好，最高20分
	cacheScore := metrics.Cache.HitRate * 10 // 缓存命中率越高越好，最高10分
	
	return cpuScore + memoryScore + responseScore + cacheScore
}

// calculateStabilityImpact 计算稳定性影响
func (a *AdaptiveTuningEngine) calculateStabilityImpact(before, after *MetricSnapshot) float64 {
	if before == nil || after == nil {
		return 0
	}
	
	// 简化的稳定性计算：比较关键指标的变化
	cpuVariation := math.Abs(after.System.CPUUsage - before.System.CPUUsage)
	memoryVariation := math.Abs(float64(after.System.MemoryUsage-before.System.MemoryUsage) / float64(before.System.MemoryUsage))
	
	// 变化越小，稳定性越好
	stabilityScore := 1.0 - (cpuVariation*0.5 + memoryVariation*0.5)
	return math.Max(0, stabilityScore)
}

// applyParameterAdjustment 应用参数调整
func (a *AdaptiveTuningEngine) applyParameterAdjustment(adjustment ParameterAdjustment) error {
	// 这里应该调用具体的参数设置函数
	// 暂时模拟实现
	fmt.Printf("🔧 [调优引擎] 调整参数 %s: %v -> %v (%.1f%%)\n", 
		adjustment.ParameterName, 
		adjustment.CurrentValue, 
		adjustment.ProposedValue,
		adjustment.AdjustmentRatio*100)
	
	return nil
}

// collectTrainingData 收集训练数据
func (a *AdaptiveTuningEngine) collectTrainingData() {
	history := a.metricCollector.GetMetricsHistory(200)
	_ = a.getTuningHistory(50) // 暂时不使用调优历史
	
	// 构建特征和标签
	for i, metrics := range history {
		if i < len(history)-1 {
			// 特征：当前指标
			features := []float64{
				metrics.System.CPUUsage,
				float64(metrics.System.MemoryUsage) / 1024 / 1024, // MB
				float64(metrics.Application.ResponseTime) / float64(time.Millisecond),
				metrics.Cache.HitRate,
				metrics.Cache.CompressionRatio,
			}
			
			// 标签：下一时刻的整体性能
			nextMetrics := history[i+1]
			label := a.calculateOverallPerformance(&nextMetrics)
			
			// 添加到学习数据集
			a.learningData.mutex.Lock()
			a.learningData.Features = append(a.learningData.Features, features)
			a.learningData.Labels = append(a.learningData.Labels, label)
			a.learningData.Weights = append(a.learningData.Weights, 1.0)
			a.learningData.mutex.Unlock()
		}
	}
	
	// 限制数据集大小
	a.learningData.mutex.Lock()
	maxSize := 1000
	if len(a.learningData.Features) > maxSize {
		excess := len(a.learningData.Features) - maxSize
		a.learningData.Features = a.learningData.Features[excess:]
		a.learningData.Labels = a.learningData.Labels[excess:]
		a.learningData.Weights = a.learningData.Weights[excess:]
	}
	a.learningData.mutex.Unlock()
}

// considerRollback 考虑回滚
func (a *AdaptiveTuningEngine) considerRollback(decision *TuningDecision, result *TuningResult) {
	if result.PerformanceGain < -a.config.RollbackThreshold {
		fmt.Printf("🔄 [调优引擎] 触发自动回滚，性能下降: %.2f%%\n", result.PerformanceGain*100)
		a.performRollback(decision)
	}
}

// performRollback 执行回滚
func (a *AdaptiveTuningEngine) performRollback(originalDecision *TuningDecision) {
	// 创建回滚决策
	rollbackDecision := &TuningDecision{
		Timestamp:   time.Now(),
		Trigger:     "automatic_rollback",
		Adjustments: make([]ParameterAdjustment, 0),
		Confidence:  1.0,
		AutoExecute: true,
	}
	
	// 反向调整所有参数
	for _, adjustment := range originalDecision.Adjustments {
		rollbackAdjustment := ParameterAdjustment{
			ParameterName:     adjustment.ParameterName,
			CurrentValue:      adjustment.ProposedValue,
			ProposedValue:     adjustment.CurrentValue,
			AdjustmentRatio:   -adjustment.AdjustmentRatio,
			Reason:            "rollback",
			ExpectedImpact:    "restore_stability",
			Risk:              "low",
		}
		rollbackDecision.Adjustments = append(rollbackDecision.Adjustments, rollbackAdjustment)
	}
	
	// 执行回滚
	result := a.executeDecision(rollbackDecision)
	
	// 记录回滚
	record := &TuningRecord{
		Timestamp:  time.Now(),
		Type:       "rollback",
		Parameters: a.extractParameters(rollbackDecision),
		Reason:     "performance_degradation",
		Result:     result,
	}
	
	a.addTuningRecord(record)
	
	// 更新统计
	atomic.AddInt64(&a.stats.RollbackCount, 1)
}

// addTuningRecord 添加调优记录
func (a *AdaptiveTuningEngine) addTuningRecord(record *TuningRecord) {
	a.historyMutex.Lock()
	defer a.historyMutex.Unlock()
	
	a.tuningHistory = append(a.tuningHistory, record)
	
	// 限制历史记录大小
	if len(a.tuningHistory) > a.maxHistorySize {
		a.tuningHistory = a.tuningHistory[1:]
	}
}

// updateTuningStats 更新调优统计
func (a *AdaptiveTuningEngine) updateTuningStats(result *TuningResult, decisionTime time.Duration) {
	a.stats.mutex.Lock()
	defer a.stats.mutex.Unlock()
	
	a.stats.TotalAdjustments++
	if result.Success {
		a.stats.SuccessfulAdjustments++
		a.stats.TotalImprovement += result.PerformanceGain
		a.stats.AverageImprovement = a.stats.TotalImprovement / float64(a.stats.SuccessfulAdjustments)
		
		if result.PerformanceGain > a.stats.MaxImprovement {
			a.stats.MaxImprovement = result.PerformanceGain
		}
	} else {
		a.stats.FailedAdjustments++
	}
	
	// 更新时间统计
	a.stats.TotalTuningTime += decisionTime
	a.stats.AverageDecisionTime = time.Duration(int64(a.stats.TotalTuningTime) / a.stats.TotalAdjustments)
	a.stats.LastTuningTime = time.Now()
}

// extractParameters 提取决策参数
func (a *AdaptiveTuningEngine) extractParameters(decision *TuningDecision) map[string]interface{} {
	params := make(map[string]interface{})
	for _, adj := range decision.Adjustments {
		params[adj.ParameterName] = adj.ProposedValue
	}
	return params
}

// getTuningHistory 获取调优历史
func (a *AdaptiveTuningEngine) getTuningHistory(limit int) []*TuningRecord {
	a.historyMutex.RLock()
	defer a.historyMutex.RUnlock()
	
	if limit <= 0 || limit > len(a.tuningHistory) {
		limit = len(a.tuningHistory)
	}
	
	history := make([]*TuningRecord, limit)
	startIndex := len(a.tuningHistory) - limit
	copy(history, a.tuningHistory[startIndex:])
	
	return history
}

// Stop 停止自适应调优引擎
func (a *AdaptiveTuningEngine) Stop() error {
	if !atomic.CompareAndSwapInt32(&a.isRunning, 1, 0) {
		return nil
	}
	
	// 停止指标收集
	a.metricCollector.Stop()
	
	// 停止所有循环
	close(a.shutdownChan)
	
	fmt.Printf("🧠 [自适应调优引擎] 已停止\n")
	return nil
}

// GetStats 获取调优引擎统计
func (a *AdaptiveTuningEngine) GetStats() *TuningEngineStats {
	a.stats.mutex.RLock()
	defer a.stats.mutex.RUnlock()
	
	statsCopy := *a.stats
	return &statsCopy
}

// GetTuningReport 获取调优报告
func (a *AdaptiveTuningEngine) GetTuningReport() map[string]interface{} {
	stats := a.GetStats()
	recentHistory := a.getTuningHistory(10)
	
	return map[string]interface{}{
		"engine_stats":    stats,
		"recent_history":  recentHistory,
		"current_trends":  a.performanceAnalyzer.GetCurrentTrends(),
		"anomalies":       a.performanceAnalyzer.GetDetectedAnomalies(),
		"predictions":     a.predictiveModel.GetPredictions(),
		"model_accuracy":  a.predictiveModel.GetAccuracy(),
	}
}