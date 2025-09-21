package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"pansou/model"
	"pansou/service"
)

var authService *service.AuthService

// SetAuthService 设置认证服务
func SetAuthService(service *service.AuthService) {
	authService = service
}

// AuthMiddleware 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取Authorization头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, model.NewErrorResponse(401, "缺少认证令牌"))
			c.Abort()
			return
		}

		// 检查Bearer格式
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, model.NewErrorResponse(401, "无效的认证格式"))
			c.Abort()
			return
		}

		// 提取令牌
		token := strings.TrimPrefix(authHeader, "Bearer ")

		// 验证令牌
		user, err := authService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, model.NewErrorResponse(401, "无效的认证令牌: "+err.Error()))
			c.Abort()
			return
		}

		// 将用户信息存储到上下文中
		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Set("user_type", user.UserType)
		c.Set("permissions", user.GetUserPermissions())

		c.Next()
	}
}

// OptionalAuthMiddleware 可选认证中间件（不强制要求认证）
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取Authorization头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		// 检查Bearer格式
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.Next()
			return
		}

		// 提取令牌
		token := strings.TrimPrefix(authHeader, "Bearer ")

		// 验证令牌
		user, err := authService.ValidateToken(token)
		if err != nil {
			c.Next()
			return
		}

		// 将用户信息存储到上下文中
		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Set("user_type", user.UserType)
		c.Set("permissions", user.GetUserPermissions())

		c.Next()
	}
}

// RequirePermission 权限检查中间件
func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户权限
		permissions, exists := c.Get("permissions")
		if !exists {
			c.JSON(http.StatusForbidden, model.NewErrorResponse(403, "需要认证"))
			c.Abort()
			return
		}

		// 检查是否有指定权限
		userPermissions := permissions.([]string)
		hasPermission := false
		for _, p := range userPermissions {
			if p == permission {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, model.NewErrorResponse(403, "权限不足"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireUserType 用户类型检查中间件
func RequireUserType(userType model.UserType) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户类型
		userTypeValue, exists := c.Get("user_type")
		if !exists {
			c.JSON(http.StatusForbidden, model.NewErrorResponse(403, "需要认证"))
			c.Abort()
			return
		}

		// 检查用户类型
		if userTypeValue.(model.UserType) != userType {
			c.JSON(http.StatusForbidden, model.NewErrorResponse(403, "用户类型不匹配"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireMember 会员检查中间件
func RequireMember() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusForbidden, model.NewErrorResponse(403, "需要认证"))
			c.Abort()
			return
		}

		// 检查是否为会员
		if !user.(*model.User).IsMember() {
			c.JSON(http.StatusForbidden, model.NewErrorResponse(403, "需要会员权限"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetCurrentUser 获取当前用户
func GetCurrentUser(c *gin.Context) *model.User {
	user, exists := c.Get("user")
	if !exists {
		return nil
	}
	return user.(*model.User)
}

// GetCurrentUserID 获取当前用户ID
func GetCurrentUserID(c *gin.Context) string {
	userID, exists := c.Get("user_id")
	if !exists {
		return ""
	}
	return userID.(string)
}

// GetCurrentUserType 获取当前用户类型
func GetCurrentUserType(c *gin.Context) model.UserType {
	userType, exists := c.Get("user_type")
	if !exists {
		return model.UserTypeNormal
	}
	return userType.(model.UserType)
}

// GetCurrentPermissions 获取当前用户权限
func GetCurrentPermissions(c *gin.Context) []string {
	permissions, exists := c.Get("permissions")
	if !exists {
		return []string{}
	}
	return permissions.([]string)
}

// HasPermission 检查是否有指定权限
func HasPermission(c *gin.Context, permission string) bool {
	permissions := GetCurrentPermissions(c)
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}
