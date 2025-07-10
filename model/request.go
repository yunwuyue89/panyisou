package model

// SearchRequest 搜索请求参数
type SearchRequest struct {
	Keyword      string   `json:"keyword" binding:"required"`
	Channels     []string `json:"channels"`
	Concurrency  int      `json:"concurrency"`
	ForceRefresh bool     `json:"force_refresh"` // 强制刷新，不使用缓存
	ResultType   string   `json:"result_type"`   // 结果类型：all, results, merged_by_type
} 