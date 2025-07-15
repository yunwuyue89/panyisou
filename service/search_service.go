package service

import (
	"context" // Added for context.WithTimeout
	"io/ioutil"
	"net/http" // Added for http.Client
	"sort"
	"strings"
	"time"

	"pansou/config"
	"pansou/model"
	"pansou/plugin"
	"pansou/util"
	"pansou/util/cache"
	"pansou/util/pool"
	"sync" // Added for sync.WaitGroup
)

// 优先关键词列表
var priorityKeywords = []string{"合集", "系列", "全", "完", "最新", "附"}

// 全局缓存实例和缓存是否初始化标志
var (
	twoLevelCache    *cache.TwoLevelCache
	enhancedTwoLevelCache *cache.EnhancedTwoLevelCache
	cacheInitialized bool
)

// 初始化缓存
func init() {
	if config.AppConfig != nil && config.AppConfig.CacheEnabled {
		var err error
		// 优先使用增强版缓存
		enhancedTwoLevelCache, err = cache.NewEnhancedTwoLevelCache()
		if err == nil {
			cacheInitialized = true
			return
		}
		
		// 如果增强版缓存初始化失败，回退到原始缓存
		twoLevelCache, err = cache.NewTwoLevelCache()
		if err == nil {
			cacheInitialized = true
		}
	}
}

// SearchService 搜索服务
type SearchService struct {
	pluginManager *plugin.PluginManager
}

// NewSearchService 创建搜索服务实例并确保缓存可用
func NewSearchService(pluginManager *plugin.PluginManager) *SearchService {
	// 检查缓存是否已初始化，如果未初始化则尝试重新初始化
	if !cacheInitialized && config.AppConfig != nil && config.AppConfig.CacheEnabled {
		var err error
		// 优先使用增强版缓存
		enhancedTwoLevelCache, err = cache.NewEnhancedTwoLevelCache()
		if err == nil {
			cacheInitialized = true
		} else {
			// 如果增强版缓存初始化失败，回退到原始缓存
			twoLevelCache, err = cache.NewTwoLevelCache()
			if err == nil {
				cacheInitialized = true
			}
		}
	}

	return &SearchService{
		pluginManager: pluginManager,
	}
}

// Search 执行搜索
func (s *SearchService) Search(keyword string, channels []string, concurrency int, forceRefresh bool, resultType string, sourceType string, plugins []string) (model.SearchResponse, error) {
	// 参数预处理
	// 源类型标准化
	if sourceType == "" {
		sourceType = "all"
	}

	// 插件参数规范化处理
	if sourceType == "tg" {
		// 对于只搜索Telegram的请求，忽略插件参数
		plugins = nil
	} else if sourceType == "all" || sourceType == "plugin" {
		// 检查是否为空列表或只包含空字符串
		if plugins == nil || len(plugins) == 0 {
			plugins = nil
		} else {
			// 检查是否有非空元素
			hasNonEmpty := false
			for _, p := range plugins {
				if p != "" {
					hasNonEmpty = true
					break
				}
			}

			// 如果全是空字符串，视为未指定
			if !hasNonEmpty {
				plugins = nil
			} else {
				// 检查是否包含所有插件
				allPlugins := s.pluginManager.GetPlugins()
				allPluginNames := make([]string, 0, len(allPlugins))
				for _, p := range allPlugins {
					allPluginNames = append(allPluginNames, strings.ToLower(p.Name()))
				}

				// 创建请求的插件名称集合（忽略空字符串）
				requestedPlugins := make([]string, 0, len(plugins))
				for _, p := range plugins {
					if p != "" {
						requestedPlugins = append(requestedPlugins, strings.ToLower(p))
					}
				}

				// 如果请求的插件数量与所有插件数量相同，检查是否包含所有插件
				if len(requestedPlugins) == len(allPluginNames) {
					// 创建映射以便快速查找
					pluginMap := make(map[string]bool)
					for _, p := range requestedPlugins {
						pluginMap[p] = true
					}

					// 检查是否包含所有插件
					allIncluded := true
					for _, name := range allPluginNames {
						if !pluginMap[name] {
							allIncluded = false
							break
						}
					}

					// 如果包含所有插件，统一设为nil
					if allIncluded {
						plugins = nil
					}
				}
			}
		}
	}

	// 并行获取TG搜索和插件搜索结果
	var tgResults []model.SearchResult
	var pluginResults []model.SearchResult
	
	var wg sync.WaitGroup
	var tgErr, pluginErr error
	
	// 如果需要搜索TG
	if sourceType == "all" || sourceType == "tg" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tgResults, tgErr = s.searchTG(keyword, channels, forceRefresh)
		}()
	}
	
	// 如果需要搜索插件
	if sourceType == "all" || sourceType == "plugin" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 对于插件搜索，我们总是希望获取最新的缓存数据
			// 因此，即使forceRefresh=false，我们也需要确保获取到最新的缓存
			pluginResults, pluginErr = s.searchPlugins(keyword, plugins, forceRefresh, concurrency)
		}()
	}
	
	// 等待所有搜索完成
	wg.Wait()
	
	// 检查错误
	if tgErr != nil {
		return model.SearchResponse{}, tgErr
	}
	if pluginErr != nil {
		return model.SearchResponse{}, pluginErr
	}
	
	// 合并结果
	allResults := mergeSearchResults(tgResults, pluginResults)

	// 过滤结果，确保标题包含搜索关键词
	filteredResults := filterResultsByKeyword(allResults, keyword)

	// 按照优化后的规则排序结果
	sortResultsByTimeAndKeywords(filteredResults)

	// 过滤结果，只保留有时间的结果或包含优先关键词的结果到Results中
	filteredForResults := make([]model.SearchResult, 0, len(filteredResults))
	for _, result := range filteredResults {
		// 有时间的结果或包含优先关键词的结果保留在Results中
		if !result.Datetime.IsZero() || getKeywordPriority(result.Title) > 0 {
			filteredForResults = append(filteredForResults, result)
		}
	}

	// 合并链接按网盘类型分组（使用所有过滤后的结果）
	mergedLinks := mergeResultsByType(filteredResults)

	// 构建响应
	var total int
	if resultType == "merged_by_type" {
		// 计算所有类型链接的总数
		total = 0
		for _, links := range mergedLinks {
			total += len(links)
		}
	} else {
		// 只计算filteredForResults的数量
		total = len(filteredForResults)
	}

	response := model.SearchResponse{
		Total:        total,
		Results:      filteredForResults, // 使用进一步过滤的结果
		MergedByType: mergedLinks,
	}

	// 根据resultType过滤返回结果
	return filterResponseByType(response, resultType), nil
}

// filterResponseByType 根据结果类型过滤响应
func filterResponseByType(response model.SearchResponse, resultType string) model.SearchResponse {
	switch resultType {
	case "merged_by_type":
		// 只返回MergedByType，Results设为nil，结合omitempty标签，JSON序列化时会忽略此字段
		return model.SearchResponse{
			Total:        response.Total,
			MergedByType: response.MergedByType,
			Results:      nil,
		}
	case "all":
		return response
	case "results":
		// 只返回Results
		return model.SearchResponse{
			Total:   response.Total,
			Results: response.Results,
		}
	default:
		// // 默认返回全部
		// return response
		return model.SearchResponse{
			Total:        response.Total,
			MergedByType: response.MergedByType,
			Results:      nil,
		}
	}
}

// 过滤结果，确保标题包含搜索关键词
func filterResultsByKeyword(results []model.SearchResult, keyword string) []model.SearchResult {
	// 预估过滤后会保留80%的结果
	filteredResults := make([]model.SearchResult, 0, len(results)*8/10)

	// 将关键词转为小写，用于不区分大小写的比较
	lowerKeyword := strings.ToLower(keyword)

	// 将关键词按空格分割，用于支持多关键词搜索
	keywords := strings.Fields(lowerKeyword)

	for _, result := range results {
		// 将标题和内容转为小写
		lowerTitle := strings.ToLower(result.Title)
		lowerContent := strings.ToLower(result.Content)

		// 检查每个关键词是否在标题或内容中
		matched := true
		for _, kw := range keywords {
			// 如果关键词是"pwd"，特殊处理，只要标题、内容或链接中包含即可
			if kw == "pwd" {
				// 检查标题、内容
				pwdInTitle := strings.Contains(lowerTitle, kw)
				pwdInContent := strings.Contains(lowerContent, kw)

				// 检查链接中是否包含pwd参数
				pwdInLinks := false
				for _, link := range result.Links {
					if strings.Contains(strings.ToLower(link.URL), "pwd=") {
						pwdInLinks = true
						break
					}
				}

				// 只要有一个包含pwd，就算匹配
				if pwdInTitle || pwdInContent || pwdInLinks {
					continue // 匹配成功，检查下一个关键词
				} else {
					matched = false
					break
				}
			} else {
				// 对于其他关键词，检查是否同时在标题和内容中
				if !strings.Contains(lowerTitle, kw) && !strings.Contains(lowerContent, kw) {
					matched = false
					break
				}
			}
		}

		if matched {
			filteredResults = append(filteredResults, result)
		}
	}

	return filteredResults
}

// 根据时间和关键词排序结果
func sortResultsByTimeAndKeywords(results []model.SearchResult) {
	sort.Slice(results, func(i, j int) bool {
		// 检查是否有零值时间
		iZeroTime := results[i].Datetime.IsZero()
		jZeroTime := results[j].Datetime.IsZero()

		// 如果两者都是零值时间，按关键词优先级排序
		if iZeroTime && jZeroTime {
			iPriority := getKeywordPriority(results[i].Title)
			jPriority := getKeywordPriority(results[j].Title)
			if iPriority != jPriority {
				return iPriority > jPriority
			}
			// 如果优先级也相同，按标题字母顺序排序
			return results[i].Title < results[j].Title
		}

		// 如果只有一个是零值时间，将其排在后面
		if iZeroTime {
			return false // i排在后面
		}
		if jZeroTime {
			return true // j排在后面，i排在前面
		}

		// 两者都有正常时间，使用原有逻辑
		// 计算两个结果的时间差（以天为单位）
		timeDiff := daysBetween(results[i].Datetime, results[j].Datetime)

		// 如果时间差超过30天，按时间排序（新的在前面）
		if abs(timeDiff) > 30 {
			return results[i].Datetime.After(results[j].Datetime)
		}

		// 如果时间差在30天内，先检查时间差是否超过1天
		if abs(timeDiff) > 1 {
			return results[i].Datetime.After(results[j].Datetime)
		}

		// 如果时间差在1天内，检查关键词优先级
		iPriority := getKeywordPriority(results[i].Title)
		jPriority := getKeywordPriority(results[j].Title)

		// 如果优先级不同，优先级高的排在前面
		if iPriority != jPriority {
			return iPriority > jPriority
		}

		// 如果优先级相同且时间差在1天内，仍然按时间排序（新的在前面）
		return results[i].Datetime.After(results[j].Datetime)
	})
}

// 计算两个时间之间的天数差
func daysBetween(t1, t2 time.Time) float64 {
	duration := t1.Sub(t2)
	return duration.Hours() / 24
}

// 绝对值
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// 获取标题中包含优先关键词的优先级
func getKeywordPriority(title string) int {
	title = strings.ToLower(title)
	for i, keyword := range priorityKeywords {
		if strings.Contains(title, keyword) {
			// 返回优先级（数组索引越小，优先级越高）
			return len(priorityKeywords) - i
		}
	}
	return 0
}

// 搜索单个频道
func (s *SearchService) searchChannel(keyword string, channel string) ([]model.SearchResult, error) {
	// 构建搜索URL
	url := util.BuildSearchURL(channel, keyword, "")

	// 使用全局HTTP客户端（已配置代理）
	client := util.GetHTTPClient()

	// 创建一个带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 解析响应
	results, _, err := util.ParseSearchResults(string(body), channel)
	if err != nil {
		return nil, err
	}

	return results, nil
}

// 将搜索结果按网盘类型分组
func mergeResultsByType(results []model.SearchResult) model.MergedLinks {
	// 创建合并结果的映射
	mergedLinks := make(model.MergedLinks, 10) // 预分配容量，假设有10种不同的网盘类型

	// 用于去重的映射，键为URL
	uniqueLinks := make(map[string]model.MergedLink)

	// 遍历所有搜索结果
	for _, result := range results {
		for _, link := range result.Links {
			// 创建合并后的链接
			mergedLink := model.MergedLink{
				URL:      link.URL,
				Password: link.Password,
				Note:     result.Title,
				Datetime: result.Datetime,
			}

			// 检查是否已存在相同URL的链接
			if existingLink, exists := uniqueLinks[link.URL]; exists {
				// 如果已存在，只有当当前链接的时间更新时才替换
				if mergedLink.Datetime.After(existingLink.Datetime) {
					uniqueLinks[link.URL] = mergedLink
				}
			} else {
				// 如果不存在，直接添加
				uniqueLinks[link.URL] = mergedLink
			}
		}
	}

	// 将去重后的链接按类型分组
	for url, mergedLink := range uniqueLinks {
		// 获取链接类型
		linkType := ""
		for _, result := range results {
			for _, link := range result.Links {
				if link.URL == url {
					linkType = link.Type
					break
				}
			}
			if linkType != "" {
				break
			}
		}

		// 如果没有找到类型，使用"unknown"
		if linkType == "" {
			linkType = "unknown"
		}

		// 添加到对应类型的列表中
		mergedLinks[linkType] = append(mergedLinks[linkType], mergedLink)
	}

	// 对每种类型的链接按时间排序（新的在前面）
	for linkType, links := range mergedLinks {
		sort.Slice(links, func(i, j int) bool {
			return links[i].Datetime.After(links[j].Datetime)
		})
		mergedLinks[linkType] = links
	}

	return mergedLinks
}

// searchTG 搜索TG频道
func (s *SearchService) searchTG(keyword string, channels []string, forceRefresh bool) ([]model.SearchResult, error) {
	// 生成缓存键
	cacheKey := cache.GenerateTGCacheKey(keyword, channels)
	
	// 如果未启用强制刷新，尝试从缓存获取结果
	if !forceRefresh && cacheInitialized && config.AppConfig.CacheEnabled {
		var data []byte
		var hit bool
		var err error
		
		// 优先使用增强版缓存
		if enhancedTwoLevelCache != nil {
			data, hit, err = enhancedTwoLevelCache.Get(cacheKey)
			
			if err == nil && hit {
				var results []model.SearchResult
				if err := enhancedTwoLevelCache.GetSerializer().Deserialize(data, &results); err == nil {
					return results, nil
				}
			}
		} else if twoLevelCache != nil {
			data, hit, err = twoLevelCache.Get(cacheKey)
			
			if err == nil && hit {
				var results []model.SearchResult
				if err := cache.DeserializeWithPool(data, &results); err == nil {
					return results, nil
				}
			}
		}
	}
	
	// 缓存未命中，执行实际搜索
	var results []model.SearchResult
	
	// 使用工作池并行搜索多个频道
	tasks := make([]pool.Task, 0, len(channels))
	
	for _, channel := range channels {
		ch := channel // 创建副本，避免闭包问题
		tasks = append(tasks, func() interface{} {
			results, err := s.searchChannel(keyword, ch)
			if err != nil {
				return nil
			}
			return results
		})
	}
	
	// 执行搜索任务并获取结果
	taskResults := pool.ExecuteBatchWithTimeout(tasks, len(channels), config.AppConfig.PluginTimeout)
	
	// 合并所有频道的结果
	for _, result := range taskResults {
		if result != nil {
			channelResults := result.([]model.SearchResult)
			results = append(results, channelResults...)
		}
	}
	
	// 异步缓存结果
	if cacheInitialized && config.AppConfig.CacheEnabled {
		go func(res []model.SearchResult) {
			ttl := time.Duration(config.AppConfig.CacheTTLMinutes) * time.Minute
			
			// 优先使用增强版缓存
			if enhancedTwoLevelCache != nil {
				data, err := enhancedTwoLevelCache.GetSerializer().Serialize(res)
				if err != nil {
					return
				}
				enhancedTwoLevelCache.Set(cacheKey, data, ttl)
			} else if twoLevelCache != nil {
				data, err := cache.SerializeWithPool(res)
				if err != nil {
					return
				}
				twoLevelCache.Set(cacheKey, data, ttl)
			}
		}(results)
	}
	
	return results, nil
}

// searchPlugins 搜索插件
func (s *SearchService) searchPlugins(keyword string, plugins []string, forceRefresh bool, concurrency int) ([]model.SearchResult, error) {
	// 生成缓存键
	cacheKey := cache.GeneratePluginCacheKey(keyword, plugins)
	
	// 如果未启用强制刷新，尝试从缓存获取结果
	if !forceRefresh && cacheInitialized && config.AppConfig.CacheEnabled {
		var data []byte
		var hit bool
		var err error
		
		// 优先使用增强版缓存
		if enhancedTwoLevelCache != nil {
			data, hit, err = enhancedTwoLevelCache.Get(cacheKey)
			
			if err == nil && hit {
				var results []model.SearchResult
				if err := enhancedTwoLevelCache.GetSerializer().Deserialize(data, &results); err == nil {
					// 确保缓存数据是最新的
					// 如果缓存数据是最近更新的（例如在过去30秒内），则直接返回
					// 否则，我们将重新执行搜索以获取最新数据
					if len(results) > 0 {
						// 获取当前时间
						now := time.Now()
						// 检查缓存数据是否是最近更新的
						// 这里我们假设如果缓存数据中有结果的时间戳在过去30秒内，则认为是最新的
						for _, result := range results {
							if !result.Datetime.IsZero() && now.Sub(result.Datetime) < 30*time.Second {
								return results, nil
							}
						}
					} else {
						// 如果缓存中没有数据，直接返回空结果
						return results, nil
					}
				}
			}
		} else if twoLevelCache != nil {
			data, hit, err = twoLevelCache.Get(cacheKey)
			
			if err == nil && hit {
				var results []model.SearchResult
				if err := cache.DeserializeWithPool(data, &results); err == nil {
					// 确保缓存数据是最新的
					// 如果缓存数据是最近更新的（例如在过去30秒内），则直接返回
					// 否则，我们将重新执行搜索以获取最新数据
					if len(results) > 0 {
						// 获取当前时间
						now := time.Now()
						// 检查缓存数据是否是最近更新的
						// 这里我们假设如果缓存数据中有结果的时间戳在过去30秒内，则认为是最新的
						for _, result := range results {
							if !result.Datetime.IsZero() && now.Sub(result.Datetime) < 30*time.Second {
								return results, nil
							}
						}
					} else {
						// 如果缓存中没有数据，直接返回空结果
						return results, nil
					}
				}
			}
		}
	}
	
	// 缓存未命中或缓存数据不是最新的，执行实际搜索
	// 获取所有可用插件
	var availablePlugins []plugin.SearchPlugin
	if s.pluginManager != nil {
		allPlugins := s.pluginManager.GetPlugins()
		
		// 确保plugins不为nil并且有非空元素
		hasPlugins := plugins != nil && len(plugins) > 0
		hasNonEmptyPlugin := false
		
		if hasPlugins {
			for _, p := range plugins {
				if p != "" {
					hasNonEmptyPlugin = true
					break
				}
			}
		}
		
		// 只有当plugins数组包含非空元素时才进行过滤
		if hasPlugins && hasNonEmptyPlugin {
			pluginMap := make(map[string]bool)
			for _, p := range plugins {
				if p != "" { // 忽略空字符串
					pluginMap[strings.ToLower(p)] = true
				}
			}
			
			for _, p := range allPlugins {
				if pluginMap[strings.ToLower(p.Name())] {
					availablePlugins = append(availablePlugins, p)
				}
			}
		} else {
			// 如果plugins为nil、空数组或只包含空字符串，视为未指定，使用所有插件
			availablePlugins = allPlugins
		}
	}
	
	// 控制并发数
	if concurrency <= 0 {
		concurrency = len(availablePlugins) + 10
		if concurrency < 1 {
			concurrency = 1
		}
	}
	
	// 使用工作池执行并行搜索
	tasks := make([]pool.Task, 0, len(availablePlugins))
	for _, p := range availablePlugins {
		plugin := p // 创建副本，避免闭包问题
		tasks = append(tasks, func() interface{} {
			results, err := plugin.Search(keyword)
			if err != nil {
				return nil
			}
			return results
		})
	}
	
	// 执行搜索任务并获取结果
	results := pool.ExecuteBatchWithTimeout(tasks, concurrency, config.AppConfig.PluginTimeout)
	
	// 合并所有插件的结果
	var allResults []model.SearchResult
	for _, result := range results {
		if result != nil {
			pluginResults := result.([]model.SearchResult)
			allResults = append(allResults, pluginResults...)
		}
	}
	
	// 异步缓存结果
	if cacheInitialized && config.AppConfig.CacheEnabled {
		go func(res []model.SearchResult) {
			ttl := time.Duration(config.AppConfig.CacheTTLMinutes) * time.Minute
			
			// 优先使用增强版缓存
			if enhancedTwoLevelCache != nil {
				data, err := enhancedTwoLevelCache.GetSerializer().Serialize(res)
				if err != nil {
					return
				}
				enhancedTwoLevelCache.Set(cacheKey, data, ttl)
			} else if twoLevelCache != nil {
				data, err := cache.SerializeWithPool(res)
				if err != nil {
					return
				}
				twoLevelCache.Set(cacheKey, data, ttl)
			}
		}(allResults)
	}
	
	return allResults, nil
}

// 合并搜索结果
func mergeSearchResults(tgResults, pluginResults []model.SearchResult) []model.SearchResult {
	// 预估合并后的结果数量
	totalSize := len(tgResults) + len(pluginResults)
	if totalSize == 0 {
		return []model.SearchResult{}
	}
	
	// 创建结果映射，用于去重
	resultMap := make(map[string]model.SearchResult, totalSize)
	
	// 添加TG搜索结果
	for _, result := range tgResults {
		resultMap[result.UniqueID] = result
	}
	
	// 添加或更新插件搜索结果（如果有重复，保留较新的）
	for _, result := range pluginResults {
		if existing, ok := resultMap[result.UniqueID]; ok {
			// 如果已存在，保留较新的
			if result.Datetime.After(existing.Datetime) {
				resultMap[result.UniqueID] = result
			}
		} else {
			resultMap[result.UniqueID] = result
		}
	}
	
	// 转换回切片
	mergedResults := make([]model.SearchResult, 0, len(resultMap))
	for _, result := range resultMap {
		mergedResults = append(mergedResults, result)
	}
	
	return mergedResults
}

// GetPluginManager 获取插件管理器
func (s *SearchService) GetPluginManager() *plugin.PluginManager {
	return s.pluginManager
}
