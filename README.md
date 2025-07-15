# PanSou 网盘搜索API

PanSou是一个高性能的网盘资源搜索API服务，支持TG搜索和网盘搜索引擎。系统设计以性能和可扩展性为核心，支持多频道并发搜索、结果智能排序和网盘类型分类。


## 特性

- **高性能搜索**：并发搜索多个Telegram频道，显著提升搜索速度；工作池设计，高效管理并发任务
- **网盘类型分类**：自动识别多种网盘链接，按类型归类展示
- **智能排序**：基于时间和关键词权重的多级排序策略
- **插件系统**：支持通过插件扩展搜索来源，已内置多个网盘搜索插件；支持"尽快响应，持续处理"的异步搜索模式
- **两级异步缓存**：内存+分片磁盘缓存机制，大幅提升重复查询速度和并发性能，即使在不强制刷新的情况下也能获取异步插件更新的最新缓存数据

## 支持的网盘类型

- 百度网盘 (`pan.baidu.com`)
- 阿里云盘 (`aliyundrive.com`, `alipan.com`)
- 夸克网盘 (`pan.quark.cn`)
- 天翼云盘 (`cloud.189.cn`)
- UC网盘 (`drive.uc.cn`)
- 移动云盘 (`caiyun.139.com`)
- 115网盘 (`115.com`, `115cdn.com`, `anxia.com`)
- PikPak (`mypikpak.com`)
- 迅雷网盘 (`pan.xunlei.com`)
- 123网盘 (`123684.com`, `123685.com`, `123912.com`, `123pan.com`, `123pan.cn`, `123592.com`)
- 磁力链接 (`magnet:?xt=urn:btih:`)
- 电驴链接 (`ed2k://`)

## 内置搜索插件

PanSou内置了多个网盘搜索插件，可以扩展搜索来源

### 环境要求

- Go 1.18+
- 可选：SOCKS5代理（用于访问受限地区的Telegram站点）

### 从源码安装

1. 克隆仓库

```bash
git clone https://github.com/fish2018/pansou.git
cd pansou
```

2. 配置环境变量（可选）

```bash
# 默认频道
export CHANNELS="tgsearchers2,xxx"

# 缓存配置
export CACHE_ENABLED=true
export CACHE_PATH="./cache"
export CACHE_MAX_SIZE=100  # MB
export CACHE_TTL=60        # 分钟
export SHARD_COUNT=8       # 分片数量

# 异步插件配置
export ASYNC_PLUGIN_ENABLED=true
export ASYNC_RESPONSE_TIMEOUT=2            # 响应超时时间（秒）
export ASYNC_MAX_BACKGROUND_WORKERS=20     # 最大后台工作者数量
export ASYNC_MAX_BACKGROUND_TASKS=100      # 最大后台任务数量
export ASYNC_CACHE_TTL_HOURS=1             # 异步缓存有效期（小时）

# 代理配置（如需）
export PROXY="socks5://127.0.0.1:7890"
```

3. 构建

```linux
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -extldflags '-static'" -o pansou .
```

4. 运行

```bash
./pansou
```

## API文档

### 搜索API

搜索网盘资源。

**接口地址**：`/api/search`  
**请求方法**：`POST` 或 `GET`  
**Content-Type**：`application/json`（POST方法）

**POST请求参数**：

| 参数名 | 类型 | 必填 | 描述 |
|--------|------|------|------|
| kw | string | 是 | 搜索关键词 |
| channels | string[] | 否 | 搜索的频道列表，不提供则使用默认配置 |
| conc | number | 否 | 并发搜索数量，不提供则自动设置为频道数+插件数+10 |
| refresh | boolean | 否 | 强制刷新，不使用缓存，便于调试和获取最新数据 |
| res | string | 否 | 结果类型：all(返回所有结果)、results(仅返回results)、merge(仅返回merged_by_type)，默认为merge |
| src | string | 否 | 数据来源类型：all(默认，全部来源)、tg(仅Telegram)、plugin(仅插件) |
| plugins | string[] | 否 | 指定搜索的插件列表，不指定则搜索全部插件 |

**GET请求参数**：

| 参数名 | 类型 | 必填 | 描述 |
|--------|------|------|------|
| kw | string | 是 | 搜索关键词 |
| channels | string | 否 | 搜索的频道列表，使用英文逗号分隔多个频道，不提供则使用默认配置 |
| conc | number | 否 | 并发搜索数量，不提供则自动设置为频道数+插件数+10 |
| refresh | boolean | 否 | 强制刷新，设置为"true"表示不使用缓存 |
| res | string | 否 | 结果类型：all(返回所有结果)、results(仅返回results)、merge(仅返回merged_by_type)，默认为merge |
| src | string | 否 | 数据来源类型：all(默认，全部来源)、tg(仅Telegram)、plugin(仅插件) |
| plugins | string | 否 | 指定搜索的插件列表，使用英文逗号分隔多个插件名，不指定则搜索全部插件 |

**POST请求示例**：

```json
{
  "kw": "速度与激情",
  "channels": ["tgsearchers2", "xxx"],
  "conc": 2,
  "refresh": true,
  "res": "merge",
  "src": "all",
  "plugins": ["jikepan"]
}
```

**GET请求示例**：

```
GET /api/search?kw=速度与激情&channels=tgsearchers2,xxx&conc=2&refresh=true&res=merge&src=tg
```

**成功响应**：

```json
{
  "total": 15,
  "results": [
    {
      "message_id": "12345",
      "unique_id": "channel-12345",
      "channel": "tgsearchers2",
      "datetime": "2023-06-10T14:23:45Z",
      "title": "速度与激情全集1-10",
      "content": "速度与激情系列全集，1080P高清...",
      "links": [
        {
          "type": "baidu",
          "url": "https://pan.baidu.com/s/1abcdef",
          "password": "1234"
        }
      ],
      "tags": ["电影", "合集"]
    },
    // 更多结果...
  ],
  "merged_by_type": {
    "baidu": [
      {
        "url": "https://pan.baidu.com/s/1abcdef",
        "password": "1234",
        "note": "速度与激情全集1-10",
        "datetime": "2023-06-10T14:23:45Z"
      },
      // 更多百度网盘链接...
    ],
    "aliyun": [
      // 阿里云盘链接...
    ]
    // 更多网盘类型...
  }
}
```

**错误响应**：

```json
{
  "code": 400,
  "message": "关键词不能为空"
}
```

### 健康检查

检查API服务是否正常运行。

**接口地址**：`/api/health`  
**请求方法**：`GET`

**成功响应**：

```json
{
  "status": "ok",
}
```

## 配置指南

### 环境变量

| 环境变量 | 描述 | 默认值 |
|----------|------|--------|
| CHANNELS | 默认搜索频道列表（逗号分隔） | tgsearchers2 |
| CONCURRENCY | 默认并发数 | 频道数+插件数+10 |
| PORT | 服务端口 | 8888 |
| PROXY | SOCKS5代理 | - |
| CACHE_ENABLED | 是否启用缓存 | true |
| CACHE_PATH | 缓存文件路径 | ./cache |
| CACHE_MAX_SIZE | 最大缓存大小(MB) | 100 |
| CACHE_TTL | 缓存生存时间(分钟) | 60 |
| SHARD_COUNT | 缓存分片数量 | 8 |
| SERIALIZER_TYPE | 序列化器类型(gob/json) | gob |
| ENABLE_COMPRESSION | 是否启用压缩 | false |
| MIN_SIZE_TO_COMPRESS | 最小压缩阈值(字节) | 1024 |
| GC_PERCENT | GC触发百分比 | 100 |
| OPTIMIZE_MEMORY | 是否优化内存 | true |
| PLUGIN_TIMEOUT | 插件执行超时时间(秒) | 30 |
| ASYNC_PLUGIN_ENABLED | 是否启用异步插件 | true |
| ASYNC_RESPONSE_TIMEOUT | 异步响应超时时间(秒) | 4 |
| ASYNC_MAX_BACKGROUND_WORKERS | 最大后台工作者数量 | 20 |
| ASYNC_MAX_BACKGROUND_TASKS | 最大后台任务数量 | 100 |
| ASYNC_CACHE_TTL_HOURS | 异步缓存有效期(小时) | 1 |
| CACHE_FRESHNESS_SECONDS | 缓存数据新鲜度(秒) | 30 |

## 性能优化

PanSou 实现了多项性能优化技术：

1. **JSON处理优化**：使用 sonic 高性能 JSON 库
2. **内存优化**：预分配策略、对象池化、GC参数优化
3. **缓存优化**：
   - 增强版两级缓存（内存+分片磁盘）
   - 分片磁盘缓存减少锁竞争
   - 高效序列化（Gob和JSON双支持）
   - 分离的缓存键（TG搜索和插件搜索独立缓存）
   - 缓存数据时间戳检查（获取最新数据）
   - 异步写入（不阻塞主流程）
4. **HTTP客户端优化**：连接池、HTTP/2支持
5. **并发优化**：工作池、智能并发控制
6. **传输压缩**：支持 gzip 压缩

## 异步插件系统

PanSou实现了高级异步插件系统，解决了某些搜索源响应时间长的问题：

### 异步插件特性

- **双级超时控制**：短超时(2秒)确保快速响应，长超时(30秒)允许完整处理
- **持久化缓存**：缓存自动保存到磁盘，系统重启后自动恢复
- **即时保存**：缓存更新后立即触发保存，不再等待定时器
- **优雅关闭**：在程序退出前保存缓存，确保数据不丢失
- **增量更新**：智能合并新旧结果，保留有价值的数据
- **后台自动刷新**：对于接近过期的缓存，在后台自动刷新
- **与主程序缓存协同**：通过时间戳检查机制，确保即使在不强制刷新的情况下也能获取最新缓存数据

### 缓存系统特点

1. **分片磁盘缓存**：
   - 将缓存数据分散到多个子目录，减少锁竞争
   - 通过哈希算法将缓存键均匀分布到不同分片
   - 提高高并发场景下的性能

2. **序列化器接口**：
   - 统一的序列化和反序列化操作
   - 支持Gob和JSON双序列化方式
   - Gob序列化提供更高性能和更小的结果大小

3. **缓存数据时间戳检查**：
   - 检查缓存数据是否是最新的（30秒内更新）
   - 确保获取异步插件在后台更新的最新缓存数据
   - 在不强制刷新的情况下也能获取最新数据

4. **分离的缓存键**：
   - TG搜索和插件搜索使用独立的缓存键
   - 实现独立更新，互不影响
   - 提高缓存命中率和更新效率