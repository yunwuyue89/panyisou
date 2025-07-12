package pansearch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"pansou/model"
	"pansou/plugin"
)

// 在init函数中注册插件
func init() {
	// 使用全局超时时间创建插件实例并注册
	plugin.RegisterGlobalPlugin(NewPanSearchPlugin())
}

const (
	// API基础URL - 完整URL，包含hash
	BaseURL = "https://www.pansearch.me/_next/data/267c2974d1894258fff4912af03ca830a831e353/search.json"
	
	// 默认参数
	DefaultTimeout = 6 * time.Second
	PageSize = 10
	MaxResults = 1000
	MaxConcurrent = 20
	MaxRetries = 2
)

// PanSearchPlugin 盘搜插件
type PanSearchPlugin struct {
	client        *http.Client
	timeout       time.Duration
	maxResults    int
	maxConcurrent int
	retries       int
}

// NewPanSearchPlugin 创建新的盘搜插件
func NewPanSearchPlugin() *PanSearchPlugin {
	timeout := DefaultTimeout
	
	return &PanSearchPlugin{
		client: &http.Client{
			Timeout: timeout,
		},
		timeout:       timeout,
		maxResults:    MaxResults,
		maxConcurrent: MaxConcurrent,
		retries:       MaxRetries,
	}
}

// Name 返回插件名称
func (p *PanSearchPlugin) Name() string {
	return "pansearch"
}

// Priority 返回插件优先级
func (p *PanSearchPlugin) Priority() int {
	return 2 // 较高优先级
}

// Search 执行搜索并返回结果
func (p *PanSearchPlugin) Search(keyword string) ([]model.SearchResult, error) {
	// 1. 发起首次请求获取total和第一页数据
	firstPageResults, total, err := p.fetchFirstPage(keyword)
	if err != nil {
		return nil, fmt.Errorf("获取首页失败: %w", err)
	}
	
	allResults := firstPageResults
	
	// 2. 计算需要的页数，但限制在最大结果数内
	remainingResults := min(total-PageSize, p.maxResults-PageSize)
	if remainingResults <= 0 {
		return p.convertResults(allResults), nil
	}
	
	neededPages := (remainingResults + PageSize - 1) / PageSize // 向上取整
	
	// 3. 创建工作池进行并发请求
	var wg sync.WaitGroup
	resultChan := make(chan []PanSearchItem, neededPages)
	errorChan := make(chan error, neededPages)
	
	// 创建信号量限制并发数
	semaphore := make(chan struct{}, p.maxConcurrent)
	
	// 分发任务
	for offset := PageSize; offset < PageSize+neededPages*PageSize; offset += PageSize {
		wg.Add(1)
		go func(offset int) {
			defer wg.Done()
			
			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			// 带重试的请求
			var pageResults []PanSearchItem
			var err error
			
			for retry := 0; retry <= p.retries; retry++ {
				pageResults, err = p.fetchPage(keyword, offset)
				if err == nil {
					break
				}
				
				if retry < p.retries {
					// 指数退避重试
					time.Sleep(time.Duration(1<<retry) * 100 * time.Millisecond)
				}
			}
			
			if err != nil {
				errorChan <- fmt.Errorf("获取偏移量 %d 的结果失败: %w", offset, err)
				return
			}
			
			resultChan <- pageResults
		}(offset)
	}
	
	// 等待所有请求完成
	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()
	
	// 收集结果
	for results := range resultChan {
		allResults = append(allResults, results...)
	}
	
	// 收集错误（但不中断处理）
	var errors []error
	for err := range errorChan {
		errors = append(errors, err)
	}
	
	// 如果所有请求都失败且没有获得首页以外的结果，则返回错误
	if len(errors) == neededPages && len(allResults) == len(firstPageResults) {
		return p.convertResults(allResults), fmt.Errorf("所有后续页面请求失败: %v", errors[0])
	}
	
	// 4. 去重和格式化结果
	uniqueResults := p.deduplicateItems(allResults)
	
	return p.convertResults(uniqueResults), nil
}

// fetchFirstPage 获取第一页结果和总数
func (p *PanSearchPlugin) fetchFirstPage(keyword string) ([]PanSearchItem, int, error) {
	// 构建请求URL
	reqURL := fmt.Sprintf("%s?keyword=%s&offset=0", BaseURL, url.QueryEscape(keyword))
	
	// 发送请求
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("创建请求失败: %w", err)
	}
	
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Referer", "https://www.pansearch.me/")
	
	// 发送请求
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()
	
	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("读取响应失败: %w", err)
	}
	
	// 解析响应
	var apiResp PanSearchResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, 0, fmt.Errorf("解析响应失败: %w", err)
	}
	
	// 获取total和结果
	total := apiResp.PageProps.Data.Total
	items := apiResp.PageProps.Data.Data
	
	return items, total, nil
}

// fetchPage 获取指定偏移量的页面
func (p *PanSearchPlugin) fetchPage(keyword string, offset int) ([]PanSearchItem, error) {
	// 构建请求URL
	reqURL := fmt.Sprintf("%s?keyword=%s&offset=%d", BaseURL, url.QueryEscape(keyword), offset)
	
	// 发送请求
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Referer", "https://www.pansearch.me/")
	
	// 发送请求
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()
	
	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	
	// 解析响应
	var apiResp PanSearchResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	
	return apiResp.PageProps.Data.Data, nil
}

// deduplicateItems 去重处理
func (p *PanSearchPlugin) deduplicateItems(items []PanSearchItem) []PanSearchItem {
	// 使用map进行去重，键为资源ID
	uniqueMap := make(map[int]PanSearchItem)
	
	for _, item := range items {
		uniqueMap[item.ID] = item
	}
	
	// 将map转回切片
	result := make([]PanSearchItem, 0, len(uniqueMap))
	for _, item := range uniqueMap {
		result = append(result, item)
	}
	
	return result
}

// convertResults 将API响应转换为标准SearchResult格式
func (p *PanSearchPlugin) convertResults(items []PanSearchItem) []model.SearchResult {
	results := make([]model.SearchResult, 0, len(items))
	
	for _, item := range items {
		// 提取链接和密码
		linkInfo := extractLinkAndPassword(item.Content)
		
		// 获取链接类型，确保映射到系统支持的类型
		linkType := item.Pan
		// 将aliyundrive映射到aliyun
		if linkType == "aliyundrive" {
			linkType = "aliyun"
		}
		
		// 创建链接
		link := model.Link{
			URL:      linkInfo.URL,
			Type:     linkType,
			Password: linkInfo.Password,
		}
		
		// 创建唯一ID
		uniqueID := fmt.Sprintf("pansearch-%d", item.ID)
		
		// 解析时间
		var datetime time.Time
		if item.Time != "" {
			// 尝试解析时间，格式：2025-07-07T13:54:43+08:00
			parsedTime, err := time.Parse(time.RFC3339, item.Time)
			if err == nil {
				datetime = parsedTime
			}
		}
		
		// 如果时间解析失败，使用零值
		if datetime.IsZero() {
			datetime = time.Time{}
		}
		
		// 清理内容中的HTML标签
		cleanedContent := cleanHTML(item.Content)
		
		// 创建搜索结果
		result := model.SearchResult{
			UniqueID:  uniqueID,
			Title:     extractTitle(item.Content),
			Content:   cleanedContent,
			Datetime:  datetime,
			Links:     []model.Link{link},
		}
		
		results = append(results, result)
	}
	
	return results
}

// LinkInfo 链接信息
type LinkInfo struct {
	URL      string
	Password string
}

// extractLinkAndPassword 从内容中提取链接和密码
func extractLinkAndPassword(content string) LinkInfo {
	// 实现从内容中提取链接和密码的逻辑
	// 这里需要解析HTML内容，提取<a>标签中的链接和密码
	// 简单实现，实际可能需要使用正则表达式或HTML解析库
	
	// 示例实现
	linkInfo := LinkInfo{}
	
	// 提取链接
	linkStartIndex := strings.Index(content, "href=\"")
	if linkStartIndex != -1 {
		linkStartIndex += 6 // "href="的长度
		linkEndIndex := strings.Index(content[linkStartIndex:], "\"")
		if linkEndIndex != -1 {
			linkInfo.URL = content[linkStartIndex : linkStartIndex+linkEndIndex]
		}
	}
	
	// 提取密码
	pwdIndex := strings.Index(content, "?pwd=")
	if pwdIndex != -1 {
		pwdStartIndex := pwdIndex + 5 // "?pwd="的长度
		pwdEndIndex := strings.Index(content[pwdStartIndex:], "\"")
		if pwdEndIndex != -1 {
			linkInfo.Password = content[pwdStartIndex : pwdStartIndex+pwdEndIndex]
		} else {
			// 可能是百度网盘链接结尾形式
			pwdEndIndex = strings.Index(content[pwdStartIndex:], "#")
			if pwdEndIndex != -1 {
				linkInfo.Password = content[pwdStartIndex : pwdStartIndex+pwdEndIndex]
			} else {
				// 取到结尾
				linkInfo.Password = content[pwdStartIndex:]
			}
		}
	}
	
	return linkInfo
}

// extractTitle 从内容中提取标题
func extractTitle(content string) string {
	// 实现从内容中提取标题的逻辑
	// 标题通常在"名称："之后
	titlePrefix := "名称："
	titleStartIndex := strings.Index(content, titlePrefix)
	if titleStartIndex == -1 {
		return "未知标题"
	}
	
	titleStartIndex += len(titlePrefix)
	titleEndIndex := strings.Index(content[titleStartIndex:], "\n")
	if titleEndIndex == -1 {
		return cleanHTML(content[titleStartIndex:])
	}
	
	return cleanHTML(content[titleStartIndex : titleStartIndex+titleEndIndex])
}

// cleanHTML 清理HTML标签
func cleanHTML(html string) string {
	// 实现清理HTML标签的逻辑
	// 这里简单实现，实际可能需要使用HTML解析库
	
	// 替换常见HTML标签
	replacements := map[string]string{
		"<span class='highlight-keyword'>": "",
		"</span>": "",
		"<a class=\"resource-link\" target=\"_blank\" href=\"": "",
		"</a>": "",
		"<br>": "\n",
		"<p>": "",
		"</p>": "\n",
	}
	
	result := html
	for tag, replacement := range replacements {
		result = strings.Replace(result, tag, replacement, -1)
	}
	
	// 清理其他HTML标签
	for {
		startIndex := strings.Index(result, "<")
		if startIndex == -1 {
			break
		}
		
		endIndex := strings.Index(result[startIndex:], ">")
		if endIndex == -1 {
			break
		}
		
		result = result[:startIndex] + result[startIndex+endIndex+1:]
	}
	
	return strings.TrimSpace(result)
}

// min 返回两个int中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// PanSearchResponse API响应结构
type PanSearchResponse struct {
	PageProps struct {
		Data struct {
			Total int            `json:"total"`
			Data  []PanSearchItem `json:"data"`
			Time  int            `json:"time"`
		} `json:"data"`
		Limit    int  `json:"limit"`
		IsMobile bool `json:"isMobile"`
	} `json:"pageProps"`
	NSSP bool `json:"__N_SSP"`
}

// PanSearchItem API响应中的单个结果项
type PanSearchItem struct {
	ID      int    `json:"id"`
	Content string `json:"content"`
	Pan     string `json:"pan"`
	Image   string `json:"image"`
	Time    string `json:"time"`
} 