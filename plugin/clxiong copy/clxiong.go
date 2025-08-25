package clxiong

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"pansou/model"
	"pansou/plugin"
)

const (
	BaseURL       = "https://www.cilixiong.org"
	SearchURL     = "https://www.cilixiong.org/e/search/index.php"
	UserAgent     = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"
	MaxRetries    = 3
	RetryDelay    = 2 * time.Second
	MaxResults    = 30
)

// ClxiongPlugin 磁力熊插件
type ClxiongPlugin struct {
	*plugin.BaseAsyncPlugin
	debugMode bool
}

func init() {
	p := &ClxiongPlugin{
		BaseAsyncPlugin: plugin.NewBaseAsyncPluginWithFilter("clxiong", 1, true), 
		debugMode:       true,
	}
	plugin.RegisterGlobalPlugin(p)
}

// Search 搜索接口实现
func (p *ClxiongPlugin) Search(keyword string, ext map[string]interface{}) ([]model.SearchResult, error) {
	result, err := p.SearchWithResult(keyword, ext)
	if err != nil {
		return nil, err
	}
	return result.Results, nil
}

// SearchWithResult 搜索并返回详细结果
func (p *ClxiongPlugin) SearchWithResult(keyword string, ext map[string]interface{}) (*model.PluginSearchResult, error) {
	if p.debugMode {
		log.Printf("[CLXIONG] 开始搜索: %s", keyword)
	}

	// 第一步：POST搜索获取searchid
	searchID, err := p.getSearchID(keyword)
	if err != nil {
		if p.debugMode {
			log.Printf("[CLXIONG] 获取searchid失败: %v", err)
		}
		return nil, fmt.Errorf("获取searchid失败: %v", err)
	}

	// 第二步：GET搜索结果
	results, err := p.getSearchResults(searchID, keyword)
	if err != nil {
		if p.debugMode {
			log.Printf("[CLXIONG] 获取搜索结果失败: %v", err)
		}
		return nil, err
	}

	// 第三步：异步获取详情页磁力链接
	p.fetchDetailLinksAsync(results)

	if p.debugMode {
		log.Printf("[CLXIONG] 搜索完成，获得 %d 个结果", len(results))
	}

	// 应用关键词过滤
	fmt.Printf("results: %v\n", results)
	filteredResults := plugin.FilterResultsByKeyword(results, keyword)

	return &model.PluginSearchResult{
		Results:   filteredResults,
		IsFinal:   true,
		Timestamp: time.Now(),
		Source:    p.Name(),
		Message:   fmt.Sprintf("找到 %d 个结果", len(filteredResults)),
	}, nil
}

// getSearchID 第一步：POST搜索获取searchid
func (p *ClxiongPlugin) getSearchID(keyword string) (string, error) {
	if p.debugMode {
		log.Printf("[CLXIONG] 正在获取searchid...")
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// 不自动跟随重定向，我们需要手动处理
			return http.ErrUseLastResponse
		},
	}

	// 准备POST数据
	formData := url.Values{}
	formData.Set("classid", "1,2")      // 1=电影，2=剧集
	formData.Set("show", "title")       // 搜索字段
	formData.Set("tempid", "1")         // 模板ID
	formData.Set("keyboard", keyword)   // 搜索关键词

	req, err := http.NewRequest("POST", SearchURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", BaseURL+"/")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")

	var resp *http.Response
	var lastErr error

	// 重试机制
	for i := 0; i < MaxRetries; i++ {
		resp, lastErr = client.Do(req)
		if lastErr == nil && (resp.StatusCode == 302 || resp.StatusCode == 301) {
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
		if i < MaxRetries-1 {
			time.Sleep(RetryDelay)
		}
	}

	if lastErr != nil {
		return "", lastErr
	}
	defer resp.Body.Close()

	// 检查重定向响应
	if resp.StatusCode != 302 && resp.StatusCode != 301 {
		return "", fmt.Errorf("期望302重定向，但得到状态码: %d", resp.StatusCode)
	}

	// 从Location头部提取searchid
	location := resp.Header.Get("Location")
	if location == "" {
		return "", fmt.Errorf("重定向响应中没有Location头部")
	}

	// 解析searchid
	searchID := p.extractSearchIDFromLocation(location)
	if searchID == "" {
		return "", fmt.Errorf("无法从Location中提取searchid: %s", location)
	}

	if p.debugMode {
		log.Printf("[CLXIONG] 获取到searchid: %s", searchID)
	}

	return searchID, nil
}

// extractSearchIDFromLocation 从Location头部提取searchid
func (p *ClxiongPlugin) extractSearchIDFromLocation(location string) string {
	// location格式: "result/?searchid=7549"
	re := regexp.MustCompile(`searchid=(\d+)`)
	matches := re.FindStringSubmatch(location)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// getSearchResults 第二步：GET搜索结果
func (p *ClxiongPlugin) getSearchResults(searchID, keyword string) ([]model.SearchResult, error) {
	if p.debugMode {
		log.Printf("[CLXIONG] 正在获取搜索结果，searchid: %s", searchID)
	}

	// 构建结果页URL
	resultURL := fmt.Sprintf("%s/e/search/result/?searchid=%s", BaseURL, searchID)

	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("GET", resultURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Referer", BaseURL+"/")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")

	var resp *http.Response
	var lastErr error

	// 重试机制
	for i := 0; i < MaxRetries; i++ {
		resp, lastErr = client.Do(req)
		if lastErr == nil && resp.StatusCode == 200 {
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
		if i < MaxRetries-1 {
			time.Sleep(RetryDelay)
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("搜索结果请求失败，状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return p.parseSearchResults(string(body))
}

// parseSearchResults 解析搜索结果页面
func (p *ClxiongPlugin) parseSearchResults(html string) ([]model.SearchResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	var results []model.SearchResult

	// 查找搜索结果项
	doc.Find(".row.row-cols-2.row-cols-lg-4 .col").Each(func(i int, s *goquery.Selection) {
		if i >= MaxResults {
			return // 限制结果数量
		}

		// 提取详情页链接
		linkEl := s.Find("a[href*='/drama/'], a[href*='/movie/']")
		if linkEl.Length() == 0 {
			return // 跳过无链接的项
		}

		detailPath, exists := linkEl.Attr("href")
		if !exists || detailPath == "" {
			return
		}

		// 构建完整的详情页URL
		detailURL := BaseURL + detailPath

		// 提取标题
		title := strings.TrimSpace(linkEl.Find("h2.h4").Text())
		if title == "" {
			return // 跳过无标题的项
		}

		// 提取评分
		rating := strings.TrimSpace(s.Find(".rank").Text())

		// 提取年份
		year := strings.TrimSpace(s.Find(".small").Last().Text())

		// 提取海报图片
		poster := ""
		cardImg := s.Find(".card-img")
		if cardImg.Length() > 0 {
			if style, exists := cardImg.Attr("style"); exists {
				poster = p.extractImageFromStyle(style)
			}
		}

		// 构建内容信息
		var contentParts []string
		if rating != "" {
			contentParts = append(contentParts, "评分: "+rating)
		}
		if year != "" {
			contentParts = append(contentParts, "年份: "+year)
		}
		if poster != "" {
			contentParts = append(contentParts, "海报: "+poster)
		}
		// 添加详情页链接到content中，供后续提取磁力链接使用
		contentParts = append(contentParts, "详情页: "+detailURL)

		content := strings.Join(contentParts, " | ")

		// 生成唯一ID
		uniqueID := p.generateUniqueID(detailPath)

		result := model.SearchResult{
			Title:    title,
			Content:  content,
			Channel:  "", // 插件搜索结果必须为空
			Tags:     []string{"磁力链接", "影视"},
			Datetime: time.Now(), // 搜索时间
			Links:    []model.Link{}, // 初始为空，后续异步获取
			UniqueID: uniqueID,
		}

		results = append(results, result)
	})

	if p.debugMode {
		log.Printf("[CLXIONG] 解析到 %d 个搜索结果", len(results))
	}

	return results, nil
}

// extractImageFromStyle 从style属性中提取背景图片URL
func (p *ClxiongPlugin) extractImageFromStyle(style string) string {
	// style格式: "background-image: url('https://i.nacloud.cc/2024/12154.webp');"
	re := regexp.MustCompile(`url\(['"]?([^'"]+)['"]?\)`)
	matches := re.FindStringSubmatch(style)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// fetchDetailLinksAsync 异步获取详情页磁力链接
func (p *ClxiongPlugin) fetchDetailLinksAsync(results []model.SearchResult) {
	if len(results) == 0 {
		return
	}

	if p.debugMode {
		log.Printf("[CLXIONG] 开始异步获取 %d 个详情页的磁力链接", len(results))
	}

	// 使用goroutine异步获取，避免阻塞主搜索流程
	for i := range results {
		go func(index int) {
			detailURL := p.extractDetailURLFromContent(results[index].Content)
			if detailURL != "" {
				magnetLinks := p.fetchDetailPageMagnetLinks(detailURL)
				if len(magnetLinks) > 0 {
					results[index].Links = magnetLinks
					if p.debugMode {
						log.Printf("[CLXIONG] 为结果 %d 获取到 %d 个磁力链接", index+1, len(magnetLinks))
					}
				}
			}
		}(i)
	}
}

// extractDetailURLFromContent 从content中提取详情页URL
func (p *ClxiongPlugin) extractDetailURLFromContent(content string) string {
	// 查找"详情页: URL"模式
	re := regexp.MustCompile(`详情页: (https?://[^\s|]+)`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// fetchDetailPageMagnetLinks 获取详情页的磁力链接
func (p *ClxiongPlugin) fetchDetailPageMagnetLinks(detailURL string) []model.Link {
	if p.debugMode {
		log.Printf("[CLXIONG] 正在获取详情页磁力链接: %s", detailURL)
	}

	client := &http.Client{Timeout: 20 * time.Second}

	req, err := http.NewRequest("GET", detailURL, nil)
	if err != nil {
		if p.debugMode {
			log.Printf("[CLXIONG] 创建详情页请求失败: %v", err)
		}
		return nil
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Referer", BaseURL+"/")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		if p.debugMode {
			log.Printf("[CLXIONG] 详情页请求失败: %v", err)
		}
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if p.debugMode {
			log.Printf("[CLXIONG] 详情页HTTP状态错误: %d", resp.StatusCode)
		}
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if p.debugMode {
			log.Printf("[CLXIONG] 读取详情页响应失败: %v", err)
		}
		return nil
	}

	return p.parseMagnetLinksFromDetail(string(body))
}

// parseMagnetLinksFromDetail 从详情页HTML中解析磁力链接
func (p *ClxiongPlugin) parseMagnetLinksFromDetail(html string) []model.Link {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		if p.debugMode {
			log.Printf("[CLXIONG] 解析详情页HTML失败: %v", err)
		}
		return nil
	}

	var links []model.Link

	// 查找磁力链接
	doc.Find(".mv_down a[href^='magnet:']").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists && href != "" {
			// 获取文件名（链接文本）
			fileName := strings.TrimSpace(s.Text())
			
			link := model.Link{
				URL:  href,
				Type: "magnet",
			}

			// 如果文件名包含大小信息，可以存储在Password字段中作为备注
			if fileName != "" && strings.Contains(fileName, "[") {
				link.Password = fileName // 临时存储文件名和大小信息
			}

			links = append(links, link)

			if p.debugMode {
				log.Printf("[CLXIONG] 找到磁力链接: %s %s", fileName, href)
			}
		}
	})

	if p.debugMode {
		log.Printf("[CLXIONG] 详情页共找到 %d 个磁力链接", len(links))
	}

	return links
}

// generateUniqueID 生成唯一ID
func (p *ClxiongPlugin) generateUniqueID(detailPath string) string {
	// 从路径中提取ID，如 "/drama/4466.html" -> "4466"
	re := regexp.MustCompile(`/(?:drama|movie)/(\d+)\.html`)
	matches := re.FindStringSubmatch(detailPath)
	if len(matches) > 1 {
		return fmt.Sprintf("clxiong-%s", matches[1])
	}
	
	// 备用方案：使用完整路径生成哈希
	hash := 0
	for _, char := range detailPath {
		hash = hash*31 + int(char)
	}
	if hash < 0 {
		hash = -hash
	}
	return fmt.Sprintf("clxiong-%d", hash)
}