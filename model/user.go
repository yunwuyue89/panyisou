package model

import (
	"time"
)

// UserType 用户类型
type UserType string

const (
	UserTypeNormal   UserType = "normal"   // 普通用户
	UserTypeMember   UserType = "member"   // 会员
	UserTypeAdmin    UserType = "admin"    // 管理员
)

// User 用户模型
type User struct {
	ID           string    `json:"id"`            // 用户ID
	Username     string    `json:"username"`      // 用户名
	Email        string    `json:"email"`         // 邮箱
	PasswordHash string    `json:"-"`             // 密码哈希（不返回给客户端）
	UserType     UserType  `json:"user_type"`     // 用户类型
	IsActive     bool      `json:"is_active"`     // 是否激活
	CreatedAt    time.Time `json:"created_at"`    // 创建时间
	UpdatedAt    time.Time `json:"updated_at"`    // 更新时间
	LastLoginAt  time.Time `json:"last_login_at"` // 最后登录时间
	LoginCount   int       `json:"login_count"`   // 登录次数
	Profile      UserProfile `json:"profile"`     // 用户资料
}

// UserProfile 用户资料
type UserProfile struct {
	Nickname    string `json:"nickname"`     // 昵称
	Avatar      string `json:"avatar"`       // 头像URL
	Bio         string `json:"bio"`          // 个人简介
	Location    string `json:"location"`     // 所在地
	Website     string `json:"website"`      // 个人网站
	Preferences UserPreferences `json:"preferences"` // 用户偏好设置
}

// UserPreferences 用户偏好设置
type UserPreferences struct {
	DefaultChannels []string `json:"default_channels"` // 默认搜索频道
	DefaultPlugins  []string `json:"default_plugins"`  // 默认搜索插件
	DefaultCloudTypes []string `json:"default_cloud_types"` // 默认网盘类型
	SearchHistory   bool    `json:"search_history"`    // 是否保存搜索历史
	Theme          string  `json:"theme"`             // 主题偏好
	Language       string  `json:"language"`          // 语言偏好
}

// Membership 会员信息
type Membership struct {
	UserID       string    `json:"user_id"`       // 用户ID
	Level        int       `json:"level"`          // 会员等级 (1-5)
	ExpiresAt    time.Time `json:"expires_at"`     // 到期时间
	IsActive     bool      `json:"is_active"`      // 是否激活
	CreatedAt    time.Time `json:"created_at"`     // 创建时间
	UpdatedAt    time.Time `json:"updated_at"`     // 更新时间
	Features     []string  `json:"features"`       // 会员功能列表
	MaxSearches  int       `json:"max_searches"`   // 每日最大搜索次数
	MaxConcurrency int     `json:"max_concurrency"` // 最大并发数
}

// UserSession 用户会话
type UserSession struct {
	ID        string    `json:"id"`         // 会话ID
	UserID    string    `json:"user_id"`    // 用户ID
	Token     string    `json:"token"`      // JWT令牌
	ExpiresAt time.Time `json:"expires_at"` // 过期时间
	CreatedAt time.Time `json:"created_at"` // 创建时间
	IPAddress string    `json:"ip_address"` // IP地址
	UserAgent string    `json:"user_agent"` // 用户代理
	IsActive  bool      `json:"is_active"`  // 是否活跃
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=20"` // 用户名
	Email    string `json:"email" binding:"required,email"`           // 邮箱
	Password string `json:"password" binding:"required,min=6"`        // 密码
	Nickname string `json:"nickname"`                                 // 昵称（可选）
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"` // 用户名或邮箱
	Password string `json:"password" binding:"required"` // 密码
	Remember bool   `json:"remember"`                    // 记住我
}

// LoginResponse 登录响应
type LoginResponse struct {
	User        User        `json:"user"`         // 用户信息
	Token       string      `json:"token"`        // JWT令牌
	ExpiresAt   time.Time   `json:"expires_at"`   // 过期时间
	Membership  *Membership `json:"membership"`   // 会员信息（如果有）
	Permissions []string    `json:"permissions"`  // 权限列表
}

// UserUpdateRequest 用户更新请求
type UserUpdateRequest struct {
	Nickname    string           `json:"nickname"`     // 昵称
	Bio         string           `json:"bio"`          // 个人简介
	Location    string           `json:"location"`     // 所在地
	Website     string           `json:"website"`      // 个人网站
	Preferences *UserPreferences `json:"preferences"`  // 偏好设置
}

// PasswordChangeRequest 密码修改请求
type PasswordChangeRequest struct {
	OldPassword string `json:"old_password" binding:"required"` // 旧密码
	NewPassword string `json:"new_password" binding:"required,min=6"` // 新密码
}

// MembershipUpgradeRequest 会员升级请求
type MembershipUpgradeRequest struct {
	Level     int       `json:"level" binding:"required,min=1,max=5"` // 会员等级
	Months    int       `json:"months" binding:"required,min=1"`      // 购买月数
	PaymentMethod string `json:"payment_method"`                      // 支付方式
}

// UserStats 用户统计
type UserStats struct {
	TotalSearches    int       `json:"total_searches"`     // 总搜索次数
	TodaySearches    int       `json:"today_searches"`     // 今日搜索次数
	LastSearchAt     time.Time `json:"last_search_at"`     // 最后搜索时间
	FavoriteChannels []string  `json:"favorite_channels"`  // 常用频道
	FavoritePlugins  []string  `json:"favorite_plugins"`   // 常用插件
}

// Permission 权限定义
const (
	PermissionSearch        = "search"         // 搜索权限
	PermissionAdvancedSearch = "advanced_search" // 高级搜索权限
	PermissionHistory       = "history"        // 搜索历史权限
	PermissionExport        = "export"         // 导出权限
	PermissionAPI           = "api"            // API访问权限
	PermissionAdmin         = "admin"          // 管理员权限
)

// GetUserPermissions 获取用户权限
func (u *User) GetUserPermissions() []string {
	permissions := []string{PermissionSearch}
	
	switch u.UserType {
	case UserTypeMember:
		permissions = append(permissions, 
			PermissionAdvancedSearch,
			PermissionHistory,
			PermissionExport,
		)
	case UserTypeAdmin:
		permissions = append(permissions,
			PermissionAdvancedSearch,
			PermissionHistory,
			PermissionExport,
			PermissionAPI,
			PermissionAdmin,
		)
	}
	
	return permissions
}

// IsMember 检查是否为会员
func (u *User) IsMember() bool {
	return u.UserType == UserTypeMember || u.UserType == UserTypeAdmin
}

// CanSearch 检查是否可以搜索
func (u *User) CanSearch() bool {
	return u.IsActive
}

// GetMaxSearchesPerDay 获取每日最大搜索次数
func (u *User) GetMaxSearchesPerDay() int {
	switch u.UserType {
	case UserTypeNormal:
		return 10 // 普通用户每日10次
	case UserTypeMember:
		return 100 // 会员每日100次
	case UserTypeAdmin:
		return -1 // 管理员无限制
	default:
		return 10
	}
}

// GetMaxConcurrency 获取最大并发数
func (u *User) GetMaxConcurrency() int {
	switch u.UserType {
	case UserTypeNormal:
		return 3 // 普通用户最大3个并发
	case UserTypeMember:
		return 10 // 会员最大10个并发
	case UserTypeAdmin:
		return 50 // 管理员最大50个并发
	default:
		return 3
	}
}
