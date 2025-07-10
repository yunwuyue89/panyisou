package api

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware 跨域中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	}
}

// LoggerMiddleware 日志中间件
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		startTime := time.Now()
		
		// 处理请求
		c.Next()
		
		// 结束时间
		endTime := time.Now()
		
		// 执行时间
		latencyTime := endTime.Sub(startTime)
		
		// 请求方式
		reqMethod := c.Request.Method
		
		// 请求路由
		reqURI := c.Request.RequestURI
		
		// 状态码
		statusCode := c.Writer.Status()
		
		// 请求IP
		clientIP := c.ClientIP()
		
		// 日志格式
		gin.DefaultWriter.Write([]byte(
			fmt.Sprintf("| %s | %s | %s | %d | %s\n", 
				clientIP, reqMethod, reqURI, statusCode, latencyTime.String())))
	}
} 