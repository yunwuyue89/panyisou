package cache

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// TrendAnalyzer 趋势分析器
type TrendAnalyzer struct {
	trends map[string]*TrendData
	mutex  sync.RWMutex
}

// TrendData 趋势数据
type TrendData struct {
	Values      []float64
	Timestamps  []time.Time
	Slope       float64
	RSquared    float64
	LastUpdate  time.Time
}

// AnomalyDetector 异常检测器
type AnomalyDetector struct {
	baselines map[string]*Baseline
	mutex     sync.RWMutex
}

// Baseline 基线数据
type Baseline struct {
	Mean       float64
	StdDev     float64
	Min        float64
	Max        float64
	SampleSize int
	LastUpdate time.Time
}

// CorrelationAnalyzer 相关性分析器
type CorrelationAnalyzer struct {
	correlationMatrix map[string]map[string]float64
	mutex             sync.RWMutex
}

// NewPerformanceAnalyzer 创建性能分析器
func NewPerformanceAnalyzer() *PerformanceAnalyzer {
	return &PerformanceAnalyzer{
		trendAnalyzer: &TrendAnalyzer{
			trends: make(map[string]*TrendData),
		},
		anomalyDetector: &AnomalyDetector{
			baselines: make(map[string]*Baseline),
		},
		correlationAnalyzer: &CorrelationAnalyzer{
			correlationMatrix: make(map[string]map[string]float64),
		},
		currentTrends:     make(map[string]Trend),
		detectedAnomalies: make([]Anomaly, 0),
		correlations:      make(map[string]float64),
	}
}

// AnalyzeTrends 分析趋势
func (p *PerformanceAnalyzer) AnalyzeTrends(history []MetricSnapshot) {
	if len(history) < 3 {
		return // 数据点太少，无法分析趋势
	}
	
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	// 分析各个指标的趋势
	metrics := map[string][]float64{
		"cpu_usage":        make([]float64, len(history)),
		"memory_usage":     make([]float64, len(history)),
		"response_time":    make([]float64, len(history)),
		"cache_hit_rate":   make([]float64, len(history)),
		"overall_performance": make([]float64, len(history)),
	}
	
	timestamps := make([]time.Time, len(history))
	
	// 提取时间序列数据
	for i, snapshot := range history {
		metrics["cpu_usage"][i] = snapshot.System.CPUUsage
		metrics["memory_usage"][i] = float64(snapshot.System.MemoryUsage) / 1024 / 1024 // MB
		metrics["response_time"][i] = float64(snapshot.Application.ResponseTime) / float64(time.Millisecond)
		metrics["cache_hit_rate"][i] = snapshot.Cache.HitRate
		metrics["overall_performance"][i] = snapshot.OverallPerformance
		timestamps[i] = snapshot.Timestamp
	}
	
	// 分析每个指标的趋势
	for metricName, values := range metrics {
		trend := p.calculateTrend(metricName, values, timestamps)
		p.currentTrends[metricName] = trend
	}
}

// calculateTrend 计算趋势
func (p *PerformanceAnalyzer) calculateTrend(metricName string, values []float64, timestamps []time.Time) Trend {
	if len(values) < 2 {
		return Trend{
			Metric:     metricName,
			Direction:  "stable",
			Slope:      0,
			Confidence: 0,
			Duration:   0,
			Prediction: values[len(values)-1],
		}
	}
	
	// 线性回归计算趋势
	slope, intercept, rSquared := p.linearRegression(values, timestamps)
	
	// 确定趋势方向
	direction := "stable"
	if math.Abs(slope) > 0.01 { // 阈值
		if slope > 0 {
			direction = "increasing"
		} else {
			direction = "decreasing"
		}
	}
	
	// 计算置信度
	confidence := math.Min(rSquared, 1.0)
	
	// 预测未来值
	futureTime := timestamps[len(timestamps)-1].Add(5 * time.Minute)
	prediction := intercept + slope*float64(futureTime.Unix())
	
	return Trend{
		Metric:     metricName,
		Direction:  direction,
		Slope:      slope,
		Confidence: confidence,
		Duration:   timestamps[len(timestamps)-1].Sub(timestamps[0]),
		Prediction: prediction,
	}
}

// linearRegression 线性回归
func (p *PerformanceAnalyzer) linearRegression(y []float64, timestamps []time.Time) (slope, intercept, rSquared float64) {
	n := float64(len(y))
	if n < 2 {
		return 0, y[0], 0
	}
	
	// 转换时间戳为数值
	x := make([]float64, len(timestamps))
	for i, t := range timestamps {
		x[i] = float64(t.Unix())
	}
	
	// 计算均值
	var sumX, sumY float64
	for i := 0; i < len(x); i++ {
		sumX += x[i]
		sumY += y[i]
	}
	meanX := sumX / n
	meanY := sumY / n
	
	// 计算斜率和截距
	var numerator, denominator float64
	for i := 0; i < len(x); i++ {
		numerator += (x[i] - meanX) * (y[i] - meanY)
		denominator += (x[i] - meanX) * (x[i] - meanX)
	}
	
	if denominator == 0 {
		return 0, meanY, 0
	}
	
	slope = numerator / denominator
	intercept = meanY - slope*meanX
	
	// 计算R²
	var ssRes, ssTot float64
	for i := 0; i < len(y); i++ {
		predicted := intercept + slope*x[i]
		ssRes += (y[i] - predicted) * (y[i] - predicted)
		ssTot += (y[i] - meanY) * (y[i] - meanY)
	}
	
	if ssTot == 0 {
		rSquared = 1.0
	} else {
		rSquared = 1.0 - ssRes/ssTot
	}
	
	return slope, intercept, math.Max(0, rSquared)
}

// DetectAnomalies 检测异常
func (p *PerformanceAnalyzer) DetectAnomalies(currentMetrics *MetricSnapshot) {
	if currentMetrics == nil {
		return
	}
	
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	// 清空之前的异常
	p.detectedAnomalies = make([]Anomaly, 0)
	
	// 检测各个指标的异常
	metrics := map[string]float64{
		"cpu_usage":         currentMetrics.System.CPUUsage,
		"memory_usage":      float64(currentMetrics.System.MemoryUsage) / 1024 / 1024,
		"response_time":     float64(currentMetrics.Application.ResponseTime) / float64(time.Millisecond),
		"error_rate":        currentMetrics.Application.ErrorRate,
		"cache_hit_rate":    currentMetrics.Cache.HitRate,
		"overall_performance": currentMetrics.OverallPerformance,
	}
	
	for metricName, value := range metrics {
		anomaly := p.detectMetricAnomaly(metricName, value, currentMetrics.Timestamp)
		if anomaly != nil {
			p.detectedAnomalies = append(p.detectedAnomalies, *anomaly)
		}
	}
}

// detectMetricAnomaly 检测单个指标异常
func (p *PerformanceAnalyzer) detectMetricAnomaly(metricName string, value float64, timestamp time.Time) *Anomaly {
	baseline := p.anomalyDetector.baselines[metricName]
	if baseline == nil {
		// 创建新基线
		p.anomalyDetector.baselines[metricName] = &Baseline{
			Mean:       value,
			StdDev:     0,
			Min:        value,
			Max:        value,
			SampleSize: 1,
			LastUpdate: timestamp,
		}
		return nil
	}
	
	// 更新基线
	p.updateBaseline(baseline, value, timestamp)
	
	// 使用3-sigma规则检测异常
	if baseline.StdDev > 0 {
		zScore := math.Abs(value-baseline.Mean) / baseline.StdDev
		
		var severity string
		var impact float64
		
		if zScore > 3.0 {
			severity = "high"
			impact = 0.8
		} else if zScore > 2.0 {
			severity = "medium"
			impact = 0.5
		} else if zScore > 1.5 {
			severity = "low"
			impact = 0.2
		} else {
			return nil // 无异常
		}
		
		// 确定异常描述
		description := fmt.Sprintf("%s异常: 当前值%.2f, 期望范围[%.2f, %.2f]", 
			metricName, value, 
			baseline.Mean-2*baseline.StdDev, 
			baseline.Mean+2*baseline.StdDev)
		
		return &Anomaly{
			Metric:        metricName,
			Timestamp:     timestamp,
			Severity:      severity,
			Value:         value,
			ExpectedRange: [2]float64{baseline.Mean - 2*baseline.StdDev, baseline.Mean + 2*baseline.StdDev},
			Description:   description,
			Impact:        impact,
		}
	}
	
	return nil
}

// updateBaseline 更新基线
func (p *PerformanceAnalyzer) updateBaseline(baseline *Baseline, newValue float64, timestamp time.Time) {
	// 增量更新均值和标准差
	oldMean := baseline.Mean
	baseline.SampleSize++
	baseline.Mean += (newValue - baseline.Mean) / float64(baseline.SampleSize)
	
	// 更新方差（Welford算法）
	if baseline.SampleSize > 1 {
		variance := (float64(baseline.SampleSize-2)*baseline.StdDev*baseline.StdDev + 
			(newValue-oldMean)*(newValue-baseline.Mean)) / float64(baseline.SampleSize-1)
		baseline.StdDev = math.Sqrt(math.Max(0, variance))
	}
	
	// 更新最值
	if newValue < baseline.Min {
		baseline.Min = newValue
	}
	if newValue > baseline.Max {
		baseline.Max = newValue
	}
	
	baseline.LastUpdate = timestamp
}

// AnalyzeCorrelations 分析相关性
func (p *PerformanceAnalyzer) AnalyzeCorrelations(history []MetricSnapshot) {
	if len(history) < 3 {
		return
	}
	
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	// 提取指标数据
	metrics := map[string][]float64{
		"cpu_usage":         make([]float64, len(history)),
		"memory_usage":      make([]float64, len(history)),
		"response_time":     make([]float64, len(history)),
		"cache_hit_rate":    make([]float64, len(history)),
		"overall_performance": make([]float64, len(history)),
	}
	
	for i, snapshot := range history {
		metrics["cpu_usage"][i] = snapshot.System.CPUUsage
		metrics["memory_usage"][i] = float64(snapshot.System.MemoryUsage) / 1024 / 1024
		metrics["response_time"][i] = float64(snapshot.Application.ResponseTime) / float64(time.Millisecond)
		metrics["cache_hit_rate"][i] = snapshot.Cache.HitRate
		metrics["overall_performance"][i] = snapshot.OverallPerformance
	}
	
	// 计算相关性矩阵
	metricNames := make([]string, 0, len(metrics))
	for name := range metrics {
		metricNames = append(metricNames, name)
	}
	sort.Strings(metricNames)
	
	for i, metric1 := range metricNames {
		if p.correlationAnalyzer.correlationMatrix[metric1] == nil {
			p.correlationAnalyzer.correlationMatrix[metric1] = make(map[string]float64)
		}
		
		for j, metric2 := range metricNames {
			if i <= j {
				correlation := p.calculateCorrelation(metrics[metric1], metrics[metric2])
				p.correlationAnalyzer.correlationMatrix[metric1][metric2] = correlation
				p.correlationAnalyzer.correlationMatrix[metric2][metric1] = correlation
				
				// 保存重要相关性
				if math.Abs(correlation) > 0.5 && metric1 != metric2 {
					p.correlations[fmt.Sprintf("%s_%s", metric1, metric2)] = correlation
				}
			}
		}
	}
}

// calculateCorrelation 计算皮尔逊相关系数
func (p *PerformanceAnalyzer) calculateCorrelation(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return 0
	}
	
	n := float64(len(x))
	
	// 计算均值
	var sumX, sumY float64
	for i := 0; i < len(x); i++ {
		sumX += x[i]
		sumY += y[i]
	}
	meanX := sumX / n
	meanY := sumY / n
	
	// 计算协方差和方差
	var covariance, varianceX, varianceY float64
	for i := 0; i < len(x); i++ {
		dx := x[i] - meanX
		dy := y[i] - meanY
		covariance += dx * dy
		varianceX += dx * dx
		varianceY += dy * dy
	}
	
	// 计算相关系数
	if varianceX == 0 || varianceY == 0 {
		return 0
	}
	
	correlation := covariance / math.Sqrt(varianceX*varianceY)
	return correlation
}

// AnalyzeIssues 分析性能问题
func (p *PerformanceAnalyzer) AnalyzeIssues(currentMetrics *MetricSnapshot) []string {
	if currentMetrics == nil {
		return nil
	}
	
	issues := make([]string, 0)
	
	// CPU使用率过高
	if currentMetrics.System.CPUUsage > 0.8 {
		issues = append(issues, "high_cpu_usage")
	}
	
	// 内存使用率过高
	memoryUsageRatio := float64(currentMetrics.System.MemoryUsage) / float64(currentMetrics.System.MemoryTotal)
	if memoryUsageRatio > 0.85 {
		issues = append(issues, "high_memory_usage")
	}
	
	// 响应时间过长
	if currentMetrics.Application.ResponseTime > 1*time.Second {
		issues = append(issues, "high_response_time")
	}
	
	// 错误率过高
	if currentMetrics.Application.ErrorRate > 0.05 {
		issues = append(issues, "high_error_rate")
	}
	
	// 缓存命中率过低
	if currentMetrics.Cache.HitRate < 0.7 {
		issues = append(issues, "low_cache_hit_rate")
	}
	
	// 整体性能过低
	if currentMetrics.OverallPerformance < 60 {
		issues = append(issues, "low_overall_performance")
	}
	
	return issues
}

// GetCurrentTrends 获取当前趋势
func (p *PerformanceAnalyzer) GetCurrentTrends() map[string]Trend {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	trends := make(map[string]Trend)
	for k, v := range p.currentTrends {
		trends[k] = v
	}
	
	return trends
}

// GetDetectedAnomalies 获取检测到的异常
func (p *PerformanceAnalyzer) GetDetectedAnomalies() []Anomaly {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	anomalies := make([]Anomaly, len(p.detectedAnomalies))
	copy(anomalies, p.detectedAnomalies)
	
	return anomalies
}

// GetCorrelations 获取相关性
func (p *PerformanceAnalyzer) GetCorrelations() map[string]float64 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	correlations := make(map[string]float64)
	for k, v := range p.correlations {
		correlations[k] = v
	}
	
	return correlations
}