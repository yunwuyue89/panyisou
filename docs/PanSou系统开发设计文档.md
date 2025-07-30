# PanSou 网盘搜索系统开发设计文档

## 📋 文档目录

- [1. 项目概述](#1-项目概述)
- [2. 系统架构设计](#2-系统架构设计)
- [3. 异步插件系统](#3-异步插件系统)
- [4. 二级缓存系统](#4-二级缓存系统)  
- [5. 核心组件实现](#5-核心组件实现)
- [6. API接口设计](#6-api接口设计)
- [7. 插件开发框架](#7-插件开发框架)
- [8. 性能优化实现](#8-性能优化实现)
- [9. 技术选型说明](#9-技术选型说明)

---

## 1. 项目概述

### 1.1 项目定位

PanSou是一个高性能的网盘资源搜索API服务，支持TG搜索和自定义插件搜索。系统采用异步插件架构，具备二级缓存机制和并发控制能力，在MacBook Pro 8GB上能够支持500用户并发访问。

### 1.2 性能表现（实测数据）

- ✅ **500用户瞬时并发**: 100%成功率，平均响应167ms
- ✅ **200用户持续并发**: 30秒内处理4725请求，QPS=148
- ✅ **缓存命中**: 99.8%请求<100ms响应时间  
- ✅ **高可用性**: 长时间运行无故障

### 1.3 核心特性

- **异步插件系统**: 双级超时控制（4秒/30秒），渐进式结果返回
- **二级缓存系统**: 分片内存缓存+分片磁盘缓存，GOB序列化
- **工作池管理**: 基于`util/pool`的并发控制
- **智能结果合并**: `mergeSearchResults`函数实现去重合并
- **多网盘类型支持**: 自动识别12种网盘类型

---

## 2. 系统架构设计

### 2.1 整体架构流程

```mermaid
graph TB
    A[用户请求] --> B{API网关<br/>中间件}
    B --> C[参数解析<br/>与验证]
    C --> D[搜索服务<br/>SearchService]
    
    D --> E{数据来源<br/>选择}
    E -->|TG| F[Telegram搜索]
    E -->|Plugin| G[插件搜索]
    E -->|All| H[并发搜索]
    
    H --> F
    H --> G
    
    F --> I[TG频道搜索]
    I --> J[HTML解析]
    J --> K[链接提取]
    K --> L[结果标准化]
    
    G --> M[插件管理器<br/>PluginManager]
    M --> N[异步插件调度]
    N --> O[插件工作池]
    O --> P[HTTP客户端]
    P --> Q[目标网站API]
    Q --> R[响应解析]
    R --> S[结果过滤]
    
    L --> T{二级缓存系统}
    S --> T
    
    T --> U[分片内存缓存<br/>LRU + 原子操作]
    T --> V[分片磁盘缓存<br/>GOB序列化]
    
    U --> W[缓存检查]
    V --> W
    W --> X{缓存命中?}
    
    X -->|是| Y[缓存反序列化]
    X -->|否| Z[执行搜索]
    
    Z --> AA[异步更新缓存]
    AA --> U
    AA --> V
    
    Y --> BB[结果合并<br/>mergeSearchResults]
    AA --> BB
    
    BB --> CC[网盘类型分类]
    CC --> DD[智能排序<br/>时间+权重]
    DD --> EE[结果过滤<br/>cloud_types]
    EE --> FF[JSON响应]
    FF --> GG[用户]
    
    %% 异步处理流程
    N --> HH[短超时处理<br/>4秒]
    HH --> II{是否完成?}
    II -->|是| JJ[返回完整结果<br/>isFinal=true]
    II -->|否| KK[返回部分结果<br/>isFinal=false]
    
    KK --> LL[后台继续处理<br/>最长30秒]
    LL --> MM[完整结果获取]
    MM --> NN[主缓存更新]
    NN --> U
    
    %% 样式定义
    classDef cacheNode fill:#e1f5fe
    classDef pluginNode fill:#f3e5f5
    classDef searchNode fill:#e8f5e8
    classDef asyncNode fill:#fff3e0
    
    class T,U,V,W,X,Y,AA cacheNode
    class M,N,O,P,G pluginNode
    class F,I,J,K,L searchNode
    class HH,II,JJ,KK,LL,MM,NN asyncNode
```

### 2.2 异步插件工作流程

```mermaid
sequenceDiagram
    participant U as 用户
    participant API as API Gateway
    participant S as SearchService  
    participant PM as PluginManager
    participant P as AsyncPlugin
    participant WP as WorkerPool
    participant C as Cache
    participant EXT as 外部API

    U->>API: 搜索请求 (kw=关键词)
    API->>S: 处理搜索
    S->>C: 检查缓存
    
    alt 缓存命中
        C-->>S: 返回缓存数据 (<100ms)
        S-->>U: 返回结果
    else 缓存未命中
        S->>PM: 调度插件搜索
        PM->>P: 设置关键词和缓存键
        P->>WP: 提交异步任务
        
        par 短超时处理 (4秒)
            WP->>EXT: HTTP请求
            EXT-->>WP: 响应数据
            WP->>P: 解析结果
            P-->>S: 部分结果 (isFinal=false)
            S->>C: 缓存部分结果
            S-->>U: 快速响应
        and 后台完整处理 (最长30秒)
            WP->>EXT: 继续处理
            EXT-->>WP: 完整数据
            WP->>P: 完整结果
            P->>S: 最终结果 (isFinal=true)
            S->>C: 更新主缓存
            Note over C: 用户下次请求将获得完整结果
        end
    end
```

### 2.3 核心组件

#### 2.3.1 HTTP服务层 (`api/`)
- **router.go**: 路由配置
- **handler.go**: 请求处理逻辑
- **middleware.go**: 中间件（日志、CORS等）

#### 2.3.2 搜索服务层 (`service/`)
- **search_service.go**: 核心搜索逻辑，结果合并

#### 2.3.3 插件系统层 (`plugin/`)
- **plugin.go**: 插件接口定义
- **baseasyncplugin.go**: 异步插件基类
- **各插件目录**: jikepan、pan666、hunhepan等

#### 2.3.4 工具层 (`util/`)
- **cache/**: 二级缓存系统实现
- **pool/**: 工作池实现
- **其他工具**: HTTP客户端、解析工具等

---

## 3. 异步插件系统

### 3.1 设计理念

异步插件系统解决传统同步搜索响应慢的问题，采用"尽快响应，持续处理"策略：
- **4秒短超时**: 快速返回部分结果（`isFinal=false`）
- **30秒长超时**: 后台继续处理，获得完整结果（`isFinal=true`）
- **主动缓存更新**: 完整结果自动更新主缓存，下次访问更快

### 3.2 插件接口实现

基于`plugin/plugin.go`的实际接口：

```go
type AsyncSearchPlugin interface {
    Name() string
    Priority() int
    
    AsyncSearch(keyword string, searchFunc func(*http.Client, string, map[string]interface{}) ([]model.SearchResult, error), 
               mainCacheKey string, ext map[string]interface{}) ([]model.SearchResult, error)
    
    SetMainCacheKey(key string)
    SetCurrentKeyword(keyword string)
    Search(keyword string, ext map[string]interface{}) ([]model.SearchResult, error)
}
```

### 3.3 基础插件类

`plugin/baseasyncplugin.go`提供通用功能：

```go
type BaseAsyncPlugin struct {
    name              string
    priority          int
    cacheTTL          time.Duration
    mainCacheKey      string
    currentKeyword    string        // 用于日志显示
    httpClient        *http.Client
    mainCacheUpdater  func(string, []model.SearchResult, time.Duration, bool, string) error
}
```

### 3.4 已实现插件列表

当前系统包含以下插件（基于`main.go`的导入）：
- **hdr4k**
- **hunhepan**
- **jikepan**
- **pan666**
- **pansearch**
- **panta**
- **qupansou**
- **susu**
- **panyq**
- **xuexizhinan**

### 3.5 插件注册机制

```go
// 全局插件注册表（plugin/plugin.go）
var globalRegistry = make(map[string]AsyncSearchPlugin)

// 插件通过init()函数自动注册
func init() {
    p := &MyPlugin{
        BaseAsyncPlugin: plugin.NewBaseAsyncPlugin("myplugin", 3),
    }
    plugin.RegisterGlobalPlugin(p)
}
```

---

## 4. 二级缓存系统

### 4.1 实现架构

基于`util/cache/`目录的实际实现：

- **enhanced_two_level_cache.go**: 二级缓存主入口
- **sharded_memory_cache.go**: 分片内存缓存（LRU+原子操作）
- **sharded_disk_cache.go**: 分片磁盘缓存
- **serializer.go**: GOB序列化器
- **cache_key.go**: 缓存键生成和管理

### 4.2 分片缓存设计

#### 4.2.1 内存缓存分片
```go
// 基于CPU核心数的动态分片
type ShardedMemoryCache struct {
    shards    []*MemoryCacheShard
    shardMask uint32
}

// 每个分片独立锁，减少竞争
type MemoryCacheShard struct {
    data map[string]*CacheItem
    lock sync.RWMutex
}
```

#### 4.2.2 磁盘缓存分片
```go
// 磁盘缓存同样采用分片设计
type ShardedDiskCache struct {
    shards    []*DiskCacheShard  
    shardMask uint32
    basePath  string
}
```

### 4.3 缓存读写策略

#### 4.3.1 读取流程
1. **内存优先**: 先检查分片内存缓存
2. **磁盘回源**: 内存未命中时读取磁盘缓存
3. **异步加载**: 磁盘命中后异步加载到内存

#### 4.3.2 写入流程  
1. **双写模式**: 同时写入内存和磁盘
2. **原子操作**: 内存缓存使用原子操作
3. **GOB序列化**: 磁盘存储使用GOB格式

### 4.4 缓存键策略

`cache_key.go`实现了智能缓存键生成：

```go
// TG搜索和插件搜索使用不同的缓存键前缀
func GenerateTGCacheKey(keyword string, channels []string) string
func GeneratePluginCacheKey(keyword string, plugins []string) string
```

**优势**:
- 独立更新：TG和插件缓存互不影响
- 提高命中率：精确的键匹配
- 并发安全：分片设计减少锁竞争

### 4.5 序列化性能

使用GOB序列化（`serializer.go`）的实际优势：
- **性能**: 比JSON序列化快约30%
- **体积**: 比JSON小约20%
- **兼容**: Go原生支持，无外部依赖

---

## 5. 核心组件实现

### 5.1 工作池系统 (`util/pool/`)

#### 5.1.1 worker_pool.go 实现
- **批量任务处理**: `ExecuteBatchWithTimeout`方法
- **超时控制**: 支持任务级别的超时设置
- **并发限制**: 控制最大工作者数量

#### 5.1.2 object_pool.go 实现  
- **对象复用**: 减少内存分配和GC压力
- **线程安全**: 支持并发访问

### 5.2 HTTP服务配置

#### 5.2.1 服务器优化（基于config/config.go）
```go
// 自动计算HTTP连接数，防止资源耗尽
func getHTTPMaxConns() int {
    cpuCount := runtime.NumCPU()
    maxConns := cpuCount * 25  // 保守配置
    
    if maxConns < 100 {
        maxConns = 100
    }
    if maxConns > 500 {
        maxConns = 500  // 限制最大值
    }
    
    return maxConns
}
```

#### 5.2.2 连接池配置（基于util/http_util.go）
```go
// HTTP客户端优化配置
transport := &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 10,
    IdleConnTimeout:     90 * time.Second,
}
```

### 5.3 结果处理系统

#### 5.3.1 智能排序（service/search_service.go）
- **时间权重排序**: 基于时间和关键词权重
- **优先关键词**: 合集、系列、全、完等优先显示

#### 5.3.2 结果合并（mergeSearchResults函数）
- **去重合并**: 基于UniqueID去重
- **完整性选择**: 选择更完整的结果保留
- **增量更新**: 新结果与缓存结果智能合并

### 5.4 网盘类型识别

支持自动识别的网盘类型（共12种）：
- 百度网盘、阿里云盘、夸克网盘、天翼云盘
- UC网盘、移动云盘、115网盘、PikPak
- 迅雷网盘、123网盘、磁力链接、电驴链接

---

## 6. API接口设计

### 6.1 核心接口实现（基于api/handler.go）

#### 6.1.1 搜索接口
```
POST /api/search
GET  /api/search
```

**核心参数**:
- `kw`: 搜索关键词（必填）
- `channels`: TG频道列表
- `plugins`: 插件列表  
- `cloud_types`: 网盘类型过滤
- `ext`: 扩展参数（JSON格式）
- `refresh`: 强制刷新缓存
- `res`: 返回格式（merge/all/results）
- `src`: 数据源（all/tg/plugin）

#### 6.1.2 健康检查接口
```
GET /api/health
```

返回系统状态和已注册插件信息。

### 6.2 中间件系统（api/middleware.go）

- **日志中间件**: 记录请求响应，支持URL解码显示
- **CORS中间件**: 跨域请求支持
- **错误处理**: 统一错误响应格式

### 6.3 扩展参数系统

通过`ext`参数支持插件特定选项：
```json
{
  "title_en": "English Title",
  "is_all": true,
  "year": 2023
}
```

---

## 7. 插件开发框架

### 7.1 基础开发模板

```go
package myplugin

import (
    "net/http"
    "pansou/model"
    "pansou/plugin"
)

type MyPlugin struct {
    *plugin.BaseAsyncPlugin
}

func init() {
    p := &MyPlugin{
        BaseAsyncPlugin: plugin.NewBaseAsyncPlugin("myplugin", 3),
    }
    plugin.RegisterGlobalPlugin(p)
}

func (p *MyPlugin) Search(keyword string, ext map[string]interface{}) ([]model.SearchResult, error) {
    return p.AsyncSearch(keyword, p.searchImpl, p.GetMainCacheKey(), ext)
}

func (p *MyPlugin) searchImpl(client *http.Client, keyword string, ext map[string]interface{}) ([]model.SearchResult, error) {
    // 实现具体搜索逻辑
    // 1. 构建请求URL
    // 2. 发送HTTP请求  
    // 3. 解析响应数据
    // 4. 转换为标准格式
    // 5. 关键词过滤
    return plugin.FilterResultsByKeyword(results, keyword), nil
}
```

### 7.2 插件注册流程

1. **自动注册**: 通过`init()`函数自动注册到全局注册表
2. **管理器加载**: `PluginManager`统一管理所有插件
3. **导入触发**: 在`main.go`中通过空导入触发注册

### 7.3 开发最佳实践

- **命名规范**: 插件名使用小写字母
- **优先级设置**: 1-5，数字越小优先级越高
- **错误处理**: 详细错误信息，便于调试
- **资源管理**: 及时释放HTTP连接

---

## 8. 性能优化实现

### 8.1 环境配置优化

基于实际性能测试结果的配置方案：

#### 8.1.1 macOS优化配置
```bash
export HTTP_MAX_CONNS=200
export ASYNC_MAX_BACKGROUND_WORKERS=15
export ASYNC_MAX_BACKGROUND_TASKS=75
export CONCURRENCY=30
```

#### 8.1.2 服务器优化配置  
```bash
export HTTP_MAX_CONNS=500
export ASYNC_MAX_BACKGROUND_WORKERS=40
export ASYNC_MAX_BACKGROUND_TASKS=200
export CONCURRENCY=50
```

### 8.2 日志控制系统

基于`config.go`的日志控制：
```bash
export ASYNC_LOG_ENABLED=false  # 控制异步插件详细日志
```

异步插件缓存更新日志可通过环境变量开关，避免生产环境日志过多。

---

## 9. 技术选型说明

### 9.1 Go语言优势
- **并发支持**: 原生goroutine，适合高并发场景
- **性能优秀**: 编译型语言，接近C的性能
- **部署简单**: 单一可执行文件，无外部依赖
- **标准库丰富**: HTTP、JSON、并发原语完备

### 9.2 GIN框架选择
- **高性能**: 路由和中间件处理效率高
- **简洁易用**: API设计简洁，学习成本低  
- **中间件生态**: 丰富的中间件支持
- **社区活跃**: 文档完善，问题解决快

### 9.3 GOB序列化选择
- **性能优势**: 比JSON快约30%
- **体积优势**: 比JSON小约20%
- **Go原生**: 无需第三方依赖
- **类型安全**: 保持Go类型信息

### 9.4 无数据库架构
- **简化部署**: 无需数据库安装配置
- **降低复杂度**: 减少组件依赖
- **提升性能**: 避免数据库IO瓶颈
- **易于扩展**: 无状态设计，支持水平扩展