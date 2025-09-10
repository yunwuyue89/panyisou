package huban

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"context"
	"sync/atomic"

	"pansou/model"
	"pansou/plugin"
	"pansou/util/json"
)

const (
	// 默认超时时间 - 优化为更短时间
	DefaultTimeout = 8 * time.Second

	// HTTP连接池配置
	MaxIdleConns        = 200
	MaxIdleConnsPerHost = 50
	MaxConnsPerHost     = 100
	IdleConnTimeout     = 90 * time.Second
	
	// 请求来源控制 - 默认开启，提高安全性
	EnableRefererCheck = false
	
	// 调试日志开关
	DebugLog = false
)

// 性能统计（原子操作）
var (
	searchRequests  int64 = 0
	totalSearchTime int64 = 0 // 纳秒
)

// 请求来源控制配置
var (
	// 允许的请求来源列表 - 参考panyq插件实现
	// 支持前缀匹配，例如 "https://example.com" 会匹配 "https://example.com/path"
	AllowedReferers = []string{
		"https://dm.xueximeng.com",
		"http://localhost:8888",
		// 可以根据需要添加更多允许的来源
	}
)

func init() {
	plugin.RegisterGlobalPlugin(NewHubanPlugin())
}

// 预编译的正则表达式
var (
	// 密码提取正则表达式
	passwordRegex = regexp.MustCompile(`\?pwd=([0-9a-zA-Z]+)`)
	password115Regex = regexp.MustCompile(`password=([0-9a-zA-Z]+)`)
	
	// 常见网盘链接的正则表达式（支持16种类型）
	quarkLinkRegex     = regexp.MustCompile(`https?://pan\.quark\.cn/s/[0-9a-zA-Z]+`)
	ucLinkRegex        = regexp.MustCompile(`https?://drive\.uc\.cn/s/[0-9a-zA-Z]+(\?[^"'\s]*)?`)
	baiduLinkRegex     = regexp.MustCompile(`https?://pan\.baidu\.com/s/[0-9a-zA-Z_\-]+(\?pwd=[0-9a-zA-Z]+)?`)
	aliyunLinkRegex    = regexp.MustCompile(`https?://(www\.)?(aliyundrive\.com|alipan\.com)/s/[0-9a-zA-Z]+`)
	xunleiLinkRegex    = regexp.MustCompile(`https?://pan\.xunlei\.com/s/[0-9a-zA-Z_\-]+(\?pwd=[0-9a-zA-Z]+)?`)
	tianyiLinkRegex    = regexp.MustCompile(`https?://cloud\.189\.cn/t/[0-9a-zA-Z]+`)
	link115Regex       = regexp.MustCompile(`https?://(115\.com|115cdn\.com)/s/[0-9a-zA-Z]+`)
	mobileLinkRegex    = regexp.MustCompile(`https?://caiyun\.feixin\.10086\.cn/[0-9a-zA-Z]+`)
	weiyunLinkRegex    = regexp.MustCompile(`https?://share\.weiyun\.com/[0-9a-zA-Z]+`)
	lanzouLinkRegex    = regexp.MustCompile(`https?://(www\.)?(lanzou[uixys]*|lan[zs]o[ux])\.(com|net|org)/[0-9a-zA-Z]+`)
	jianguoyunLinkRegex = regexp.MustCompile(`https?://(www\.)?jianguoyun\.com/p/[0-9a-zA-Z]+`)
	link123Regex       = regexp.MustCompile(`https?://(123pan\.com|www\.123912\.com|www\.123865\.com|www\.123684\.com)/s/[0-9a-zA-Z]+`)
	pikpakLinkRegex    = regexp.MustCompile(`https?://mypikpak\.com/s/[0-9a-zA-Z]+`)
	magnetLinkRegex    = regexp.MustCompile(`magnet:\?xt=urn:btih:[0-9a-fA-F]{40}`)
	ed2kLinkRegex      = regexp.MustCompile(`ed2k://\|file\|.+\|\d+\|[0-9a-fA-F]{32}\|/`)
)

// HubanAsyncPlugin Huban异步插件
type HubanAsyncPlugin struct {
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
	}

	return &http.Client{
		Transport: transport,
		Timeout:   DefaultTimeout,
	}
}

// NewHubanPlugin 创建新的Huban异步插件
func NewHubanPlugin() *HubanAsyncPlugin {
	return &HubanAsyncPlugin{
		BaseAsyncPlugin: plugin.NewBaseAsyncPlugin("huban", 2),
		optimizedClient: createOptimizedHTTPClient(),
	}
}

// Search 同步搜索接口
func (p *HubanAsyncPlugin) Search(keyword string, ext map[string]interface{}) ([]model.SearchResult, error) {
	// 请求来源检查 - 参考panyq插件实现
	if EnableRefererCheck && ext != nil {
		referer := ""
		if refererVal, ok := ext["referer"].(string); ok {
			referer = refererVal
		}
		
		// 检查referer是否在允许列表中
		allowed := false
		for _, allowedReferer := range AllowedReferers {
			if strings.HasPrefix(referer, allowedReferer) {
				if DebugLog {
					fmt.Printf("[%s] 允许来自 %s 的请求\n", p.Name(), referer)
				}
				allowed = true
				break
			}
		}
		
		if !allowed {
			if DebugLog {
				fmt.Printf("[%s] 拒绝来自 %s 的请求\n", p.Name(), referer)
			}
			return nil, fmt.Errorf("[%s] 请求来源不被允许", p.Name())
		}
	}
	
	result, err := p.SearchWithResult(keyword, ext)
	if err != nil {
		return nil, err
	}
	return result.Results, nil
}

// SearchWithResult 带结果统计的搜索接口
func (p *HubanAsyncPlugin) SearchWithResult(keyword string, ext map[string]interface{}) (model.PluginSearchResult, error) {
	return p.AsyncSearchWithResult(keyword, p.searchImpl, p.MainCacheKey, ext)
}

// searchImpl 搜索实现（双域名支持）
func (p *HubanAsyncPlugin) searchImpl(client *http.Client, keyword string, ext map[string]interface{}) ([]model.SearchResult, error) {
	// 性能统计
	start := time.Now()
	atomic.AddInt64(&searchRequests, 1)
	defer func() {
		duration := time.Since(start).Nanoseconds()
		atomic.AddInt64(&totalSearchTime, duration)
	}()

	// 使用优化的客户端
	if p.optimizedClient != nil {
		client = p.optimizedClient
	}

	// 定义双域名 - 主备模式
	urls := []string{
		fmt.Sprintf("http://xsayang.fun:12512/api.php/provide/vod?ac=detail&wd=%s", url.QueryEscape(keyword)),
		fmt.Sprintf("http://103.45.162.207:20720/api.php/provide/vod?ac=detail&wd=%s", url.QueryEscape(keyword)),
	}
	
	// 主备模式：优先使用第一个域名，失败时切换到第二个
	for i, searchURL := range urls {
		if results, err := p.tryRequest(searchURL, client); err == nil {
			return results, nil
		} else if i == 0 {
			// 第一个域名失败，记录日志但继续尝试第二个
			// fmt.Printf("[%s] 域名1失败，尝试域名2: %v\n", p.Name(), err)
		}
	}
	
	return nil, fmt.Errorf("[%s] 所有域名都请求失败", p.Name())
}

// tryRequest 尝试单个域名请求
func (p *HubanAsyncPlugin) tryRequest(searchURL string, client *http.Client) ([]model.SearchResult, error) {
	// 创建HTTP请求
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建搜索请求失败: %w", err)
	}
	
	// 设置请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cache-Control", "no-cache")
	
	// 发送请求
	resp, err := p.doRequestWithRetry(req, client)
	if err != nil {
		return nil, fmt.Errorf("搜索请求失败: %w", err)
	}
	defer resp.Body.Close()
	
	// 解析JSON响应
	body, _ := io.ReadAll(resp.Body)
	
	var apiResponse HubanAPIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("解析JSON响应失败: %w", err)
	}
	
	// 检查API响应状态
	if apiResponse.Code != 1 {
		return nil, fmt.Errorf("API返回错误: %s", apiResponse.Msg)
	}
	
	// 解析搜索结果
	var results []model.SearchResult
	for _, item := range apiResponse.List {
		if result := p.parseAPIItem(item); result.Title != "" {
			results = append(results, result)
		}
	}
	
	return results, nil
}

// HubanAPIResponse API响应结构
type HubanAPIResponse struct {
	Code      int             `json:"code"`
	Msg       string          `json:"msg"`
	Page      int             `json:"page"`
	PageCount int             `json:"pagecount"`
	Limit     interface{}     `json:"limit"` // 可能是字符串或数字
	Total     int             `json:"total"`
	List      []HubanAPIItem  `json:"list"`
}

// HubanAPIItem API数据项
type HubanAPIItem struct {
	VodID        int    `json:"vod_id"`
	VodName      string `json:"vod_name"`
	VodActor     string `json:"vod_actor"`
	VodDirector  string `json:"vod_director"`
	VodDownFrom  string `json:"vod_down_from"`
	VodDownURL   string `json:"vod_down_url"`
	VodRemarks   string `json:"vod_remarks"`
	VodPubdate   string `json:"vod_pubdate"`
	VodArea      string `json:"vod_area"`
	VodLang      string `json:"vod_lang"`
	VodYear      string `json:"vod_year"`
	VodContent   string `json:"vod_content"`
	VodBlurb     string `json:"vod_blurb"`
	VodPic       string `json:"vod_pic"`
}

// parseAPIItem 解析API数据项
func (p *HubanAsyncPlugin) parseAPIItem(item HubanAPIItem) model.SearchResult {
	// 构建唯一ID
	uniqueID := fmt.Sprintf("%s-%d", p.Name(), item.VodID)
	
	// 构建标题
	title := strings.TrimSpace(item.VodName)
	if title == "" {
		return model.SearchResult{}
	}
	
	// 构建描述（需要清理数据）
	content := p.buildContent(item)
	
	// 解析下载链接（huban特殊格式）
	links := p.parseHubanLinks(item.VodDownFrom, item.VodDownURL)
	
	// 构建标签
	var tags []string
	if item.VodYear != "" {
		tags = append(tags, item.VodYear)
	}
	// area通常为空，不添加
	
	return model.SearchResult{
		UniqueID: uniqueID,
		Title:    title,
		Content:  content,
		Links:    links,
		Tags:     tags,
		Channel:  "", // 插件搜索结果Channel为空
		Datetime: time.Time{}, // 使用零值而不是nil，参考jikepan插件标准
	}
}

// buildContent 构建内容描述（清理特殊字符）
func (p *HubanAsyncPlugin) buildContent(item HubanAPIItem) string {
	var contentParts []string
	
	// 清理演员字段（移除前后逗号）
	if item.VodActor != "" {
		actor := strings.Trim(item.VodActor, ",")
		actor = strings.TrimSpace(actor)
		if actor != "" {
			contentParts = append(contentParts, fmt.Sprintf("主演: %s", actor))
		}
	}
	
	// 清理导演字段（移除前后逗号）
	if item.VodDirector != "" {
		director := strings.Trim(item.VodDirector, ",")
		director = strings.TrimSpace(director)
		if director != "" {
			contentParts = append(contentParts, fmt.Sprintf("导演: %s", director))
		}
	}
	
	if item.VodYear != "" {
		contentParts = append(contentParts, fmt.Sprintf("年份: %s", item.VodYear))
	}
	
	if item.VodRemarks != "" {
		contentParts = append(contentParts, fmt.Sprintf("状态: %s", item.VodRemarks))
	}
	
	return strings.Join(contentParts, " | ")
}

// parseHubanLinks 解析huban特殊格式的链接
func (p *HubanAsyncPlugin) parseHubanLinks(vodDownFrom, vodDownURL string) []model.Link {
	if vodDownFrom == "" || vodDownURL == "" {
		return nil
	}
	
	// 按$$$分隔网盘类型
	fromParts := strings.Split(vodDownFrom, "$$$")
	urlParts := strings.Split(vodDownURL, "$$$")
	
	var links []model.Link
	minLen := len(fromParts)
	if len(urlParts) < minLen {
		minLen = len(urlParts)
	}
	
	for i := 0; i < minLen; i++ {
		linkType := p.mapHubanCloudType(fromParts[i])
		if linkType == "" {
			continue
		}
		
		// 解析单个网盘类型的多个链接
		// 格式: "来源$链接1#标题1$链接2#标题2#"
		urlSection := urlParts[i]
		
		// 移除来源前缀（如"小虎斑$"）
		if strings.Contains(urlSection, "$") {
			urlSection = urlSection[strings.Index(urlSection, "$")+1:]
		}
		
		// 按#分隔多个链接
		linkParts := strings.Split(urlSection, "#")
		for j := 0; j < len(linkParts); j++ {
			linkURL := strings.TrimSpace(linkParts[j])
			
			// 跳过空链接和标题（标题通常不是链接格式）
			if linkURL == "" || !p.isValidNetworkDriveURL(linkURL) {
				continue
			}
			
			// 提取密码
			password := p.extractPassword(linkURL)
			
			links = append(links, model.Link{
				Type:     linkType,
				URL:      linkURL,
				Password: password,
			})
		}
	}
	
	// 去重（可能存在重复链接）
	return p.deduplicateLinks(links)
}

// mapHubanCloudType 映射huban特有的网盘标识符
func (p *HubanAsyncPlugin) mapHubanCloudType(apiType string) string {
	switch strings.ToUpper(apiType) {
	case "UCWP":
		return "uc"
	case "KKWP":
		return "quark"
	case "ALWP":
		return "aliyun"
	case "BDWP":
		return "baidu"
	case "123WP":
		return "123"
	case "115WP":
		return "115"
	case "TYWP":
		return "tianyi"
	case "XYWP":
		return "xunlei"
	case "WYWP":
		return "weiyun"
	case "LZWP":
		return "lanzou"
	case "JGYWP":
		return "jianguoyun"
	case "PKWP":
		return "pikpak"
	default:
		return ""
	}
}

// isValidNetworkDriveURL 检查URL是否为有效的网盘链接
func (p *HubanAsyncPlugin) isValidNetworkDriveURL(url string) bool {
	// 过滤掉明显无效的链接
	if strings.Contains(url, "javascript:") || 
	   url == "" ||
	   (!strings.HasPrefix(url, "http") && !strings.HasPrefix(url, "magnet:") && !strings.HasPrefix(url, "ed2k:")) {
		return false
	}
	
	// 检查是否匹配任何支持的网盘格式（16种）
	return quarkLinkRegex.MatchString(url) ||
		   ucLinkRegex.MatchString(url) ||
		   baiduLinkRegex.MatchString(url) ||
		   aliyunLinkRegex.MatchString(url) ||
		   xunleiLinkRegex.MatchString(url) ||
		   tianyiLinkRegex.MatchString(url) ||
		   link115Regex.MatchString(url) ||
		   mobileLinkRegex.MatchString(url) ||
		   weiyunLinkRegex.MatchString(url) ||
		   lanzouLinkRegex.MatchString(url) ||
		   jianguoyunLinkRegex.MatchString(url) ||
		   link123Regex.MatchString(url) ||
		   pikpakLinkRegex.MatchString(url) ||
		   magnetLinkRegex.MatchString(url) ||
		   ed2kLinkRegex.MatchString(url)
}

// determineLinkType 根据URL确定链接类型（支持16种类型）
func (p *HubanAsyncPlugin) determineLinkType(url string) string {
	switch {
	case quarkLinkRegex.MatchString(url):
		return "quark"
	case ucLinkRegex.MatchString(url):
		return "uc"
	case baiduLinkRegex.MatchString(url):
		return "baidu"
	case aliyunLinkRegex.MatchString(url):
		return "aliyun"
	case xunleiLinkRegex.MatchString(url):
		return "xunlei"
	case tianyiLinkRegex.MatchString(url):
		return "tianyi"
	case link115Regex.MatchString(url):
		return "115"
	case mobileLinkRegex.MatchString(url):
		return "mobile"
	case weiyunLinkRegex.MatchString(url):
		return "weiyun"
	case lanzouLinkRegex.MatchString(url):
		return "lanzou"
	case jianguoyunLinkRegex.MatchString(url):
		return "jianguoyun"
	case link123Regex.MatchString(url):
		return "123"
	case pikpakLinkRegex.MatchString(url):
		return "pikpak"
	case magnetLinkRegex.MatchString(url):
		return "magnet"
	case ed2kLinkRegex.MatchString(url):
		return "ed2k"
	default:
		return "" // 不支持的类型返回空字符串
	}
}

// extractPassword 从URL中提取密码
func (p *HubanAsyncPlugin) extractPassword(url string) string {
	// 百度网盘密码
	if matches := passwordRegex.FindStringSubmatch(url); len(matches) > 1 {
		return matches[1]
	}
	
	// 115网盘密码
	if matches := password115Regex.FindStringSubmatch(url); len(matches) > 1 {
		return matches[1]
	}
	
	return ""
}

// deduplicateLinks 去重链接
func (p *HubanAsyncPlugin) deduplicateLinks(links []model.Link) []model.Link {
	seen := make(map[string]bool)
	var result []model.Link
	
	for _, link := range links {
		key := fmt.Sprintf("%s-%s", link.Type, link.URL)
		if !seen[key] {
			seen[key] = true
			result = append(result, link)
		}
	}
	
	return result
}

// doRequestWithRetry 带重试的HTTP请求（优化JSON API的重试策略）
func (p *HubanAsyncPlugin) doRequestWithRetry(req *http.Request, client *http.Client) (*http.Response, error) {
	maxRetries := 2  // 对于JSON API减少重试次数
	var lastErr error
	
	for i := 0; i < maxRetries; i++ {
		resp, err := client.Do(req)
		if err == nil {
			if resp.StatusCode == http.StatusOK {
				return resp, nil
			}
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP状态码: %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		
		// JSON API快速重试：只等待很短时间
		if i < maxRetries-1 {
			time.Sleep(100 * time.Millisecond) // 从秒级改为100毫秒
		}
	}
	
	return nil, fmt.Errorf("[%s] 请求失败，重试%d次后仍失败: %w", p.Name(), maxRetries, lastErr)
}

// GetPerformanceStats 获取性能统计信息
func (p *HubanAsyncPlugin) GetPerformanceStats() map[string]interface{} {
	totalRequests := atomic.LoadInt64(&searchRequests)
	totalTime := atomic.LoadInt64(&totalSearchTime)
	
	var avgTime float64
	if totalRequests > 0 {
		avgTime = float64(totalTime) / float64(totalRequests) / 1e6 // 转换为毫秒
	}
	
	return map[string]interface{}{
		"search_requests":    totalRequests,
		"avg_search_time_ms": avgTime,
		"total_search_time_ns": totalTime,
	}
}

// AddAllowedReferer 添加允许的请求来源
func AddAllowedReferer(referer string) {
	for _, existing := range AllowedReferers {
		if existing == referer {
			return // 已存在，不重复添加
		}
	}
	AllowedReferers = append(AllowedReferers, referer)
}

// RemoveAllowedReferer 移除允许的请求来源
func RemoveAllowedReferer(referer string) {
	for i, existing := range AllowedReferers {
		if existing == referer {
			AllowedReferers = append(AllowedReferers[:i], AllowedReferers[i+1:]...)
			return
		}
	}
}

// GetAllowedReferers 获取当前允许的请求来源列表
func GetAllowedReferers() []string {
	result := make([]string, len(AllowedReferers))
	copy(result, AllowedReferers)
	return result
}

// IsRefererAllowed 检查指定的referer是否被允许
func IsRefererAllowed(referer string) bool {
	for _, allowedReferer := range AllowedReferers {
		if strings.HasPrefix(referer, allowedReferer) {
			return true
		}
	}
	return false
}