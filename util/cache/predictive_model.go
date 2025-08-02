package cache

import (
	"fmt"
	"math"
	"time"
)

// NewPredictiveModel 创建预测模型
func NewPredictiveModel() *PredictiveModel {
	return &PredictiveModel{
		modelType:       "linear_regression",
		coefficients:    make([]float64, 0),
		seasonalFactors: make([]float64, 0),
		trainingData:    make([]DataPoint, 0),
		testData:        make([]DataPoint, 0),
		predictions:     make(map[string]Prediction),
	}
}

// Train 训练模型
func (p *PredictiveModel) Train(dataset *LearningDataset) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	if dataset == nil || len(dataset.Features) == 0 {
		return fmt.Errorf("训练数据集为空")
	}
	
	// 准备训练数据
	p.prepareTrainingData(dataset)
	
	// 根据模型类型进行训练
	switch p.modelType {
	case "linear_regression":
		return p.trainLinearRegression(dataset)
	case "exponential_smoothing":
		return p.trainExponentialSmoothing(dataset)
	default:
		return fmt.Errorf("不支持的模型类型: %s", p.modelType)
	}
}

// prepareTrainingData 准备训练数据
func (p *PredictiveModel) prepareTrainingData(dataset *LearningDataset) {
	dataset.mutex.RLock()
	defer dataset.mutex.RUnlock()
	
	totalSamples := len(dataset.Features)
	if totalSamples == 0 {
		return
	}
	
	// 数据分割
	trainSize := int(float64(totalSamples) * 0.8) // 80%用于训练
	
	p.trainingData = make([]DataPoint, trainSize)
	p.testData = make([]DataPoint, totalSamples-trainSize)
	
	// 填充训练数据
	for i := 0; i < trainSize; i++ {
		p.trainingData[i] = DataPoint{
			Timestamp: time.Now().Add(-time.Duration(totalSamples-i) * time.Minute),
			Values:    make(map[string]float64),
		}
		
		// 转换特征为命名值
		if len(dataset.Features[i]) >= 5 {
			p.trainingData[i].Values["cpu_usage"] = dataset.Features[i][0]
			p.trainingData[i].Values["memory_usage"] = dataset.Features[i][1]
			p.trainingData[i].Values["response_time"] = dataset.Features[i][2]
			p.trainingData[i].Values["cache_hit_rate"] = dataset.Features[i][3]
			p.trainingData[i].Values["compression_ratio"] = dataset.Features[i][4]
		}
	}
	
	// 填充测试数据
	for i := 0; i < len(p.testData); i++ {
		testIndex := trainSize + i
		p.testData[i] = DataPoint{
			Timestamp: time.Now().Add(-time.Duration(totalSamples-testIndex) * time.Minute),
			Values:    make(map[string]float64),
		}
		
		if len(dataset.Features[testIndex]) >= 5 {
			p.testData[i].Values["cpu_usage"] = dataset.Features[testIndex][0]
			p.testData[i].Values["memory_usage"] = dataset.Features[testIndex][1]
			p.testData[i].Values["response_time"] = dataset.Features[testIndex][2]
			p.testData[i].Values["cache_hit_rate"] = dataset.Features[testIndex][3]
			p.testData[i].Values["compression_ratio"] = dataset.Features[testIndex][4]
		}
	}
}

// trainLinearRegression 训练线性回归模型
func (p *PredictiveModel) trainLinearRegression(dataset *LearningDataset) error {
	dataset.mutex.RLock()
	defer dataset.mutex.RUnlock()
	
	if len(dataset.Features) == 0 || len(dataset.Labels) != len(dataset.Features) {
		return fmt.Errorf("训练数据不匹配")
	}
	
	featuresCount := len(dataset.Features[0])
	samplesCount := len(dataset.Features)
	
	// 初始化系数（包括偏置项）
	p.coefficients = make([]float64, featuresCount+1)
	
	// 使用梯度下降训练
	learningRate := 0.01
	iterations := 1000
	
	for iter := 0; iter < iterations; iter++ {
		// 计算预测值和误差
		totalLoss := 0.0
		gradients := make([]float64, len(p.coefficients))
		
		for i := 0; i < samplesCount; i++ {
			// 计算预测值
			predicted := p.coefficients[0] // 偏置项
			for j := 0; j < featuresCount; j++ {
				predicted += p.coefficients[j+1] * dataset.Features[i][j]
			}
			
			// 计算误差
			error := predicted - dataset.Labels[i]
			totalLoss += error * error
			
			// 计算梯度
			gradients[0] += error // 偏置项梯度
			for j := 0; j < featuresCount; j++ {
				gradients[j+1] += error * dataset.Features[i][j]
			}
		}
		
		// 更新参数
		for j := 0; j < len(p.coefficients); j++ {
			p.coefficients[j] -= learningRate * gradients[j] / float64(samplesCount)
		}
		
		// 计算平均损失
		avgLoss := totalLoss / float64(samplesCount)
		
		// 早停条件
		if avgLoss < 0.001 {
			break
		}
	}
	
	return nil
}

// trainExponentialSmoothing 训练指数平滑模型
func (p *PredictiveModel) trainExponentialSmoothing(dataset *LearningDataset) error {
	dataset.mutex.RLock()
	defer dataset.mutex.RUnlock()
	
	if len(dataset.Labels) < 2 {
		return fmt.Errorf("指数平滑需要至少2个数据点")
	}
	
	// 简单指数平滑参数
	alpha := 0.3 // 平滑参数
	
	// 初始化
	p.coefficients = make([]float64, 2)
	p.coefficients[0] = dataset.Labels[0] // 初始水平
	p.coefficients[1] = alpha             // 平滑参数
	
	// 计算趋势组件
	if len(dataset.Labels) > 1 {
		p.trendComponent = dataset.Labels[1] - dataset.Labels[0]
	}
	
	return nil
}

// Predict 进行预测
func (p *PredictiveModel) Predict(features []float64, horizon time.Duration) (*Prediction, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	if len(p.coefficients) == 0 {
		return nil, fmt.Errorf("模型尚未训练")
	}
	
	var predictedValue float64
	var confidence float64
	
	switch p.modelType {
	case "linear_regression":
		if len(features) != len(p.coefficients)-1 {
			return nil, fmt.Errorf("特征维度不匹配")
		}
		
		// 线性回归预测
		predictedValue = p.coefficients[0] // 偏置项
		for i, feature := range features {
			predictedValue += p.coefficients[i+1] * feature
		}
		
		// 置信度基于训练数据的拟合程度
		confidence = math.Max(0.5, p.rmse) // 简化的置信度计算
		
	case "exponential_smoothing":
		// 指数平滑预测
		predictedValue = p.coefficients[0] + p.trendComponent*float64(horizon/time.Minute)
		confidence = 0.7 // 固定置信度
		
	default:
		return nil, fmt.Errorf("不支持的模型类型: %s", p.modelType)
	}
	
	return &Prediction{
		Metric:      "overall_performance",
		FutureValue: predictedValue,
		Confidence:  confidence,
		TimeHorizon: horizon,
		PredictedAt: time.Now(),
	}, nil
}

// Validate 验证模型性能
func (p *PredictiveModel) Validate() float64 {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	if len(p.testData) == 0 {
		return 0
	}
	
	correctPredictions := 0
	totalPredictions := len(p.testData)
	
	for _, testPoint := range p.testData {
		// 提取特征
		features := []float64{
			testPoint.Values["cpu_usage"],
			testPoint.Values["memory_usage"],
			testPoint.Values["response_time"],
			testPoint.Values["cache_hit_rate"],
			testPoint.Values["compression_ratio"],
		}
		
		// 进行预测
		prediction, err := p.Predict(features, 5*time.Minute)
		if err != nil {
			continue
		}
		
		// 计算实际性能分数（简化）
		actualPerformance := p.calculatePerformanceScore(testPoint.Values)
		
		// 判断预测是否准确（容忍误差10%）
		errorRate := math.Abs(prediction.FutureValue-actualPerformance) / actualPerformance
		if errorRate < 0.1 {
			correctPredictions++
		}
	}
	
	accuracy := float64(correctPredictions) / float64(totalPredictions)
	p.accuracy = accuracy
	
	// 计算RMSE
	p.calculateRMSE()
	
	return accuracy
}

// calculatePerformanceScore 计算性能分数
func (p *PredictiveModel) calculatePerformanceScore(values map[string]float64) float64 {
	// 简化的性能分数计算
	cpuScore := (1.0 - values["cpu_usage"]) * 30
	memoryScore := (1.0 - values["memory_usage"]/1000) * 25 // 假设内存以MB为单位
	responseScore := (1.0 - values["response_time"]/1000) * 25 // 假设响应时间以ms为单位
	cacheScore := values["cache_hit_rate"] * 20
	
	return math.Max(0, cpuScore+memoryScore+responseScore+cacheScore)
}

// calculateRMSE 计算均方根误差
func (p *PredictiveModel) calculateRMSE() {
	if len(p.testData) == 0 {
		return
	}
	
	var sumSquaredErrors float64
	validPredictions := 0
	
	for _, testPoint := range p.testData {
		features := []float64{
			testPoint.Values["cpu_usage"],
			testPoint.Values["memory_usage"],
			testPoint.Values["response_time"],
			testPoint.Values["cache_hit_rate"],
			testPoint.Values["compression_ratio"],
		}
		
		prediction, err := p.Predict(features, 5*time.Minute)
		if err != nil {
			continue
		}
		
		actualPerformance := p.calculatePerformanceScore(testPoint.Values)
		error := prediction.FutureValue - actualPerformance
		sumSquaredErrors += error * error
		validPredictions++
	}
	
	if validPredictions > 0 {
		p.rmse = math.Sqrt(sumSquaredErrors / float64(validPredictions))
	}
}

// PredictMultiple 预测多个指标
func (p *PredictiveModel) PredictMultiple(currentMetrics *MetricSnapshot, horizons []time.Duration) map[string]Prediction {
	if currentMetrics == nil {
		return nil
	}
	
	features := []float64{
		currentMetrics.System.CPUUsage,
		float64(currentMetrics.System.MemoryUsage) / 1024 / 1024,
		float64(currentMetrics.Application.ResponseTime) / float64(time.Millisecond),
		currentMetrics.Cache.HitRate,
		currentMetrics.Cache.CompressionRatio,
	}
	
	predictions := make(map[string]Prediction)
	
	for _, horizon := range horizons {
		predictionKey := fmt.Sprintf("performance_%s", horizon.String())
		
		prediction, err := p.Predict(features, horizon)
		if err == nil {
			predictions[predictionKey] = *prediction
		}
	}
	
	return predictions
}

// GetAccuracy 获取模型准确率
func (p *PredictiveModel) GetAccuracy() float64 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	return p.accuracy
}

// GetPredictions 获取所有预测结果
func (p *PredictiveModel) GetPredictions() map[string]Prediction {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	predictions := make(map[string]Prediction)
	for k, v := range p.predictions {
		predictions[k] = v
	}
	
	return predictions
}

// UpdatePredictions 更新预测结果
func (p *PredictiveModel) UpdatePredictions(currentMetrics *MetricSnapshot) {
	if currentMetrics == nil {
		return
	}
	
	// 预测未来1小时的性能
	horizons := []time.Duration{
		5 * time.Minute,
		15 * time.Minute,
		30 * time.Minute,
		1 * time.Hour,
	}
	
	newPredictions := p.PredictMultiple(currentMetrics, horizons)
	
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	// 更新预测结果
	for k, v := range newPredictions {
		p.predictions[k] = v
	}
	
	// 清理过期预测
	now := time.Now()
	for k, v := range p.predictions {
		if now.Sub(v.PredictedAt) > 2*time.Hour {
			delete(p.predictions, k)
		}
	}
}