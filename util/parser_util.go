package util

import (
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"pansou/model"
)

// isSupportedLink 检查链接是否为支持的网盘链接
func isSupportedLink(url string) bool {
	lowerURL := strings.ToLower(url)
	
	// 检查是否为百度网盘链接
	if BaiduPanPattern.MatchString(lowerURL) {
		return true
	}
	
	// 检查是否为天翼云盘链接
	if TianyiPanPattern.MatchString(lowerURL) {
		return true
	}
	
	// 检查是否为UC网盘链接
	if UCPanPattern.MatchString(lowerURL) {
		return true
	}
	
	// 检查是否为123网盘链接
	if Pan123Pattern.MatchString(lowerURL) {
		return true
	}
	
	// 检查是否为夸克网盘链接
	if QuarkPanPattern.MatchString(lowerURL) {
		return true
	}
	
	// 检查是否为迅雷网盘链接
	if XunleiPanPattern.MatchString(lowerURL) {
		return true
	}
	
	// 检查是否为115网盘链接
	if Pan115Pattern.MatchString(lowerURL) {
		return true
	}
	
	// 使用通用模式检查其他网盘链接
	return AllPanLinksPattern.MatchString(lowerURL)
}

// normalizeBaiduPanURL 标准化百度网盘URL，确保链接格式正确并且包含密码参数
func normalizeBaiduPanURL(url string, password string) string {
	// 清理URL，确保获取正确的链接部分
	url = CleanBaiduPanURL(url)
	
	// 如果URL已经包含pwd参数，不需要再添加
	if strings.Contains(url, "?pwd=") {
		return url
	}
	
	// 如果有提取到密码，且URL不包含pwd参数，则添加
	if password != "" {
		// 确保密码是4位
		if len(password) > 4 {
			password = password[:4]
		}
		return url + "?pwd=" + password
	}
	
	return url
}

// normalizeTianyiPanURL 标准化天翼云盘URL，确保链接格式正确
func normalizeTianyiPanURL(url string, password string) string {
	// 清理URL，确保获取正确的链接部分
	url = CleanTianyiPanURL(url)
	
	// 天翼云盘链接通常不在URL中包含密码参数，所以这里不做处理
	// 但是我们确保返回的是干净的链接
	return url
}

// normalizeUCPanURL 标准化UC网盘URL，确保链接格式正确
func normalizeUCPanURL(url string, password string) string {
	// 清理URL，确保获取正确的链接部分
	url = CleanUCPanURL(url)
	
	// UC网盘链接通常使用?public=1参数表示公开分享
	// 确保链接格式正确，但不添加密码参数
	return url
}

// normalize123PanURL 标准化123网盘URL，确保链接格式正确
func normalize123PanURL(url string, password string) string {
	// 清理URL，确保获取正确的链接部分
	url = Clean123PanURL(url)
	
	// 123网盘链接通常不在URL中包含密码参数
	// 但是我们确保返回的是干净的链接
	return url
}

// normalize115PanURL 标准化115网盘URL，确保链接格式正确
func normalize115PanURL(url string, password string) string {
	// 清理URL，确保获取正确的链接部分，只保留到password=后面4位密码
	url = Clean115PanURL(url)
	
	// 115网盘链接已经在Clean115PanURL中处理了密码部分
	// 这里不需要额外添加密码参数
	return url
}

// ParseSearchResults 解析搜索结果页面
func ParseSearchResults(html string, channel string) ([]model.SearchResult, string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, "", err
	}

	var results []model.SearchResult
	var nextPageParam string

	// 查找分页链接 - 使用next而不是prev来获取下一页
	// doc.Find("link[rel='next']").Each(func(i int, s *goquery.Selection) {
	// 	href, exists := s.Attr("href")
	// 	if exists {
	// 		// 从href中提取before参数
	// 		parts := strings.Split(href, "before=")
	// 		if len(parts) > 1 {
	// 			nextPageParam = strings.Split(parts[1], "&")[0]
	// 		}
	// 	}
	// })

	// 查找消息块
	doc.Find(".tgme_widget_message_wrap").Each(func(i int, s *goquery.Selection) {
		messageDiv := s.Find(".tgme_widget_message")
		
		// 提取消息ID
		dataPost, exists := messageDiv.Attr("data-post")
		if !exists {
			return
		}
		
		parts := strings.Split(dataPost, "/")
		if len(parts) != 2 {
			return
		}
		
		messageID := parts[1]
		
		// 生成全局唯一ID
		uniqueID := channel + "_" + messageID
		
		// 提取时间
		timeStr, exists := messageDiv.Find(".tgme_widget_message_date time").Attr("datetime")
		if !exists {
			return
		}
		
		datetime, err := time.Parse(time.RFC3339, timeStr)
		if err != nil {
			return
		}
		
		// 获取消息文本元素
		messageTextElem := messageDiv.Find(".tgme_widget_message_text")
		
		// 获取消息文本的HTML内容
		messageHTML, _ := messageTextElem.Html()
		
		// 获取消息的纯文本内容
		messageText := messageTextElem.Text()
		
		// 提取标题
		title := extractTitle(messageHTML, messageText)
		
		// 提取网盘链接 - 使用更精确的方法
		var links []model.Link
		var foundLinks = make(map[string]bool) // 用于去重
		var baiduLinkPasswords = make(map[string]string) // 存储百度链接和对应的密码
		var tianyiLinkPasswords = make(map[string]string) // 存储天翼链接和对应的密码
		var ucLinkPasswords = make(map[string]string) // 存储UC链接和对应的密码
		var pan123LinkPasswords = make(map[string]string) // 存储123网盘链接和对应的密码
		var pan115LinkPasswords = make(map[string]string) // 存储115网盘链接和对应的密码
		var aliyunLinkPasswords = make(map[string]string) // 存储阿里云盘链接和对应的密码
		
		// 1. 从文本内容中提取所有网盘链接和密码
		extractedLinks := ExtractNetDiskLinks(messageText)
		
		// 2. 从a标签中提取链接
		messageTextElem.Find("a").Each(func(i int, a *goquery.Selection) {
			href, exists := a.Attr("href")
			if !exists {
				return
			}
			
			// 使用更精确的方式匹配网盘链接
			if isSupportedLink(href) {
				linkType := GetLinkType(href)
				password := ExtractPassword(messageText, href)
				
				// 如果是百度网盘链接，记录链接和密码的对应关系
				if linkType == "baidu" {
					// 提取链接的基本部分（不含密码参数）
					baseURL := href
					if strings.Contains(href, "?pwd=") {
						baseURL = href[:strings.Index(href, "?pwd=")]
					}
					
					// 记录密码
					if password != "" {
						baiduLinkPasswords[baseURL] = password
					}
				} else if linkType == "tianyi" {
					// 如果是天翼云盘链接，记录链接和密码的对应关系
					baseURL := CleanTianyiPanURL(href)
					
					// 记录密码
					if password != "" {
						tianyiLinkPasswords[baseURL] = password
					} else {
						// 即使没有密码，也添加到映射中，以便后续处理
						if _, exists := tianyiLinkPasswords[baseURL]; !exists {
							tianyiLinkPasswords[baseURL] = ""
						}
					}
				} else if linkType == "uc" {
					// 如果是UC网盘链接，记录链接和密码的对应关系
					baseURL := CleanUCPanURL(href)
					
					// 记录密码
					if password != "" {
						ucLinkPasswords[baseURL] = password
					} else {
						// 即使没有密码，也添加到映射中，以便后续处理
						if _, exists := ucLinkPasswords[baseURL]; !exists {
							ucLinkPasswords[baseURL] = ""
						}
					}
				} else if linkType == "123" {
					// 如果是123网盘链接，记录链接和密码的对应关系
					baseURL := Clean123PanURL(href)
					
					// 记录密码
					if password != "" {
						pan123LinkPasswords[baseURL] = password
					} else {
						// 即使没有密码，也添加到映射中，以便后续处理
						if _, exists := pan123LinkPasswords[baseURL]; !exists {
							pan123LinkPasswords[baseURL] = ""
						}
					}
				} else if linkType == "115" {
					// 如果是115网盘链接，记录链接和密码的对应关系
					baseURL := Clean115PanURL(href)
					
					// 记录密码
					if password != "" {
						pan115LinkPasswords[baseURL] = password
					} else {
						// 即使没有密码，也添加到映射中，以便后续处理
						if _, exists := pan115LinkPasswords[baseURL]; !exists {
							pan115LinkPasswords[baseURL] = ""
						}
					}
				} else if linkType == "aliyun" {
					// 如果是阿里云盘链接，记录链接和密码的对应关系
					baseURL := CleanAliyunPanURL(href)
					
					// 记录密码
					if password != "" {
						aliyunLinkPasswords[baseURL] = password
					} else {
						// 即使没有密码，也添加到映射中，以便后续处理
						if _, exists := aliyunLinkPasswords[baseURL]; !exists {
							aliyunLinkPasswords[baseURL] = ""
						}
					}
				} else {
					// 非特殊处理的网盘链接直接添加
					if !foundLinks[href] {
						foundLinks[href] = true
						links = append(links, model.Link{
							Type:     linkType,
							URL:      href,
							Password: password,
						})
					}
				}
			}
		})
		
		// 3. 处理从文本中提取的链接
		for _, linkURL := range extractedLinks {
			linkType := GetLinkType(linkURL)
			password := ExtractPassword(messageText, linkURL)
			
			// 如果是百度网盘链接，记录链接和密码的对应关系
			if linkType == "baidu" {
				// 提取链接的基本部分（不含密码参数）
				baseURL := linkURL
				if strings.Contains(linkURL, "?pwd=") {
					baseURL = linkURL[:strings.Index(linkURL, "?pwd=")]
				}
				
				// 记录密码
				if password != "" {
					baiduLinkPasswords[baseURL] = password
				}
			} else if linkType == "tianyi" {
				// 如果是天翼云盘链接，记录链接和密码的对应关系
				baseURL := CleanTianyiPanURL(linkURL)
				
				// 记录密码
				if password != "" {
					tianyiLinkPasswords[baseURL] = password
				} else {
					// 即使没有密码，也添加到映射中，以便后续处理
					if _, exists := tianyiLinkPasswords[baseURL]; !exists {
						tianyiLinkPasswords[baseURL] = ""
					}
				}
			} else if linkType == "uc" {
				// 如果是UC网盘链接，记录链接和密码的对应关系
				baseURL := CleanUCPanURL(linkURL)
				
				// 记录密码
				if password != "" {
					ucLinkPasswords[baseURL] = password
				} else {
					// 即使没有密码，也添加到映射中，以便后续处理
					if _, exists := ucLinkPasswords[baseURL]; !exists {
						ucLinkPasswords[baseURL] = ""
					}
				}
			} else if linkType == "123" {
				// 如果是123网盘链接，记录链接和密码的对应关系
				baseURL := Clean123PanURL(linkURL)
				
				// 记录密码
				if password != "" {
					pan123LinkPasswords[baseURL] = password
				} else {
					// 即使没有密码，也添加到映射中，以便后续处理
					if _, exists := pan123LinkPasswords[baseURL]; !exists {
						pan123LinkPasswords[baseURL] = ""
					}
				}
			} else if linkType == "115" {
				// 如果是115网盘链接，记录链接和密码的对应关系
				baseURL := Clean115PanURL(linkURL)
				
				// 记录密码
				if password != "" {
					pan115LinkPasswords[baseURL] = password
				} else {
					// 即使没有密码，也添加到映射中，以便后续处理
					if _, exists := pan115LinkPasswords[baseURL]; !exists {
						pan115LinkPasswords[baseURL] = ""
					}
				}
			} else if linkType == "aliyun" {
				// 如果是阿里云盘链接，记录链接和密码的对应关系
				baseURL := CleanAliyunPanURL(linkURL)
				
				// 记录密码
				if password != "" {
					aliyunLinkPasswords[baseURL] = password
				} else {
					// 即使没有密码，也添加到映射中，以便后续处理
					if _, exists := aliyunLinkPasswords[baseURL]; !exists {
						aliyunLinkPasswords[baseURL] = ""
					}
				}
			} else {
				// 非特殊处理的网盘链接直接添加
				if !foundLinks[linkURL] {
					foundLinks[linkURL] = true
					links = append(links, model.Link{
						Type:     linkType,
						URL:      linkURL,
						Password: password,
					})
				}
			}
		}
		
		// 4. 处理百度网盘链接，确保每个链接只有一个版本（带密码的完整版本）
		for baseURL, password := range baiduLinkPasswords {
			normalizedURL := normalizeBaiduPanURL(baseURL, password)
			
			// 确保链接不重复
			if !foundLinks[normalizedURL] {
				foundLinks[normalizedURL] = true
				links = append(links, model.Link{
					Type:     "baidu",
					URL:      normalizedURL,
					Password: password,
				})
			}
		}
		
		// 5. 处理天翼云盘链接，确保每个链接只有一个版本
		for baseURL, password := range tianyiLinkPasswords {
			normalizedURL := normalizeTianyiPanURL(baseURL, password)
			
			// 确保链接不重复
			if !foundLinks[normalizedURL] {
				foundLinks[normalizedURL] = true
				links = append(links, model.Link{
					Type:     "tianyi",
					URL:      normalizedURL,
					Password: password,
				})
			}
		}
		
		// 6. 处理UC网盘链接，确保每个链接只有一个版本
		for baseURL, password := range ucLinkPasswords {
			normalizedURL := normalizeUCPanURL(baseURL, password)
			
			// 确保链接不重复
			if !foundLinks[normalizedURL] {
				foundLinks[normalizedURL] = true
				links = append(links, model.Link{
					Type:     "uc",
					URL:      normalizedURL,
					Password: password,
				})
			}
		}
		
		// 7. 处理123网盘链接，确保每个链接只有一个版本
		for baseURL, password := range pan123LinkPasswords {
			normalizedURL := normalize123PanURL(baseURL, password)
			
			// 确保链接不重复
			if !foundLinks[normalizedURL] {
				foundLinks[normalizedURL] = true
				links = append(links, model.Link{
					Type:     "123",
					URL:      normalizedURL,
					Password: password,
				})
			}
		}
		
		// 8. 处理115网盘链接，确保每个链接只有一个版本
		for baseURL, password := range pan115LinkPasswords {
			normalizedURL := normalize115PanURL(baseURL, password)
			
			// 确保链接不重复
			if !foundLinks[normalizedURL] {
				foundLinks[normalizedURL] = true
				links = append(links, model.Link{
					Type:     "115",
					URL:      normalizedURL,
					Password: password,
				})
			}
		}
		
		// 9. 处理阿里云盘链接，确保每个链接只有一个版本
		for baseURL, password := range aliyunLinkPasswords {
			normalizedURL := CleanAliyunPanURL(baseURL) // 阿里云盘URL通常不包含密码参数
			
			// 确保链接不重复
			if !foundLinks[normalizedURL] {
				foundLinks[normalizedURL] = true
				links = append(links, model.Link{
					Type:     "aliyun",
					URL:      normalizedURL,
					Password: password,
				})
			}
		}
		
		// 提取标签
		var tags []string
		messageTextElem.Find("a[href^='?q=%23']").Each(func(i int, a *goquery.Selection) {
			tag := a.Text()
			if strings.HasPrefix(tag, "#") {
				tags = append(tags, tag[1:])
			}
		})
		
		// 只有包含链接的消息才添加到结果中
		if len(links) > 0 {
			results = append(results, model.SearchResult{
				MessageID: messageID,
				UniqueID:  uniqueID,
				Channel:   channel,
				Datetime:  datetime,
				Title:     title,
				Content:   messageText,
				Links:     links,
				Tags:      tags,
			})
		}
	})

	return results, nextPageParam, nil
}

// extractTitle 从消息HTML和文本内容中提取标题
func extractTitle(htmlContent string, textContent string) string {
	// 从HTML内容中提取标题
	if brIndex := strings.Index(htmlContent, "<br"); brIndex > 0 {
		// 提取<br>前的HTML内容
		firstLineHTML := htmlContent[:brIndex]
		
		// 创建一个文档来解析这个HTML片段
		doc, err := goquery.NewDocumentFromReader(strings.NewReader("<div>" + firstLineHTML + "</div>"))
		if err == nil {
			// 获取解析后的文本
			firstLine := strings.TrimSpace(doc.Text())
			
			// 如果第一行以"名称："开头，则提取冒号后面的内容作为标题
			if strings.HasPrefix(firstLine, "名称：") {
				return strings.TrimSpace(firstLine[len("名称："):])
			}
			
			return firstLine
		}
	}
	
	// 如果HTML解析失败，则使用纯文本内容
	lines := strings.Split(textContent, "\n")
	if len(lines) == 0 {
		return ""
	}
	
	// 第一行通常是标题
	firstLine := strings.TrimSpace(lines[0])
	
	// 如果第一行以"名称："开头，则提取冒号后面的内容作为标题
	if strings.HasPrefix(firstLine, "名称：") {
		return strings.TrimSpace(firstLine[len("名称："):])
	}
	
	// 否则直接使用第一行作为标题
	return firstLine
} 