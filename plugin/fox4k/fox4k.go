package fox4k

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PuerkitoBio/goquery"
	"pansou/model"
	"pansou/plugin"
)

// 常量定义
const (
	// 搜索URL格式
	SearchURL = "https://www.4kfox.com/search/%s-------------.html"
	
	// 分页搜索URL格式
	SearchPageURL = "https://www.4kfox.com/search/%s----------%d---.html"
	
	// 详情页URL格式
	DetailURL = "https://www.4kfox.com/video/%s.html"
	
	// 默认超时时间 - 优化为更短时间
	DefaultTimeout = 8 * time.Second
	
	// 并发数限制 - 大幅提高并发数
	MaxConcurrency = 50
	
	// 最大分页数（避免无限请求）
	MaxPages = 10
	
	// HTTP连接池配置
	MaxIdleConns        = 200
	MaxIdleConnsPerHost = 50
	MaxConnsPerHost     = 100
	IdleConnTimeout     = 90 * time.Second
)

// 预编译正则表达式
var (
	// 从详情页URL中提取ID的正则表达式
	detailIDRegex = regexp.MustCompile(`/video/(\d+)\.html`)
	
	// 磁力链接的正则表达式
	magnetLinkRegex = regexp.MustCompile(`magnet:\?xt=urn:btih:[0-9a-fA-F]{40}[^"'\s]*`)
	
	// 电驴链接的正则表达式
	ed2kLinkRegex = regexp.MustCompile(`ed2k://\|file\|[^|]+\|[^|]+\|[^|]+\|/?`)
	
	// 年份提取正则表达式
	yearRegex = regexp.MustCompile(`(\d{4})`)
	
	// 网盘链接正则表达式（排除夸克）
	panLinkRegexes = map[string]*regexp.Regexp{
		"baidu":   regexp.MustCompile(`https?://pan\.baidu\.com/s/[0-9a-zA-Z_-]+(?:\?pwd=[0-9a-zA-Z]+)?(?:&v=\d+)?`),
		"aliyun":  regexp.MustCompile(`https?://(?:www\.)?alipan\.com/s/[0-9a-zA-Z_-]+`),
		"tianyi":  regexp.MustCompile(`https?://cloud\.189\.cn/t/[0-9a-zA-Z_-]+(?:\([^)]*\))?`),
		"uc":      regexp.MustCompile(`https?://drive\.uc\.cn/s/[0-9a-fA-F]+(?:\?[^"\s]*)?`),
		"mobile":  regexp.MustCompile(`https?://caiyun\.139\.com/[^"\s]+`),
		"115":     regexp.MustCompile(`https?://115\.com/s/[0-9a-zA-Z_-]+`),
		"pikpak":  regexp.MustCompile(`https?://mypikpak\.com/s/[0-9a-zA-Z_-]+`),
		"xunlei":  regexp.MustCompile(`https?://pan\.xunlei\.com/s/[0-9a-zA-Z_-]+(?:\?pwd=[0-9a-zA-Z]+)?`),
		"123":     regexp.MustCompile(`https?://(?:www\.)?123pan\.com/s/[0-9a-zA-Z_-]+`),
	}
	
	// 夸克网盘链接正则表达式（用于排除）
	quarkLinkRegex = regexp.MustCompile(`https?://pan\.quark\.cn/s/[0-9a-fA-F]+(?:\?pwd=[0-9a-zA-Z]+)?`)
	
	// 密码提取正则表达式
	passwordRegexes = []*regexp.Regexp{
		regexp.MustCompile(`\?pwd=([0-9a-zA-Z]+)`),                           // URL中的pwd参数
		regexp.MustCompile(`提取码[：:]\s*([0-9a-zA-Z]+)`),                    // 提取码：xxxx
		regexp.MustCompile(`访问码[：:]\s*([0-9a-zA-Z]+)`),                    // 访问码：xxxx
		regexp.MustCompile(`密码[：:]\s*([0-9a-zA-Z]+)`),                     // 密码：xxxx
		regexp.MustCompile(`（访问码[：:]\s*([0-9a-zA-Z]+)）`),                  // （访问码：xxxx）
	}
	
	// 缓存相关
	detailCache     = sync.Map{} // 缓存详情页解析结果
	lastCleanupTime = time.Now()
	cacheTTL        = 1 * time.Hour // 缩短缓存时间
	
	// 性能统计（原子操作）
	searchRequests     int64 = 0
	detailPageRequests int64 = 0
	cacheHits          int64 = 0
	cacheMisses        int64 = 0
	totalSearchTime    int64 = 0 // 纳秒
	totalDetailTime    int64 = 0 // 纳秒
)

// 缓存的详情页响应
type detailPageResponse struct {
	Title     string
	ImageURL  string
	Downloads []model.Link
	Tags      []string
	Content   string
	Timestamp time.Time
}

// Fox4kPlugin 极狐4K搜索插件
type Fox4kPlugin struct {
	*plugin.BaseAsyncPlugin
	optimizedClient *http.Client
}

// createOptimizedHTTPClient 创建优化的HTTP客户端
func createOptimizedHTTPClient() *http.Client {
	transport := &http.Transport{
		MaxIdleConns:        MaxIdleConns,
		MaxIdleConnsPerHost: MaxIdleConnsPerHost,
		MaxConnsPerHost:     MaxConnsPerHost,
		IdleConnTimeout:     IdleConnTimeout,
		DisableKeepAlives:   false,
		DisableCompression:  false,
		WriteBufferSize:     16 * 1024,
		ReadBufferSize:      16 * 1024,
	}
	
	return &http.Client{
		Transport: transport,
		Timeout:   DefaultTimeout,
	}
}

// NewFox4kPlugin 创建新的极狐4K搜索异步插件
func NewFox4kPlugin() *Fox4kPlugin {
	return &Fox4kPlugin{
		BaseAsyncPlugin: plugin.NewBaseAsyncPlugin("fox4k", 3), 
		optimizedClient: createOptimizedHTTPClient(),
	}
}

// 初始化插件
func init() {
	plugin.RegisterGlobalPlugin(NewFox4kPlugin())
	
	// 启动缓存清理
	go startCacheCleaner()
}

// startCacheCleaner 定期清理缓存
func startCacheCleaner() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		// 清空详情页缓存
		detailCache = sync.Map{}
		lastCleanupTime = time.Now()
	}
}

// Search 执行搜索并返回结果（兼容性方法）
func (p *Fox4kPlugin) Search(keyword string, ext map[string]interface{}) ([]model.SearchResult, error) {
	result, err := p.SearchWithResult(keyword, ext)
	if err != nil {
		return nil, err
	}
	return result.Results, nil
}

// SearchWithResult 执行搜索并返回包含IsFinal标记的结果
func (p *Fox4kPlugin) SearchWithResult(keyword string, ext map[string]interface{}) (model.PluginSearchResult, error) {
	return p.AsyncSearchWithResult(keyword, p.searchImpl, p.MainCacheKey, ext)
}

// searchImpl 实现具体的搜索逻辑（支持分页）
func (p *Fox4kPlugin) searchImpl(client *http.Client, keyword string, ext map[string]interface{}) ([]model.SearchResult, error) {
	startTime := time.Now()
	atomic.AddInt64(&searchRequests, 1)
	
	// 使用优化的客户端
	if p.optimizedClient != nil {
		client = p.optimizedClient
	}
	
	encodedKeyword := url.QueryEscape(keyword)
	allResults := make([]model.SearchResult, 0)
	
	// 1. 搜索第一页
	firstPageResults, totalPages, err := p.searchPage(client, encodedKeyword, 1)
	if err != nil {
		return nil, err
	}
	allResults = append(allResults, firstPageResults...)
	
	// 2. 如果有多页，继续搜索其他页面（限制最大页数）
	maxPagesToSearch := totalPages
	if maxPagesToSearch > MaxPages {
		maxPagesToSearch = MaxPages
	}
	
	if totalPages > 1 && maxPagesToSearch > 1 {
		// 并发搜索其他页面
		var wg sync.WaitGroup
		var mu sync.Mutex
		results := make([][]model.SearchResult, maxPagesToSearch-1)
		
		for page := 2; page <= maxPagesToSearch; page++ {
			wg.Add(1)
			go func(pageNum int) {
				defer wg.Done()
				pageResults, _, err := p.searchPage(client, encodedKeyword, pageNum)
				if err == nil {
					mu.Lock()
					results[pageNum-2] = pageResults
					mu.Unlock()
				}
			}(page)
		}
		
		wg.Wait()
		
		// 合并所有页面的结果
		for _, pageResults := range results {
			allResults = append(allResults, pageResults...)
		}
	}
	
	// 3. 并发获取详情页信息
	allResults = p.enrichWithDetailInfo(allResults, client)
	
	// 4. 过滤关键词匹配的结果
	results := plugin.FilterResultsByKeyword(allResults, keyword)
	
	// 记录性能统计
	searchDuration := time.Since(startTime)
	atomic.AddInt64(&totalSearchTime, int64(searchDuration))
	
	return results, nil
}

// searchPage 搜索指定页面
func (p *Fox4kPlugin) searchPage(client *http.Client, encodedKeyword string, page int) ([]model.SearchResult, int, error) {
	// 1. 构建搜索URL
	var searchURL string
	if page == 1 {
		searchURL = fmt.Sprintf(SearchURL, encodedKeyword)
	} else {
		searchURL = fmt.Sprintf(SearchPageURL, encodedKeyword, page)
	}
	
	// 2. 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	
	// 3. 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("[%s] 创建请求失败: %w", p.Name(), err)
	}
	
	// 4. 设置完整的请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Referer", "https://www.4kfox.com/")
	
	// 5. 发送HTTP请求
	resp, err := p.doRequestWithRetry(req, client)
	if err != nil {
		return nil, 0, fmt.Errorf("[%s] 第%d页搜索请求失败: %w", p.Name(), page, err)
	}
	defer resp.Body.Close()
	
	// 6. 检查状态码
	if resp.StatusCode != 200 {
		return nil, 0, fmt.Errorf("[%s] 第%d页请求返回状态码: %d", p.Name(), page, resp.StatusCode)
	}
	
	// 7. 解析HTML响应
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("[%s] 第%d页HTML解析失败: %w", p.Name(), page, err)
	}
	
	// 8. 解析分页信息
	totalPages := p.parseTotalPages(doc)
	
	// 9. 提取搜索结果
	results := make([]model.SearchResult, 0)
	doc.Find(".hl-list-item").Each(func(i int, s *goquery.Selection) {
		result := p.parseSearchResultItem(s)
		if result != nil {
			results = append(results, *result)
		}
	})
	
	return results, totalPages, nil
}

// parseTotalPages 解析总页数
func (p *Fox4kPlugin) parseTotalPages(doc *goquery.Document) int {
	// 查找分页信息，格式为 "1 / 2"
	pageInfo := doc.Find(".hl-page-tips a").Text()
	if pageInfo == "" {
		return 1
	}
	
	// 解析 "1 / 2" 格式
	parts := strings.Split(pageInfo, "/")
	if len(parts) != 2 {
		return 1
	}
	
	totalPagesStr := strings.TrimSpace(parts[1])
	totalPages, err := strconv.Atoi(totalPagesStr)
	if err != nil || totalPages < 1 {
		return 1
	}
	
	return totalPages
}

// parseSearchResultItem 解析单个搜索结果项
func (p *Fox4kPlugin) parseSearchResultItem(s *goquery.Selection) *model.SearchResult {
	// 获取详情页链接
	linkElement := s.Find(".hl-item-pic a").First()
	href, exists := linkElement.Attr("href")
	if !exists || href == "" {
		return nil
	}
	
	// 补全URL
	if strings.HasPrefix(href, "/") {
		href = "https://www.4kfox.com" + href
	}
	
	// 提取ID
	matches := detailIDRegex.FindStringSubmatch(href)
	if len(matches) < 2 {
		return nil
	}
	id := matches[1]
	
	// 获取标题
	titleElement := s.Find(".hl-item-title a").First()
	title := strings.TrimSpace(titleElement.Text())
	if title == "" {
		return nil
	}
	
	// 获取封面图片
	imgElement := s.Find(".hl-item-thumb")
	imageURL, _ := imgElement.Attr("data-original")
	if imageURL != "" && strings.HasPrefix(imageURL, "/") {
		imageURL = "https://www.4kfox.com" + imageURL
	}
	
	// 获取资源状态
	status := strings.TrimSpace(s.Find(".hl-pic-text .remarks").Text())
	
	// 获取评分
	score := strings.TrimSpace(s.Find(".hl-text-conch.score").Text())
	
	// 获取基本信息（年份、地区、类型）
	basicInfo := strings.TrimSpace(s.Find(".hl-item-sub").First().Text())
	
	// 获取简介
	description := strings.TrimSpace(s.Find(".hl-item-sub").Last().Text())
	
	// 解析年份、地区、类型
	var year, region, category string
	if basicInfo != "" {
		parts := strings.Split(basicInfo, "·")
		for i, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			
			// 跳过评分
			if strings.Contains(part, score) {
				continue
			}
			
			// 第一个通常是年份
			if i == 0 || (i == 1 && strings.Contains(parts[0], score)) {
				if yearRegex.MatchString(part) {
					year = part
				}
			} else if region == "" {
				region = part
			} else if category == "" {
				category = part
			} else {
				category += " " + part
			}
		}
	}
	
	// 构建标签
	tags := make([]string, 0)
	if status != "" {
		tags = append(tags, status)
	}
	if year != "" {
		tags = append(tags, year)
	}
	if region != "" {
		tags = append(tags, region)
	}
	if category != "" {
		tags = append(tags, category)
	}
	
	// 构建内容描述
	content := description
	if basicInfo != "" {
		content = basicInfo + "\n" + description
	}
	if score != "" {
		content = "评分: " + score + "\n" + content
	}
	
	return &model.SearchResult{
		UniqueID: fmt.Sprintf("%s-%s", p.Name(), id),
		Title:    title,
		Content:  content,
		Datetime: time.Time{}, // 使用零值而不是nil，参考jikepan插件标准
		Tags:     tags,
		Links:    []model.Link{}, // 初始为空，后续在详情页中填充
		Channel:  "",             // 插件搜索结果，Channel必须为空
	}
}

// enrichWithDetailInfo 并发获取详情页信息并丰富搜索结果
func (p *Fox4kPlugin) enrichWithDetailInfo(results []model.SearchResult, client *http.Client) []model.SearchResult {
	if len(results) == 0 {
		return results
	}
	
	// 使用信号量控制并发数
	semaphore := make(chan struct{}, MaxConcurrency)
	var wg sync.WaitGroup
	var mutex sync.Mutex
	
	enrichedResults := make([]model.SearchResult, len(results))
	copy(enrichedResults, results)
	
	for i := range enrichedResults {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			
			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			// 从UniqueID中提取ID
			parts := strings.Split(enrichedResults[index].UniqueID, "-")
			if len(parts) < 2 {
				return
			}
			id := parts[len(parts)-1]
			
			// 获取详情页信息
			detailInfo := p.getDetailInfo(id, client)
			if detailInfo != nil {
				mutex.Lock()
				enrichedResults[index].Links = detailInfo.Downloads
				if detailInfo.Content != "" {
					enrichedResults[index].Content = detailInfo.Content
				}
				// 补充标签
				for _, tag := range detailInfo.Tags {
					found := false
					for _, existingTag := range enrichedResults[index].Tags {
						if existingTag == tag {
							found = true
							break
						}
					}
					if !found {
						enrichedResults[index].Tags = append(enrichedResults[index].Tags, tag)
					}
				}
				mutex.Unlock()
			}
		}(i)
	}
	
	wg.Wait()
	
	// 过滤掉没有有效下载链接的结果
	var validResults []model.SearchResult
	for _, result := range enrichedResults {
		if len(result.Links) > 0 {
			validResults = append(validResults, result)
		}
	}
	
	return validResults
}

// getDetailInfo 获取详情页信息
func (p *Fox4kPlugin) getDetailInfo(id string, client *http.Client) *detailPageResponse {
	startTime := time.Now()
	atomic.AddInt64(&detailPageRequests, 1)
	
	// 检查缓存
	if cached, ok := detailCache.Load(id); ok {
		if detail, ok := cached.(*detailPageResponse); ok {
			if time.Since(detail.Timestamp) < cacheTTL {
				atomic.AddInt64(&cacheHits, 1)
				return detail
			}
		}
	}
	
	// 缓存未命中
	atomic.AddInt64(&cacheMisses, 1)
	
	// 构建详情页URL
	detailURL := fmt.Sprintf(DetailURL, id)
	
	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	
	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", detailURL, nil)
	if err != nil {
		return nil
	}
	
	// 设置请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", "https://www.4kfox.com/")
	
	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return nil
	}
	
	// 解析HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil
	}
	
	// 解析详情页信息
	detail := &detailPageResponse{
		Downloads: make([]model.Link, 0),
		Tags:      make([]string, 0),
		Timestamp: time.Now(),
	}
	
	// 获取标题
	detail.Title = strings.TrimSpace(doc.Find("h2.hl-dc-title").Text())
	
	// 获取封面图片
	imgElement := doc.Find(".hl-dc-pic .hl-item-thumb")
	if imageURL, exists := imgElement.Attr("data-original"); exists && imageURL != "" {
		if strings.HasPrefix(imageURL, "/") {
			imageURL = "https://www.4kfox.com" + imageURL
		}
		detail.ImageURL = imageURL
	}
	
	// 获取剧情简介
	detail.Content = strings.TrimSpace(doc.Find(".hl-content-wrap .hl-content-text").Text())
	
	// 提取详细信息作为标签
	doc.Find(".hl-vod-data ul li").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			// 清理标签文本
			text = strings.ReplaceAll(text, "：", ": ")
			if strings.Contains(text, "类型:") || strings.Contains(text, "地区:") || strings.Contains(text, "语言:") {
				detail.Tags = append(detail.Tags, text)
			}
		}
	})
	
	// 提取下载链接
	p.extractDownloadLinks(doc, detail)
	
	// 缓存结果
	detailCache.Store(id, detail)
	
	// 记录性能统计
	detailDuration := time.Since(startTime)
	atomic.AddInt64(&totalDetailTime, int64(detailDuration))
	
	return detail
}

// GetPerformanceStats 获取性能统计信息（调试用）
func (p *Fox4kPlugin) GetPerformanceStats() map[string]interface{} {
	totalSearches := atomic.LoadInt64(&searchRequests)
	totalDetails := atomic.LoadInt64(&detailPageRequests)
	hits := atomic.LoadInt64(&cacheHits)
	misses := atomic.LoadInt64(&cacheMisses)
	searchTime := atomic.LoadInt64(&totalSearchTime)
	detailTime := atomic.LoadInt64(&totalDetailTime)
	
	stats := map[string]interface{}{
		"search_requests":      totalSearches,
		"detail_page_requests": totalDetails,
		"cache_hits":           hits,
		"cache_misses":         misses,
		"cache_hit_rate":       float64(hits) / float64(hits+misses) * 100,
	}
	
	if totalSearches > 0 {
		stats["avg_search_time_ms"] = float64(searchTime) / float64(totalSearches) / 1000000
	}
	if totalDetails > 0 {
		stats["avg_detail_time_ms"] = float64(detailTime) / float64(totalDetails) / 1000000
	}
	
	return stats
}

// extractDownloadLinks 提取下载链接（包括磁力链接、电驴链接和网盘链接）
func (p *Fox4kPlugin) extractDownloadLinks(doc *goquery.Document, detail *detailPageResponse) {
	// 提取页面中所有文本内容，寻找链接
	pageText := doc.Text()
	
	// 1. 提取磁力链接
	magnetMatches := magnetLinkRegex.FindAllString(pageText, -1)
	for _, magnetLink := range magnetMatches {
		p.addDownloadLink(detail, "magnet", magnetLink, "")
	}
	
	// 2. 提取电驴链接
	ed2kMatches := ed2kLinkRegex.FindAllString(pageText, -1)
	for _, ed2kLink := range ed2kMatches {
		p.addDownloadLink(detail, "ed2k", ed2kLink, "")
	}
	
	// 3. 提取网盘链接（排除夸克）
	for panType, regex := range panLinkRegexes {
		matches := regex.FindAllString(pageText, -1)
		for _, panLink := range matches {
			// 提取密码（如果有）
			password := p.extractPasswordFromText(pageText, panLink)
			p.addDownloadLink(detail, panType, panLink, password)
		}
	}
	
	// 4. 在特定的下载区域查找链接
	doc.Find(".hl-rb-downlist").Each(func(i int, downlistSection *goquery.Selection) {
		// 获取质量版本信息
		var currentQuality string
		downlistSection.Find(".hl-tabs-btn").Each(func(j int, tabBtn *goquery.Selection) {
			if tabBtn.HasClass("active") {
				currentQuality = strings.TrimSpace(tabBtn.Text())
			}
		})
		
		// 提取各种下载链接
		downlistSection.Find(".hl-downs-list li").Each(func(k int, linkItem *goquery.Selection) {
			itemText := linkItem.Text()
			itemHTML, _ := linkItem.Html()
			
			// 从 data-clipboard-text 属性提取链接
			if clipboardText, exists := linkItem.Find(".down-copy").Attr("data-clipboard-text"); exists {
				p.processFoundLink(detail, clipboardText, currentQuality)
			}
			
			// 从 href 属性提取链接
			linkItem.Find("a").Each(func(l int, link *goquery.Selection) {
				if href, exists := link.Attr("href"); exists {
					p.processFoundLink(detail, href, currentQuality)
				}
			})
			
			// 从文本内容中提取链接
			p.extractLinksFromText(detail, itemText, currentQuality)
			p.extractLinksFromText(detail, itemHTML, currentQuality)
		})
	})
	
	// 5. 在播放源区域也查找链接
	doc.Find(".hl-rb-playlist").Each(func(i int, playlistSection *goquery.Selection) {
		sectionText := playlistSection.Text()
		sectionHTML, _ := playlistSection.Html()
		p.extractLinksFromText(detail, sectionText, "播放源")
		p.extractLinksFromText(detail, sectionHTML, "播放源")
	})
}

// processFoundLink 处理找到的链接
func (p *Fox4kPlugin) processFoundLink(detail *detailPageResponse, link, quality string) {
	if link == "" {
		return
	}
	
	// 排除夸克网盘链接
	if quarkLinkRegex.MatchString(link) {
		return
	}
	
	// 检查磁力链接
	if magnetLinkRegex.MatchString(link) {
		p.addDownloadLink(detail, "magnet", link, "")
		return
	}
	
	// 检查电驴链接
	if ed2kLinkRegex.MatchString(link) {
		p.addDownloadLink(detail, "ed2k", link, "")
		return
	}
	
	// 检查网盘链接
	for panType, regex := range panLinkRegexes {
		if regex.MatchString(link) {
			password := p.extractPasswordFromLink(link)
			p.addDownloadLink(detail, panType, link, password)
			return
		}
	}
}

// extractLinksFromText 从文本中提取各种类型的链接
func (p *Fox4kPlugin) extractLinksFromText(detail *detailPageResponse, text, quality string) {
	// 排除包含夸克链接的文本
	if quarkLinkRegex.MatchString(text) {
		// 如果文本中有夸克链接，我们跳过整个文本块
		// 这是因为通常一个区域要么是夸克专区，要么不是
		return
	}
	
	// 磁力链接
	magnetMatches := magnetLinkRegex.FindAllString(text, -1)
	for _, magnetLink := range magnetMatches {
		p.addDownloadLink(detail, "magnet", magnetLink, "")
	}
	
	// 电驴链接
	ed2kMatches := ed2kLinkRegex.FindAllString(text, -1)
	for _, ed2kLink := range ed2kMatches {
		p.addDownloadLink(detail, "ed2k", ed2kLink, "")
	}
	
	// 网盘链接
	for panType, regex := range panLinkRegexes {
		matches := regex.FindAllString(text, -1)
		for _, panLink := range matches {
			password := p.extractPasswordFromText(text, panLink)
			p.addDownloadLink(detail, panType, panLink, password)
		}
	}
}

// extractPasswordFromLink 从链接URL中提取密码
func (p *Fox4kPlugin) extractPasswordFromLink(link string) string {
	// 首先检查URL参数中的密码
	for _, regex := range passwordRegexes {
		if matches := regex.FindStringSubmatch(link); len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

// extractPasswordFromText 从文本中提取指定链接的密码
func (p *Fox4kPlugin) extractPasswordFromText(text, link string) string {
	// 首先从链接本身提取密码
	if password := p.extractPasswordFromLink(link); password != "" {
		return password
	}
	
	// 然后从周围文本中查找密码
	for _, regex := range passwordRegexes {
		if matches := regex.FindStringSubmatch(text); len(matches) > 1 {
			return matches[1]
		}
	}
	
	return ""
}

// addDownloadLink 添加下载链接
func (p *Fox4kPlugin) addDownloadLink(detail *detailPageResponse, linkType, linkURL, password string) {
	if linkURL == "" {
		return
	}
	
	// 跳过夸克网盘链接
	if quarkLinkRegex.MatchString(linkURL) {
		return
	}
	
	// 检查是否已存在
	for _, existingLink := range detail.Downloads {
		if existingLink.URL == linkURL {
			return
		}
	}
	
	// 创建链接对象
	link := model.Link{
		Type:     linkType,
		URL:      linkURL,
		Password: password,
	}
	
	detail.Downloads = append(detail.Downloads, link)
}

// doRequestWithRetry 带重试机制的HTTP请求
func (p *Fox4kPlugin) doRequestWithRetry(req *http.Request, client *http.Client) (*http.Response, error) {
	maxRetries := 3
	var lastErr error
	
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			// 指数退避重试
			backoff := time.Duration(1<<uint(i-1)) * 200 * time.Millisecond
			time.Sleep(backoff)
		}
		
		// 克隆请求避免并发问题
		reqClone := req.Clone(req.Context())
		
		resp, err := client.Do(reqClone)
		if err == nil && resp.StatusCode == 200 {
			return resp, nil
		}
		
		if resp != nil {
			resp.Body.Close()
		}
		lastErr = err
	}
	
	return nil, fmt.Errorf("重试 %d 次后仍然失败: %w", maxRetries, lastErr)
}