# 盘搜API认证文档

## 概述

盘搜系统现在支持用户注册、登录和权限管理功能。系统支持两种用户类型：普通用户和会员用户。

## 用户类型

### 普通用户 (normal)
- 每日搜索次数：10次
- 最大并发数：3个
- 权限：基础搜索功能

### 会员用户 (member)
- 每日搜索次数：100次（根据会员等级）
- 最大并发数：10个（根据会员等级）
- 权限：高级搜索、搜索历史、导出功能

### 管理员 (admin)
- 无搜索次数限制
- 最大并发数：50个
- 权限：所有功能

## API端点

### 认证相关

#### 用户注册
```
POST /api/auth/register
```

**请求体：**
```json
{
  "username": "用户名",
  "email": "邮箱地址",
  "password": "密码",
  "nickname": "昵称（可选）"
}
```

**响应：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user": {
      "id": "用户ID",
      "username": "用户名",
      "email": "邮箱",
      "user_type": "normal",
      "is_active": true,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z",
      "profile": {
        "nickname": "昵称",
        "preferences": {
          "default_channels": ["tgsearchers3"],
          "default_plugins": [],
          "default_cloud_types": [],
          "search_history": true,
          "theme": "light",
          "language": "zh-CN"
        }
      }
    },
    "message": "注册成功"
  }
}
```

#### 用户登录
```
POST /api/auth/login
```

**请求体：**
```json
{
  "username": "用户名或邮箱",
  "password": "密码",
  "remember": true
}
```

**响应：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user": {
      "id": "用户ID",
      "username": "用户名",
      "email": "邮箱",
      "user_type": "normal",
      "is_active": true,
      "last_login_at": "2024-01-01T00:00:00Z",
      "login_count": 1,
      "profile": {
        "nickname": "昵称",
        "preferences": {
          "default_channels": ["tgsearchers3"],
          "default_plugins": [],
          "default_cloud_types": [],
          "search_history": true,
          "theme": "light",
          "language": "zh-CN"
        }
      }
    },
    "token": "JWT令牌",
    "expires_at": "2024-01-02T00:00:00Z",
    "membership": null,
    "permissions": ["search"]
  }
}
```

#### 用户登出
```
POST /api/auth/logout
```

**请求头：**
```
Authorization: Bearer <JWT令牌>
```

**响应：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "登出成功"
  }
}
```

#### 刷新令牌
```
POST /api/auth/refresh
```

**请求头：**
```
Authorization: Bearer <JWT令牌>
```

**响应：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "新的JWT令牌",
    "expires_at": "2024-01-02T00:00:00Z",
    "message": "令牌刷新成功"
  }
}
```

### 用户管理

#### 获取用户资料
```
GET /api/user/profile
```

**请求头：**
```
Authorization: Bearer <JWT令牌>
```

**响应：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user": {
      "id": "用户ID",
      "username": "用户名",
      "email": "邮箱",
      "user_type": "normal",
      "is_active": true,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z",
      "last_login_at": "2024-01-01T00:00:00Z",
      "login_count": 1,
      "profile": {
        "nickname": "昵称",
        "bio": "个人简介",
        "location": "所在地",
        "website": "个人网站",
        "preferences": {
          "default_channels": ["tgsearchers3"],
          "default_plugins": [],
          "default_cloud_types": [],
          "search_history": true,
          "theme": "light",
          "language": "zh-CN"
        }
      }
    }
  }
}
```

#### 更新用户资料
```
PUT /api/user/profile
```

**请求头：**
```
Authorization: Bearer <JWT令牌>
```

**请求体：**
```json
{
  "nickname": "新昵称",
  "bio": "个人简介",
  "location": "所在地",
  "website": "个人网站",
  "preferences": {
    "default_channels": ["tgsearchers3", "Aliyun_4K_Movies"],
    "default_plugins": ["labi", "zhizhen"],
    "default_cloud_types": ["baidu", "aliyun"],
    "search_history": true,
    "theme": "dark",
    "language": "zh-CN"
  }
}
```

#### 修改密码
```
POST /api/user/change-password
```

**请求头：**
```
Authorization: Bearer <JWT令牌>
```

**请求体：**
```json
{
  "old_password": "旧密码",
  "new_password": "新密码"
}
```

#### 升级会员
```
POST /api/user/upgrade-membership
```

**请求头：**
```
Authorization: Bearer <JWT令牌>
```

**请求体：**
```json
{
  "level": 1,
  "months": 12,
  "payment_method": "alipay"
}
```

#### 获取用户统计
```
GET /api/user/stats
```

**请求头：**
```
Authorization: Bearer <JWT令牌>
```

**响应：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "stats": {
      "total_searches": 0,
      "today_searches": 0,
      "last_search_at": "2024-01-01T00:00:00Z",
      "favorite_channels": ["tgsearchers3"],
      "favorite_plugins": []
    }
  }
}
```

### 搜索功能

#### 基础搜索（可选认证）
```
POST /api/search
GET /api/search
```

**请求头（可选）：**
```
Authorization: Bearer <JWT令牌>
```

**请求体：**
```json
{
  "kw": "搜索关键词",
  "channels": ["tgsearchers3"],
  "plugins": ["labi", "zhizhen"],
  "cloud_types": ["baidu", "aliyun"],
  "conc": 5,
  "refresh": false,
  "res": "merged_by_type",
  "src": "all",
  "ext": {}
}
```

#### 高级搜索（需要会员权限）
```
POST /api/search/advanced
GET /api/search/advanced
```

**请求头：**
```
Authorization: Bearer <JWT令牌>
```

#### 搜索历史（需要认证）
```
GET /api/search/history
DELETE /api/search/history
```

**请求头：**
```
Authorization: Bearer <JWT令牌>
```

## 权限说明

### 普通用户权限
- `search`: 基础搜索功能

### 会员用户权限
- `search`: 基础搜索功能
- `advanced_search`: 高级搜索功能
- `history`: 搜索历史功能
- `export`: 导出功能

### 管理员权限
- 所有会员权限
- `api`: API访问权限
- `admin`: 管理员权限

## 使用示例

### JavaScript/Node.js
```javascript
// 用户注册
const registerResponse = await fetch('http://localhost:8888/api/auth/register', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    username: 'testuser',
    email: 'test@example.com',
    password: 'password123',
    nickname: '测试用户'
  })
});

// 用户登录
const loginResponse = await fetch('http://localhost:8888/api/auth/login', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    username: 'testuser',
    password: 'password123',
    remember: true
  })
});

const loginData = await loginResponse.json();
const token = loginData.data.token;

// 带认证的搜索请求
const searchResponse = await fetch('http://localhost:8888/api/search', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`
  },
  body: JSON.stringify({
    kw: '测试搜索',
    channels: ['tgsearchers3'],
    conc: 5
  })
});
```

### Python
```python
import requests

# 用户注册
register_data = {
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123",
    "nickname": "测试用户"
}

register_response = requests.post(
    'http://localhost:8888/api/auth/register',
    json=register_data
)

# 用户登录
login_data = {
    "username": "testuser",
    "password": "password123",
    "remember": True
}

login_response = requests.post(
    'http://localhost:8888/api/auth/login',
    json=login_data
)

token = login_response.json()['data']['token']

# 带认证的搜索请求
search_data = {
    "kw": "测试搜索",
    "channels": ["tgsearchers3"],
    "conc": 5
}

headers = {
    'Authorization': f'Bearer {token}'
}

search_response = requests.post(
    'http://localhost:8888/api/search',
    json=search_data,
    headers=headers
)
```

## 错误码说明

- `400`: 请求参数错误
- `401`: 未认证或认证失败
- `403`: 权限不足
- `404`: 资源不存在
- `500`: 服务器内部错误

## 注意事项

1. JWT令牌有效期为24小时
2. 用户密码使用SHA256加密存储
3. 未认证用户有搜索限制（最大3个并发）
4. 会员用户根据等级有不同的搜索限制
5. 所有API都支持CORS跨域请求
6. 建议在生产环境中修改JWT密钥
