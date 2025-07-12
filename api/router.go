package api

import (
	"github.com/gin-gonic/gin"
	"pansou/service"
	"pansou/util"
)

// SetupRouter 设置路由
func SetupRouter(searchService *service.SearchService) *gin.Engine {
	// 设置搜索服务
	SetSearchService(searchService)
	
	// 设置为生产模式
	gin.SetMode(gin.ReleaseMode)
	
	// 创建默认路由
	r := gin.Default()
	
	// 添加中间件
	r.Use(CORSMiddleware())
	r.Use(LoggerMiddleware())
	r.Use(util.GzipMiddleware()) // 添加压缩中间件
	
	// 定义API路由组
	api := r.Group("/api")
	{
		// 搜索接口 - 支持POST和GET两种方式
		api.POST("/search", SearchHandler)
		api.GET("/search", SearchHandler) // 添加GET方式支持
		
		// 健康检查接口
		api.GET("/health", func(c *gin.Context) {
			pluginCount := 0
			if searchService != nil && searchService.GetPluginManager() != nil {
				pluginCount = len(searchService.GetPluginManager().GetPlugins())
			}
			
			c.JSON(200, gin.H{
				"status": "ok",
				"plugins_enabled": true,
				"plugin_count": pluginCount,
			})
		})
	}
	
	return r
} 