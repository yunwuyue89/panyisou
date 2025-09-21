package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"pansou/model"
	"pansou/service"
	jsonutil "pansou/util/json"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Register 用户注册
func (h *AuthHandler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "请求参数错误: "+err.Error()))
		return
	}

	// 注册用户
	user, err := h.authService.Register(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, err.Error()))
		return
	}

	// 返回用户信息（不包含密码）
	response := model.NewSuccessResponse(gin.H{
		"user": user,
		"message": "注册成功",
	})
	
	jsonData, _ := jsonutil.Marshal(response)
	c.Data(http.StatusOK, "application/json", jsonData)
}

// Login 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "请求参数错误: "+err.Error()))
		return
	}

	// 获取客户端信息
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	// 登录
	response, err := h.authService.Login(&req, ipAddress, userAgent)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.NewErrorResponse(401, err.Error()))
		return
	}

	// 返回登录信息
	jsonData, _ := jsonutil.Marshal(model.NewSuccessResponse(response))
	c.Data(http.StatusOK, "application/json", jsonData)
}

// Logout 用户登出
func (h *AuthHandler) Logout(c *gin.Context) {
	// 获取令牌
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "缺少认证令牌"))
		return
	}

	token := authHeader[7:] // 移除 "Bearer " 前缀

	// 登出
	if err := h.authService.Logout(token); err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "登出失败: "+err.Error()))
		return
	}

	// 返回成功信息
	response := model.NewSuccessResponse(gin.H{
		"message": "登出成功",
	})
	
	jsonData, _ := jsonutil.Marshal(response)
	c.Data(http.StatusOK, "application/json", jsonData)
}

// GetProfile 获取用户资料
func (h *AuthHandler) GetProfile(c *gin.Context) {
	user := GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, model.NewErrorResponse(401, "需要认证"))
		return
	}

	// 返回用户资料
	response := model.NewSuccessResponse(gin.H{
		"user": user,
	})
	
	jsonData, _ := jsonutil.Marshal(response)
	c.Data(http.StatusOK, "application/json", jsonData)
}

// UpdateProfile 更新用户资料
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	user := GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, model.NewErrorResponse(401, "需要认证"))
		return
	}

	var req model.UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "请求参数错误: "+err.Error()))
		return
	}

	// 更新用户资料
	updatedUser, err := h.authService.UpdateUser(user.ID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, err.Error()))
		return
	}

	// 返回更新后的用户信息
	response := model.NewSuccessResponse(gin.H{
		"user": updatedUser,
		"message": "资料更新成功",
	})
	
	jsonData, _ := jsonutil.Marshal(response)
	c.Data(http.StatusOK, "application/json", jsonData)
}

// ChangePassword 修改密码
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	user := GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, model.NewErrorResponse(401, "需要认证"))
		return
	}

	var req model.PasswordChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "请求参数错误: "+err.Error()))
		return
	}

	// 修改密码
	if err := h.authService.ChangePassword(user.ID, &req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, err.Error()))
		return
	}

	// 返回成功信息
	response := model.NewSuccessResponse(gin.H{
		"message": "密码修改成功",
	})
	
	jsonData, _ := jsonutil.Marshal(response)
	c.Data(http.StatusOK, "application/json", jsonData)
}

// UpgradeMembership 升级会员
func (h *AuthHandler) UpgradeMembership(c *gin.Context) {
	user := GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, model.NewErrorResponse(401, "需要认证"))
		return
	}

	var req model.MembershipUpgradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "请求参数错误: "+err.Error()))
		return
	}

	// 升级会员
	membership, err := h.authService.UpgradeMembership(user.ID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, err.Error()))
		return
	}

	// 返回会员信息
	response := model.NewSuccessResponse(gin.H{
		"membership": membership,
		"message": "会员升级成功",
	})
	
	jsonData, _ := jsonutil.Marshal(response)
	c.Data(http.StatusOK, "application/json", jsonData)
}

// GetUserStats 获取用户统计信息
func (h *AuthHandler) GetUserStats(c *gin.Context) {
	user := GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, model.NewErrorResponse(401, "需要认证"))
		return
	}

	// 这里可以实现用户统计信息的获取
	// 暂时返回模拟数据
	stats := model.UserStats{
		TotalSearches:    0,
		TodaySearches:    0,
		LastSearchAt:     user.LastLoginAt,
		FavoriteChannels: user.Profile.Preferences.DefaultChannels,
		FavoritePlugins:  user.Profile.Preferences.DefaultPlugins,
	}

	// 返回统计信息
	response := model.NewSuccessResponse(gin.H{
		"stats": stats,
	})
	
	jsonData, _ := jsonutil.Marshal(response)
	c.Data(http.StatusOK, "application/json", jsonData)
}

// RefreshToken 刷新令牌
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	user := GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, model.NewErrorResponse(401, "需要认证"))
		return
	}

	// 生成新的JWT令牌
	token, expiresAt, err := h.authService.(*service.AuthService).generateJWT(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "生成令牌失败: "+err.Error()))
		return
	}

	// 返回新令牌
	response := model.NewSuccessResponse(gin.H{
		"token": token,
		"expires_at": expiresAt,
		"message": "令牌刷新成功",
	})
	
	jsonData, _ := jsonutil.Marshal(response)
	c.Data(http.StatusOK, "application/json", jsonData)
}
