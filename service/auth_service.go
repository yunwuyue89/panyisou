package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"pansou/model"
	"pansou/util"
)

// AuthService 认证服务
type AuthService struct {
	usersFile    string
	sessionsFile string
	jwtSecret    []byte
	users        map[string]*model.User
	sessions     map[string]*model.UserSession
}

// JWTClaims JWT声明
type JWTClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	UserType string `json:"user_type"`
	jwt.RegisteredClaims
}

// NewAuthService 创建认证服务
func NewAuthService() *AuthService {
	service := &AuthService{
		usersFile:    "data/users.json",
		sessionsFile: "data/sessions.json",
		jwtSecret:    []byte("pansou-secret-key-2024"), // 在生产环境中应该从环境变量获取
		users:        make(map[string]*model.User),
		sessions:     make(map[string]*model.UserSession),
	}
	
	// 确保数据目录存在
	os.MkdirAll("data", 0755)
	
	// 加载用户和会话数据
	service.loadUsers()
	service.loadSessions()
	
	return service
}

// Register 用户注册
func (s *AuthService) Register(req *model.RegisterRequest) (*model.User, error) {
	// 检查用户名是否已存在
	if s.getUserByUsername(req.Username) != nil {
		return nil, fmt.Errorf("用户名已存在")
	}
	
	// 检查邮箱是否已存在
	if s.getUserByEmail(req.Email) != nil {
		return nil, fmt.Errorf("邮箱已被使用")
	}
	
	// 生成用户ID
	userID := s.generateUserID()
	
	// 加密密码
	passwordHash, err := s.hashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %v", err)
	}
	
	// 创建用户
	user := &model.User{
		ID:           userID,
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
		UserType:     model.UserTypeNormal,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Profile: model.UserProfile{
			Nickname: req.Nickname,
			Preferences: model.UserPreferences{
				DefaultChannels: []string{"tgsearchers3"},
				DefaultPlugins:  []string{},
				DefaultCloudTypes: []string{},
				SearchHistory:   true,
				Theme:          "light",
				Language:       "zh-CN",
			},
		},
	}
	
	// 保存用户
	s.users[userID] = user
	if err := s.saveUsers(); err != nil {
		return nil, fmt.Errorf("保存用户失败: %v", err)
	}
	
	return user, nil
}

// Login 用户登录
func (s *AuthService) Login(req *model.LoginRequest, ipAddress, userAgent string) (*model.LoginResponse, error) {
	// 查找用户
	user := s.getUserByUsername(req.Username)
	if user == nil {
		user = s.getUserByEmail(req.Username)
	}
	
	if user == nil {
		return nil, fmt.Errorf("用户名或密码错误")
	}
	
	// 验证密码
	if !s.verifyPassword(req.Password, user.PasswordHash) {
		return nil, fmt.Errorf("用户名或密码错误")
	}
	
	// 检查用户是否激活
	if !user.IsActive {
		return nil, fmt.Errorf("账户已被禁用")
	}
	
	// 生成JWT令牌
	token, expiresAt, err := s.generateJWT(user)
	if err != nil {
		return nil, fmt.Errorf("生成令牌失败: %v", err)
	}
	
	// 创建会话
	session := &model.UserSession{
		ID:        s.generateSessionID(),
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
		IPAddress: ipAddress,
		UserAgent: userAgent,
		IsActive:  true,
	}
	
	s.sessions[session.ID] = session
	
	// 更新用户登录信息
	user.LastLoginAt = time.Now()
	user.LoginCount++
	user.UpdatedAt = time.Now()
	
	// 保存数据
	if err := s.saveUsers(); err != nil {
		return nil, fmt.Errorf("保存用户失败: %v", err)
	}
	if err := s.saveSessions(); err != nil {
		return nil, fmt.Errorf("保存会话失败: %v", err)
	}
	
	// 获取会员信息
	membership := s.getMembership(user.ID)
	
	// 构建响应
	response := &model.LoginResponse{
		User:        *user,
		Token:       token,
		ExpiresAt:   expiresAt,
		Membership:  membership,
		Permissions: user.GetUserPermissions(),
	}
	
	return response, nil
}

// Logout 用户登出
func (s *AuthService) Logout(token string) error {
	// 查找并删除会话
	for sessionID, session := range s.sessions {
		if session.Token == token {
			session.IsActive = false
			delete(s.sessions, sessionID)
			break
		}
	}
	
	return s.saveSessions()
}

// ValidateToken 验证令牌
func (s *AuthService) ValidateToken(token string) (*model.User, error) {
	// 解析JWT令牌
	claims, err := s.parseJWT(token)
	if err != nil {
		return nil, fmt.Errorf("无效的令牌: %v", err)
	}
	
	// 检查令牌是否过期
	if time.Now().After(claims.ExpiresAt.Time) {
		return nil, fmt.Errorf("令牌已过期")
	}
	
	// 查找用户
	user, exists := s.users[claims.UserID]
	if !exists {
		return nil, fmt.Errorf("用户不存在")
	}
	
	// 检查用户是否激活
	if !user.IsActive {
		return nil, fmt.Errorf("账户已被禁用")
	}
	
	// 检查会话是否有效
	for _, session := range s.sessions {
		if session.Token == token && session.IsActive {
			return user, nil
		}
	}
	
	return nil, fmt.Errorf("会话无效")
}

// GetUser 获取用户信息
func (s *AuthService) GetUser(userID string) (*model.User, error) {
	user, exists := s.users[userID]
	if !exists {
		return nil, fmt.Errorf("用户不存在")
	}
	return user, nil
}

// UpdateUser 更新用户信息
func (s *AuthService) UpdateUser(userID string, req *model.UserUpdateRequest) (*model.User, error) {
	user, exists := s.users[userID]
	if !exists {
		return nil, fmt.Errorf("用户不存在")
	}
	
	// 更新用户信息
	if req.Nickname != "" {
		user.Profile.Nickname = req.Nickname
	}
	if req.Bio != "" {
		user.Profile.Bio = req.Bio
	}
	if req.Location != "" {
		user.Profile.Location = req.Location
	}
	if req.Website != "" {
		user.Profile.Website = req.Website
	}
	if req.Preferences != nil {
		user.Profile.Preferences = *req.Preferences
	}
	
	user.UpdatedAt = time.Now()
	
	// 保存用户
	if err := s.saveUsers(); err != nil {
		return nil, fmt.Errorf("保存用户失败: %v", err)
	}
	
	return user, nil
}

// ChangePassword 修改密码
func (s *AuthService) ChangePassword(userID string, req *model.PasswordChangeRequest) error {
	user, exists := s.users[userID]
	if !exists {
		return fmt.Errorf("用户不存在")
	}
	
	// 验证旧密码
	if !s.verifyPassword(req.OldPassword, user.PasswordHash) {
		return fmt.Errorf("旧密码错误")
	}
	
	// 加密新密码
	newPasswordHash, err := s.hashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("密码加密失败: %v", err)
	}
	
	// 更新密码
	user.PasswordHash = newPasswordHash
	user.UpdatedAt = time.Now()
	
	// 保存用户
	if err := s.saveUsers(); err != nil {
		return fmt.Errorf("保存用户失败: %v", err)
	}
	
	return nil
}

// UpgradeMembership 升级会员
func (s *AuthService) UpgradeMembership(userID string, req *model.MembershipUpgradeRequest) (*model.Membership, error) {
	user, exists := s.users[userID]
	if !exists {
		return nil, fmt.Errorf("用户不存在")
	}
	
	// 计算到期时间
	expiresAt := time.Now().AddDate(0, req.Months, 0)
	
	// 创建或更新会员信息
	membership := &model.Membership{
		UserID:        userID,
		Level:         req.Level,
		ExpiresAt:     expiresAt,
		IsActive:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		MaxSearches:   s.getMaxSearchesForLevel(req.Level),
		MaxConcurrency: s.getMaxConcurrencyForLevel(req.Level),
		Features:      s.getFeaturesForLevel(req.Level),
	}
	
	// 更新用户类型
	user.UserType = model.UserTypeMember
	user.UpdatedAt = time.Now()
	
	// 保存数据
	if err := s.saveUsers(); err != nil {
		return nil, fmt.Errorf("保存用户失败: %v", err)
	}
	
	return membership, nil
}

// 私有方法

func (s *AuthService) getUserByUsername(username string) *model.User {
	for _, user := range s.users {
		if user.Username == username {
			return user
		}
	}
	return nil
}

func (s *AuthService) getUserByEmail(email string) *model.User {
	for _, user := range s.users {
		if user.Email == email {
			return user
		}
	}
	return nil
}

func (s *AuthService) getMembership(userID string) *model.Membership {
	// 这里可以实现会员信息存储和查询
	// 暂时返回nil，表示没有会员信息
	return nil
}

func (s *AuthService) generateUserID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (s *AuthService) generateSessionID() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (s *AuthService) hashPassword(password string) (string, error) {
	hash := sha256.Sum256([]byte(password + "pansou-salt"))
	return hex.EncodeToString(hash[:]), nil
}

func (s *AuthService) verifyPassword(password, hash string) bool {
	hashedPassword, err := s.hashPassword(password)
	if err != nil {
		return false
	}
	return hashedPassword == hash
}

func (s *AuthService) generateJWT(user *model.User) (string, time.Time, error) {
	expiresAt := time.Now().Add(24 * time.Hour) // 24小时过期
	
	claims := &JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		UserType: string(user.UserType),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", time.Time{}, err
	}
	
	return tokenString, expiresAt, nil
}

func (s *AuthService) parseJWT(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})
	
	if err != nil {
		return nil, err
	}
	
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}
	
	return nil, fmt.Errorf("无效的令牌")
}

func (s *AuthService) getMaxSearchesForLevel(level int) int {
	switch level {
	case 1:
		return 50
	case 2:
		return 100
	case 3:
		return 200
	case 4:
		return 500
	case 5:
		return -1 // 无限制
	default:
		return 50
	}
}

func (s *AuthService) getMaxConcurrencyForLevel(level int) int {
	switch level {
	case 1:
		return 5
	case 2:
		return 10
	case 3:
		return 20
	case 4:
		return 30
	case 5:
		return 50
	default:
		return 5
	}
}

func (s *AuthService) getFeaturesForLevel(level int) []string {
	features := []string{"advanced_search", "search_history", "export"}
	
	if level >= 2 {
		features = append(features, "priority_support")
	}
	if level >= 3 {
		features = append(features, "api_access")
	}
	if level >= 4 {
		features = append(features, "custom_plugins")
	}
	if level >= 5 {
		features = append(features, "unlimited_searches", "white_label")
	}
	
	return features
}

func (s *AuthService) loadUsers() error {
	if !util.FileExists(s.usersFile) {
		return nil
	}
	
	data, err := os.ReadFile(s.usersFile)
	if err != nil {
		return err
	}
	
	return json.Unmarshal(data, &s.users)
}

func (s *AuthService) saveUsers() error {
	data, err := json.MarshalIndent(s.users, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(s.usersFile, data, 0644)
}

func (s *AuthService) loadSessions() error {
	if !util.FileExists(s.sessionsFile) {
		return nil
	}
	
	data, err := os.ReadFile(s.sessionsFile)
	if err != nil {
		return err
	}
	
	return json.Unmarshal(data, &s.sessions)
}

func (s *AuthService) saveSessions() error {
	data, err := json.MarshalIndent(s.sessions, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(s.sessionsFile, data, 0644)
}
