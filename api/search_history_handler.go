package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"pansou/model"
	jsonutil "pansou/util/json"
)

// SearchHistoryEntry 搜索历史条目
type SearchHistoryEntry struct {
	ID        string    `json:"id"`
	Keyword   string    `json:"keyword"`
	Channels  []string  `json:"channels"`
	Plugins   []string  `json:"plugins"`
	CloudTypes []string `json:"cloud_types"`
	ResultCount int     `json:"result_count"`
	SearchedAt time.Time `json:"searched_at"`
}

// SearchHistoryHandler 获取搜索历史
func SearchHistoryHandler(c *gin.Context) {
	user := GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, model.NewErrorResponse(401, "需要认证"))
		return
	}

	// 这里可以实现搜索历史的存储和查询
	// 暂时返回模拟数据
	history := []SearchHistoryEntry{
		{
			ID:          "1",
			Keyword:     "测试关键词",
			Channels:    []string{"tgsearchers3"},
			Plugins:     []string{},
			CloudTypes:  []string{},
			ResultCount: 10,
			SearchedAt:  time.Now().Add(-1 * time.Hour),
		},
	}

	// 返回搜索历史
	response := model.NewSuccessResponse(gin.H{
		"history": history,
		"total":   len(history),
	})
	
	jsonData, _ := jsonutil.Marshal(response)
	c.Data(http.StatusOK, "application/json", jsonData)
}

// ClearSearchHistoryHandler 清空搜索历史
func ClearSearchHistoryHandler(c *gin.Context) {
	user := GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, model.NewErrorResponse(401, "需要认证"))
		return
	}

	// 这里可以实现清空搜索历史的逻辑
	// 暂时返回成功信息

	// 返回成功信息
	response := model.NewSuccessResponse(gin.H{
		"message": "搜索历史已清空",
	})
	
	jsonData, _ := jsonutil.Marshal(response)
	c.Data(http.StatusOK, "application/json", jsonData)
}
