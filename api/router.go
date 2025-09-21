package api

import (
	"github.com/gin-gonic/gin"
	"pansou/config"
	"pansou/service"
	"pansou/util"
)

// SetupRouter 设置路由
func SetupRouter(searchService *service.SearchService) *gin.Engine {
	// 设置搜索服务
	SetSearchService(searchService)
	
	// 创建认证服务
	authService := service.NewAuthService()
	SetAuthService(authService)
	
	// 创建认证处理器
	authHandler := NewAuthHandler(authService)
	
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
		// 认证相关路由（不需要认证）
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)           // 用户注册
			auth.POST("/login", authHandler.Login)                 // 用户登录
			auth.POST("/logout", AuthMiddleware(), authHandler.Logout) // 用户登出
			auth.POST("/refresh", AuthMiddleware(), authHandler.RefreshToken) // 刷新令牌
		}
		
		// 用户相关路由（需要认证）
		user := api.Group("/user")
		user.Use(AuthMiddleware())
		{
			user.GET("/profile", authHandler.GetProfile)                    // 获取用户资料
			user.PUT("/profile", authHandler.UpdateProfile)                 // 更新用户资料
			user.POST("/change-password", authHandler.ChangePassword)       // 修改密码
			user.POST("/upgrade-membership", authHandler.UpgradeMembership) // 升级会员
			user.GET("/stats", authHandler.GetUserStats)                    // 获取用户统计
		}
		
		// 搜索接口 - 支持POST和GET两种方式（可选认证）
		api.POST("/search", OptionalAuthMiddleware(), SearchHandler)
		api.GET("/search", OptionalAuthMiddleware(), SearchHandler)
		
		// 高级搜索接口（需要会员权限）
		api.POST("/search/advanced", AuthMiddleware(), RequireMember(), SearchHandler)
		api.GET("/search/advanced", AuthMiddleware(), RequireMember(), SearchHandler)
		
		// 搜索历史接口（需要认证）
		api.GET("/search/history", AuthMiddleware(), SearchHistoryHandler)
		api.DELETE("/search/history", AuthMiddleware(), ClearSearchHistoryHandler)
		
		// 健康检查接口
		api.GET("/health", func(c *gin.Context) {
			// 根据配置决定是否返回插件信息
			pluginCount := 0
			pluginNames := []string{}
			pluginsEnabled := config.AppConfig.AsyncPluginEnabled
			
			if pluginsEnabled && searchService != nil && searchService.GetPluginManager() != nil {
				plugins := searchService.GetPluginManager().GetPlugins()
				pluginCount = len(plugins)
				for _, p := range plugins {
					pluginNames = append(pluginNames, p.Name())
				}
			}
			
			// 获取频道信息
			channels := config.AppConfig.DefaultChannels
			channelsCount := len(channels)
			
			response := gin.H{
				"status": "ok",
				"plugins_enabled": pluginsEnabled,
				"channels": channels,
				"channels_count": channelsCount,
			}
			
			// 只有当插件启用时才返回插件相关信息
			if pluginsEnabled {
				response["plugin_count"] = pluginCount
				response["plugins"] = pluginNames
			}
			
			c.JSON(200, response)
		})
	}
	
	return r
} 