package cache

import (
	"sort"
	"time"
)

// NewTuningStrategy 创建调优策略
func NewTuningStrategy() *TuningStrategy {
	strategy := &TuningStrategy{
		strategyType:         "balanced",
		rules:                make([]*TuningRule, 0),
		parameterAdjustments: make(map[string]ParameterAdjustment),
		executionHistory:     make([]*StrategyExecution, 0),
	}
	
	// 初始化调优规则
	strategy.initializeRules()
	
	return strategy
}

// NewLearningDataset 创建学习数据集
func NewLearningDataset() *LearningDataset {
	return &LearningDataset{
		Features:        make([][]float64, 0),
		Labels:          make([]float64, 0),
		Weights:         make([]float64, 0),
		FeatureStats:    make([]FeatureStatistics, 0),
		TrainingSplit:   0.8,
		ValidationSplit: 0.1,
		TestSplit:       0.1,
	}
}

// initializeRules 初始化调优规则
func (t *TuningStrategy) initializeRules() {
	t.rules = []*TuningRule{
		{
			Name:     "高CPU使用率调优",
			Priority: 1,
			Enabled:  true,
			Condition: func(metrics *MetricSnapshot) bool {
				return metrics.System.CPUUsage > 0.8
			},
			Action: func(engine *AdaptiveTuningEngine) (*TuningDecision, error) {
				return t.createCPUOptimizationDecision(engine)
			},
		},
		{
			Name:     "高内存使用调优",
			Priority: 2,
			Enabled:  true,
			Condition: func(metrics *MetricSnapshot) bool {
				memoryRatio := float64(metrics.System.MemoryUsage) / float64(metrics.System.MemoryTotal)
				return memoryRatio > 0.85
			},
			Action: func(engine *AdaptiveTuningEngine) (*TuningDecision, error) {
				return t.createMemoryOptimizationDecision(engine)
			},
		},
		{
			Name:     "响应时间过长调优",
			Priority: 3,
			Enabled:  true,
			Condition: func(metrics *MetricSnapshot) bool {
				return metrics.Application.ResponseTime > 500*time.Millisecond
			},
			Action: func(engine *AdaptiveTuningEngine) (*TuningDecision, error) {
				return t.createResponseTimeOptimizationDecision(engine)
			},
		},
		{
			Name:     "缓存命中率低调优",
			Priority: 4,
			Enabled:  true,
			Condition: func(metrics *MetricSnapshot) bool {
				return metrics.Cache.HitRate < 0.7
			},
			Action: func(engine *AdaptiveTuningEngine) (*TuningDecision, error) {
				return t.createCacheOptimizationDecision(engine)
			},
		},
		{
			Name:     "整体性能低调优",
			Priority: 5,
			Enabled:  true,
			Condition: func(metrics *MetricSnapshot) bool {
				return metrics.OverallPerformance < 60
			},
			Action: func(engine *AdaptiveTuningEngine) (*TuningDecision, error) {
				return t.createOverallPerformanceDecision(engine)
			},
		},
		{
			Name:     "预防性调优",
			Priority: 10,
			Enabled:  true,
			Condition: func(metrics *MetricSnapshot) bool {
				// 基于趋势的预防性调优
				return false // 暂时禁用，需要趋势数据
			},
			Action: func(engine *AdaptiveTuningEngine) (*TuningDecision, error) {
				return t.createPreventiveDecision(engine)
			},
		},
	}
}

// GenerateDecision 生成调优决策
func (t *TuningStrategy) GenerateDecision(metrics *MetricSnapshot, issues []string) *TuningDecision {
	if metrics == nil {
		return nil
	}
	
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	// 按优先级排序规则
	sort.Slice(t.rules, func(i, j int) bool {
		return t.rules[i].Priority < t.rules[j].Priority
	})
	
	// 检查规则并生成决策
	for _, rule := range t.rules {
		if !rule.Enabled {
			continue
		}
		
		// 检查冷却时间（防止频繁调优）
		if time.Since(rule.LastTriggered) < 5*time.Minute {
			continue
		}
		
		// 检查条件
		if rule.Condition(metrics) {
			decision, err := rule.Action(nil) // 简化实现，不传递engine
			if err != nil {
				continue
			}
			
			// 更新规则状态
			rule.LastTriggered = time.Now()
			rule.TriggerCount++
			
			// 设置决策基本信息
			decision.Timestamp = time.Now()
			decision.Trigger = rule.Name
			
			return decision
		}
	}
	
	return nil
}

// createCPUOptimizationDecision 创建CPU优化决策
func (t *TuningStrategy) createCPUOptimizationDecision(engine *AdaptiveTuningEngine) (*TuningDecision, error) {
	adjustments := []ParameterAdjustment{
		{
			ParameterName:   "batch_interval",
			CurrentValue:    "60s",
			ProposedValue:   "90s",
			AdjustmentRatio: 0.5, // 增加50%
			Reason:          "减少CPU负载",
			ExpectedImpact:  "降低CPU使用率",
			Risk:            "medium",
		},
		{
			ParameterName:   "batch_size",
			CurrentValue:    100,
			ProposedValue:   150,
			AdjustmentRatio: 0.5,
			Reason:          "减少处理频率",
			ExpectedImpact:  "降低CPU负载",
			Risk:            "low",
		},
	}
	
	return &TuningDecision{
		Adjustments:         adjustments,
		Confidence:          0.8,
		ExpectedImprovement: 0.15,
		Risk:                0.3,
		AutoExecute:         true,
	}, nil
}

// createMemoryOptimizationDecision 创建内存优化决策
func (t *TuningStrategy) createMemoryOptimizationDecision(engine *AdaptiveTuningEngine) (*TuningDecision, error) {
	adjustments := []ParameterAdjustment{
		{
			ParameterName:   "max_buffer_size",
			CurrentValue:    1000,
			ProposedValue:   700,
			AdjustmentRatio: -0.3, // 减少30%
			Reason:          "减少内存占用",
			ExpectedImpact:  "降低内存使用率",
			Risk:            "medium",
		},
		{
			ParameterName:   "cache_cleanup_frequency",
			CurrentValue:    "5m",
			ProposedValue:   "3m",
			AdjustmentRatio: -0.4,
			Reason:          "更频繁清理缓存",
			ExpectedImpact:  "释放内存空间",
			Risk:            "low",
		},
	}
	
	return &TuningDecision{
		Adjustments:         adjustments,
		Confidence:          0.85,
		ExpectedImprovement: 0.2,
		Risk:                0.25,
		AutoExecute:         true,
	}, nil
}

// createResponseTimeOptimizationDecision 创建响应时间优化决策
func (t *TuningStrategy) createResponseTimeOptimizationDecision(engine *AdaptiveTuningEngine) (*TuningDecision, error) {
	adjustments := []ParameterAdjustment{
		{
			ParameterName:   "batch_interval",
			CurrentValue:    "60s",
			ProposedValue:   "30s",
			AdjustmentRatio: -0.5, // 减少50%
			Reason:          "更快的数据写入",
			ExpectedImpact:  "降低响应时间",
			Risk:            "medium",
		},
		{
			ParameterName:   "concurrent_workers",
			CurrentValue:    4,
			ProposedValue:   6,
			AdjustmentRatio: 0.5,
			Reason:          "增加并发处理",
			ExpectedImpact:  "提高处理速度",
			Risk:            "high",
		},
	}
	
	return &TuningDecision{
		Adjustments:         adjustments,
		Confidence:          0.75,
		ExpectedImprovement: 0.25,
		Risk:                0.4,
		AutoExecute:         false, // 高风险，需要手动确认
	}, nil
}

// createCacheOptimizationDecision 创建缓存优化决策
func (t *TuningStrategy) createCacheOptimizationDecision(engine *AdaptiveTuningEngine) (*TuningDecision, error) {
	adjustments := []ParameterAdjustment{
		{
			ParameterName:   "cache_ttl",
			CurrentValue:    "1h",
			ProposedValue:   "2h",
			AdjustmentRatio: 1.0, // 增加100%
			Reason:          "延长缓存生存时间",
			ExpectedImpact:  "提高缓存命中率",
			Risk:            "low",
		},
		{
			ParameterName:   "cache_size_limit",
			CurrentValue:    1000,
			ProposedValue:   1500,
			AdjustmentRatio: 0.5,
			Reason:          "增加缓存容量",
			ExpectedImpact:  "减少缓存驱逐",
			Risk:            "medium",
		},
	}
	
	return &TuningDecision{
		Adjustments:         adjustments,
		Confidence:          0.9,
		ExpectedImprovement: 0.3,
		Risk:                0.2,
		AutoExecute:         true,
	}, nil
}

// createOverallPerformanceDecision 创建整体性能优化决策
func (t *TuningStrategy) createOverallPerformanceDecision(engine *AdaptiveTuningEngine) (*TuningDecision, error) {
	adjustments := []ParameterAdjustment{
		{
			ParameterName:   "global_optimization",
			CurrentValue:    false,
			ProposedValue:   true,
			AdjustmentRatio: 1.0,
			Reason:          "启用全局优化",
			ExpectedImpact:  "整体性能提升",
			Risk:            "medium",
		},
		{
			ParameterName:   "compression_level",
			CurrentValue:    "standard",
			ProposedValue:   "high",
			AdjustmentRatio: 0.3,
			Reason:          "提高压缩效率",
			ExpectedImpact:  "减少存储开销",
			Risk:            "low",
		},
	}
	
	return &TuningDecision{
		Adjustments:         adjustments,
		Confidence:          0.7,
		ExpectedImprovement: 0.2,
		Risk:                0.35,
		AutoExecute:         true,
	}, nil
}

// createPreventiveDecision 创建预防性调优决策
func (t *TuningStrategy) createPreventiveDecision(engine *AdaptiveTuningEngine) (*TuningDecision, error) {
	// 基于趋势预测的预防性调优
	adjustments := []ParameterAdjustment{
		{
			ParameterName:   "preventive_scaling",
			CurrentValue:    1.0,
			ProposedValue:   1.1,
			AdjustmentRatio: 0.1,
			Reason:          "预防性资源扩展",
			ExpectedImpact:  "避免性能下降",
			Risk:            "low",
		},
	}
	
	return &TuningDecision{
		Adjustments:         adjustments,
		Confidence:          0.6,
		ExpectedImprovement: 0.1,
		Risk:                0.15,
		AutoExecute:         true,
	}, nil
}

// ExecuteDecision 执行决策
func (t *TuningStrategy) ExecuteDecision(decision *TuningDecision) *StrategyExecution {
	execution := &StrategyExecution{
		Timestamp: time.Now(),
		Decision:  decision,
		Executed:  false,
	}
	
	// 简化的执行逻辑
	execution.Executed = true
	execution.Result = &ExecutionResult{
		Success:           true,
		PerformanceBefore: 70.0, // 模拟值
		PerformanceAfter:  85.0, // 模拟值
		Improvement:       0.15,
		SideEffects:       []string{},
	}
	
	// 记录执行历史
	t.mutex.Lock()
	t.executionHistory = append(t.executionHistory, execution)
	
	// 限制历史记录大小
	if len(t.executionHistory) > 100 {
		t.executionHistory = t.executionHistory[1:]
	}
	t.mutex.Unlock()
	
	return execution
}

// GetExecutionHistory 获取执行历史
func (t *TuningStrategy) GetExecutionHistory(limit int) []*StrategyExecution {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	
	if limit <= 0 || limit > len(t.executionHistory) {
		limit = len(t.executionHistory)
	}
	
	history := make([]*StrategyExecution, limit)
	startIndex := len(t.executionHistory) - limit
	copy(history, t.executionHistory[startIndex:])
	
	return history
}

// UpdateStrategy 更新策略
func (t *TuningStrategy) UpdateStrategy(strategyType string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	t.strategyType = strategyType
	
	// 根据策略类型调整规则优先级和启用状态
	switch strategyType {
	case "conservative":
		// 保守策略：只启用低风险规则
		for _, rule := range t.rules {
			rule.Enabled = rule.Priority <= 5
		}
		
	case "aggressive":
		// 激进策略：启用所有规则
		for _, rule := range t.rules {
			rule.Enabled = true
		}
		
	case "balanced":
		// 平衡策略：默认设置
		for _, rule := range t.rules {
			rule.Enabled = true
		}
	}
}

// GetStrategyStats 获取策略统计
func (t *TuningStrategy) GetStrategyStats() map[string]interface{} {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	
	stats := map[string]interface{}{
		"strategy_type":     t.strategyType,
		"total_executions":  len(t.executionHistory),
		"enabled_rules":     0,
		"rule_statistics":   make(map[string]interface{}),
	}
	
	enabledRules := 0
	ruleStats := make(map[string]interface{})
	
	for _, rule := range t.rules {
		if rule.Enabled {
			enabledRules++
		}
		
		ruleStats[rule.Name] = map[string]interface{}{
			"enabled":        rule.Enabled,
			"priority":       rule.Priority,
			"trigger_count":  rule.TriggerCount,
			"last_triggered": rule.LastTriggered,
		}
	}
	
	stats["enabled_rules"] = enabledRules
	stats["rule_statistics"] = ruleStats
	
	// 计算成功率
	successfulExecutions := 0
	for _, execution := range t.executionHistory {
		if execution.Result != nil && execution.Result.Success {
			successfulExecutions++
		}
	}
	
	if len(t.executionHistory) > 0 {
		stats["success_rate"] = float64(successfulExecutions) / float64(len(t.executionHistory))
	} else {
		stats["success_rate"] = 0.0
	}
	
	return stats
}