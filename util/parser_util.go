package util

import (
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"pansou/model"
)

// ParseSearchResults 解析搜索结果页面
func ParseSearchResults(html string, channel string) ([]model.SearchResult, string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, "", err
	}

	var results []model.SearchResult
	var nextPageParam string

	// 查找分页链接 - 使用next而不是prev来获取下一页
	doc.Find("link[rel='next']").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			// 从href中提取before参数
			parts := strings.Split(href, "before=")
			if len(parts) > 1 {
				nextPageParam = strings.Split(parts[1], "&")[0]
			}
		}
	})

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
		
		// 提取网盘链接
		var links []model.Link
		messageTextElem.Find("a").Each(func(i int, a *goquery.Selection) {
			href, exists := a.Attr("href")
			if !exists {
				return
			}
			
			// 使用正则表达式匹配网盘链接
			if AllPanLinksPattern.MatchString(href) {
				linkType := GetLinkType(href)
				password := ExtractPassword(messageText, href)
				
				links = append(links, model.Link{
					Type:     linkType,
					URL:      href,
					Password: password,
				})
			}
		})
		
		// 如果没有从a标签中找到链接，尝试从文本中提取
		if len(links) == 0 {
			matches := AllPanLinksPattern.FindAllString(messageText, -1)
			for _, match := range matches {
				linkType := GetLinkType(match)
				password := ExtractPassword(messageText, match)
				
				links = append(links, model.Link{
					Type:     linkType,
					URL:      match,
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