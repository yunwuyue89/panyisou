package cache

import (
	"crypto/md5"
	"encoding/hex"
	"sort"
)

// GenerateCacheKey 根据查询和过滤器生成缓存键
func GenerateCacheKey(query string, filters map[string]string) string {
	// 如果只需要基于关键词的缓存，不考虑过滤器
	if filters == nil || len(filters) == 0 {
		// 直接使用查询字符串生成键，添加前缀以区分
		keyStr := "keyword_only:" + query
		hash := md5.Sum([]byte(keyStr))
		return hex.EncodeToString(hash[:])
	}
	
	// 创建包含查询和所有过滤器的字符串
	keyStr := query

	// 按字母顺序排序过滤器键，确保相同的过滤器集合总是产生相同的键
	var keys []string
	for k := range filters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 添加过滤器到键字符串
	for _, k := range keys {
		keyStr += "|" + k + "=" + filters[k]
	}

	// 计算MD5哈希
	hash := md5.Sum([]byte(keyStr))
	return hex.EncodeToString(hash[:])
} 