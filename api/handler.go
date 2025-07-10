package api

import (
	"net/http"
	
	"github.com/gin-gonic/gin"
	"pansou/config"
	"pansou/model"
	"pansou/service"
	jsonutil "pansou/util/json"
	"pansou/util"
)

// SearchHandler 搜索处理函数
func SearchHandler(c *gin.Context) {
	var req model.SearchRequest
	var err error

	// 根据请求方法不同处理参数
	if c.Request.Method == http.MethodGet {
		// GET方式：从URL参数获取
		keyword := c.Query("keyword")
		channels := c.QueryArray("channels")
		concurrency := 0
		if c.Query("concurrency") != "" {
			concurrency = util.StringToInt(c.Query("concurrency"))
		}
		forceRefresh := false
		if c.Query("force_refresh") == "true" {
			forceRefresh = true
		}
		resultType := c.Query("result_type")

		req = model.SearchRequest{
			Keyword:      keyword,
			Channels:     channels,
			Concurrency:  concurrency,
			ForceRefresh: forceRefresh,
			ResultType:   resultType,
		}
	} else {
		// POST方式：从请求体获取
		data, err := c.GetRawData()
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "读取请求数据失败: "+err.Error()))
			return
		}

		if err := jsonutil.Unmarshal(data, &req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "无效的请求参数: "+err.Error()))
			return
		}
	}
		
	// 检查并设置默认值
	if len(req.Channels) == 0 {
		req.Channels = config.AppConfig.DefaultChannels
	}
	
	// 如果未指定结果类型，默认返回全部
	if req.ResultType == "" {
		req.ResultType = "all"
	}
	
	// 创建搜索服务并执行搜索
	searchService := service.NewSearchService()
	result, err := searchService.Search(req.Keyword, req.Channels, req.Concurrency, req.ForceRefresh, req.ResultType)
	
	if err != nil {
		response := model.NewErrorResponse(500, "搜索失败: "+err.Error())
		jsonData, _ := jsonutil.Marshal(response)
		c.Data(http.StatusInternalServerError, "application/json", jsonData)
		return
	}

	// 返回结果
	response := model.NewSuccessResponse(result)
	jsonData, _ := jsonutil.Marshal(response)
	c.Data(http.StatusOK, "application/json", jsonData)
} 