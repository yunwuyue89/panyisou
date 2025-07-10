package util

import (
	"regexp"
	"strings"
)

// 通用网盘链接匹配正则表达式
var AllPanLinksPattern = regexp.MustCompile(`(?i)(?:链接[:：]\s*)?(?:(?:magnet:\?xt=urn:btih:[a-zA-Z0-9]+)|(?:ed2k://\|file\|[^|]+\|\d+\|[A-Fa-f0-9]+\|/?)|(?:https?://(?:(?:[\w.-]+\.)?(?:pan\.(?:baidu|quark)\.cn|(?:www\.)?(?:alipan|aliyundrive)\.com|drive\.uc\.cn|cloud\.189\.cn|caiyun\.139\.com|(?:www\.)?123(?:684|685|912|pan|592)\.(?:com|cn)|115\.com|115cdn\.com|anxia\.com|pan\.xunlei\.com|mypikpak\.com))(?:/[^\s'"<>()]*)?))`)

// 提取码匹配正则表达式
var PasswordPattern = regexp.MustCompile(`(?i)(?:(?:提取|访问|提取密|密)码|pwd)[：:]\s*([a-zA-Z0-9]{4,})`)
var UrlPasswordPattern = regexp.MustCompile(`(?i)[?&]pwd=([a-zA-Z0-9]{4,})`)

// GetLinkType 获取链接类型
func GetLinkType(url string) string {
	url = strings.ToLower(url)
	
	// 处理可能带有"链接："前缀的情况
	if strings.Contains(url, "链接：") || strings.Contains(url, "链接:") {
		url = strings.Split(url, "链接")[1]
		if strings.HasPrefix(url, "：") || strings.HasPrefix(url, ":") {
			url = url[1:]
		}
		url = strings.TrimSpace(url)
	}
	
	// 根据关键词判断ed2k链接
	if strings.Contains(url, "ed2k:") {
		return "ed2k"
	}
	
	if strings.HasPrefix(url, "magnet:") {
		return "magnet"
	}
	
	if strings.Contains(url, "pan.baidu.com") {
		return "baidu"
	}
	if strings.Contains(url, "pan.quark.cn") {
		return "quark"
	}
	if strings.Contains(url, "alipan.com") || strings.Contains(url, "aliyundrive.com") {
		return "aliyun"
	}
	if strings.Contains(url, "cloud.189.cn") {
		return "tianyi"
	}
	if strings.Contains(url, "drive.uc.cn") {
		return "uc"
	}
	if strings.Contains(url, "caiyun.139.com") {
		return "mobile"
	}
	if strings.Contains(url, "115.com") || strings.Contains(url, "115cdn.com") || strings.Contains(url, "anxia.com") {
		return "115"
	}
	if strings.Contains(url, "mypikpak.com") {
		return "pikpak"
	}
	if strings.Contains(url, "pan.xunlei.com") {
		return "xunlei"
	}
	
	// 123网盘有多个域名
	if strings.Contains(url, "123684.com") || strings.Contains(url, "123685.com") || 
	   strings.Contains(url, "123912.com") || strings.Contains(url, "123pan.com") || 
	   strings.Contains(url, "123pan.cn") || strings.Contains(url, "123592.com") {
		return "123"
	}
	
	return "others"
}

// ExtractPassword 提取链接密码
func ExtractPassword(content, url string) string {
	// 先从URL中提取密码
	matches := UrlPasswordPattern.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}
	
	// 再从内容中提取密码
	matches = PasswordPattern.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	
	return ""
} 