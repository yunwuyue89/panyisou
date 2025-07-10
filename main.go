package main

import (
	"fmt"
	"log"

	"pansou/api"
	"pansou/config"
	"pansou/util"
)

func main() {
	// 初始化配置
	config.Init()
	
	// 初始化HTTP客户端
	util.InitHTTPClient()
	
	// 设置路由
	router := api.SetupRouter()
	
	// 获取端口配置
	port := config.AppConfig.Port
	
	// 启动服务器
	fmt.Printf("服务器启动在 http://localhost:%s\n", port)
	
	// 输出代理信息
	if config.AppConfig.UseProxy {
		fmt.Printf("使用SOCKS5代理: %s\n", config.AppConfig.ProxyURL)
	} else {
		fmt.Println("未使用代理")
	}
	
	// 输出缓存信息
	if config.AppConfig.CacheEnabled {
		fmt.Printf("缓存已启用: 路径=%s, 最大大小=%dMB, TTL=%d分钟\n", 
			config.AppConfig.CachePath, 
			config.AppConfig.CacheMaxSizeMB,
			config.AppConfig.CacheTTLMinutes)
	} else {
		fmt.Println("缓存已禁用")
	}
	
	// 输出压缩信息
	if config.AppConfig.EnableCompression {
		fmt.Printf("响应压缩已启用: 最小压缩大小=%d字节\n", 
			config.AppConfig.MinSizeToCompress)
	} else {
		fmt.Println("响应压缩已禁用")
	}
	
	// 输出GC配置信息
	fmt.Printf("GC配置: 触发阈值=%d%%, 内存优化=%v\n", 
		config.AppConfig.GCPercent, 
		config.AppConfig.OptimizeMemory)
	
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}
} 