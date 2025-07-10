package service

import (
	"io/ioutil"
	"sort"
	"strings"
	"time"

	"pansou/model"
	"pansou/util"
	"pansou/util/cache"
	"pansou/util/json"
	"pansou/util/pool"
	"pansou/config"
)

// 优先关键词列表
var priorityKeywords = []string{"全", "合集", "系列", "完整", "最新", "附"}

// 全局缓存实例和缓存是否初始化标志
var (
	twoLevelCache *cache.TwoLevelCache
	cacheInitialized bool
)

// 初始化缓存
func init() {
	if config.AppConfig != nil && config.AppConfig.CacheEnabled {
		var err error
		twoLevelCache, err = cache.NewTwoLevelCache()
		if err == nil {
			cacheInitialized = true
		}
	}
}

// SearchService 搜索服务
type SearchService struct{}

// NewSearchService 创建搜索服务实例并确保缓存可用
func NewSearchService() *SearchService {
	// 检查缓存是否已初始化，如果未初始化则尝试重新初始化
	if !cacheInitialized && config.AppConfig != nil && config.AppConfig.CacheEnabled {
		var err error
		twoLevelCache, err = cache.NewTwoLevelCache()
		if err == nil {
			cacheInitialized = true
		}
	}
	
	return &SearchService{}
}

// Search 执行搜索
func (s *SearchService) Search(keyword string, channels []string, concurrency int, forceRefresh bool, resultType string) (model.SearchResponse, error) {
	// 立即生成缓存键并检查缓存
	cacheKey := cache.GenerateCacheKey(keyword, nil)
	
	// 如果未启用强制刷新，尝试从缓存获取结果
	if !forceRefresh && twoLevelCache != nil && config.AppConfig.CacheEnabled {
		data, hit, err := twoLevelCache.Get(cacheKey)
		
		if err == nil && hit {
			var response model.SearchResponse
			if err := json.Unmarshal(data, &response); err == nil {
				// 根据resultType过滤返回结果
				return filterResponseByType(response, resultType), nil
			}
		}
	}
	
	// 控制并发数：如果用户没有指定有效值，则默认使用"频道数+10"的并发数
	if concurrency <= 0 {
		concurrency = len(channels) + 10
		if concurrency < 1 {
			concurrency = 1
		}
	}

	// 使用工作池执行并行搜索
	tasks := make([]pool.Task, len(channels))
	for i, channel := range channels {
		ch := channel // 创建副本，避免闭包问题
		tasks[i] = func() interface{} {
			results, err := s.searchChannel(keyword, ch)
			if err != nil {
				return nil
			}
			return results
		}
	}
	
	// 使用工作池执行所有任务并获取结果
	results := pool.ExecuteBatch(tasks, concurrency)
	
	// 预估每个频道平均返回22个结果
	allResults := make([]model.SearchResult, 0, len(channels)*22)
	
	// 合并所有结果
	for _, result := range results {
		if result != nil {
			channelResults := result.([]model.SearchResult)
			allResults = append(allResults, channelResults...)
		}
	}
	
	// 过滤结果，确保标题包含搜索关键词
	filteredResults := filterResultsByKeyword(allResults, keyword)
	
	// 按照优化后的规则排序结果
	sortResultsByTimeAndKeywords(filteredResults)
	
	// 合并链接按网盘类型分组
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
		total = len(filteredResults)
	}
	
	response := model.SearchResponse{
		Total:        total,
		Results:      filteredResults,
		MergedByType: mergedLinks,
	}

	// 异步缓存搜索结果（缓存完整结果，以便后续可以根据不同resultType过滤）
	if twoLevelCache != nil && config.AppConfig.CacheEnabled {
		go func(resp model.SearchResponse) {
			data, err := json.Marshal(resp)
			if err != nil {
				return
			}
			
			ttl := time.Duration(config.AppConfig.CacheTTLMinutes) * time.Minute
			twoLevelCache.Set(cacheKey, data, ttl)
		}(response)
	}
	
	// 根据resultType过滤返回结果
	return filterResponseByType(response, resultType), nil
}

// filterResponseByType 根据结果类型过滤响应
func filterResponseByType(response model.SearchResponse, resultType string) model.SearchResponse {
	switch resultType {
	case "results":
		// 只返回Results
		return model.SearchResponse{
			Total:   response.Total,
			Results: response.Results,
		}
	case "merged_by_type":
		// 只返回MergedByType
		return model.SearchResponse{
			Total:        response.Total,
			MergedByType: response.MergedByType,
		}
	default:
		// 默认返回全部
		return response
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
		// 将标题转为小写
		lowerTitle := strings.ToLower(result.Title)
		
		// 检查每个关键词是否在标题中
		matched := true
		for _, kw := range keywords {
			if !strings.Contains(lowerTitle, kw) {
				matched = false
				break
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
	
	// 获取HTTP客户端
	client := util.GetHTTPClient()
	
	// 发送请求
	resp, err := client.Get(url)
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