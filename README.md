# PanSou 网盘搜索API

PanSou是一个高性能的网盘资源搜索API服务，系统设计以性能和可扩展性为核心，支持多频道并发搜索、结果智能排序和网盘类型分类。


## 特性

- **高性能搜索**：并发搜索多个Telegram频道，显著提升搜索速度
- **智能排序**：基于时间和关键词权重的多级排序策略
- **网盘类型分类**：自动识别多种网盘链接，按类型归类展示
- **两级缓存**：内存+磁盘缓存机制，大幅提升重复查询速度
- **高并发支持**：工作池设计，高效管理并发任务
- **灵活扩展**：易于支持新的网盘类型和数据来源

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

## 快速开始

### 环境要求

- Go 1.18+
- 可选：SOCKS5代理（用于访问受限地区的Telegram站点）

### 安装

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
| keyword | string | 是 | 搜索关键词 |
| channels | string[] | 否 | 搜索的频道列表，不提供则使用默认配置 |
| concurrency | number | 否 | 并发搜索数量，不提供则自动设置为频道数+10 |
| force_refresh | boolean | 否 | 强制刷新，不使用缓存，便于调试和获取最新数据 |
| result_type | string | 否 | 结果类型：all(默认，返回所有结果)、results(仅返回results)、merged_by_type(仅返回merged_by_type) |

**GET请求参数**：

| 参数名 | 类型 | 必填 | 描述 |
|--------|------|------|------|
| keyword | string | 是 | 搜索关键词 |
| channels | string | 否 | 搜索的频道列表，可多次使用此参数指定多个频道，不提供则使用默认配置 |
| concurrency | number | 否 | 并发搜索数量，不提供则自动设置为频道数+10 |
| force_refresh | boolean | 否 | 强制刷新，设置为"true"表示不使用缓存 |
| result_type | string | 否 | 结果类型：all(默认，返回所有结果)、results(仅返回results)、merged_by_type(仅返回merged_by_type) |

**POST请求示例**：

```json
{
  "keyword": "速度与激情",
  "channels": ["tgsearchers2", "xxx"],
  "concurrency": 2,
  "force_refresh": true,
  "result_type": "merged_by_type"
}
```

**GET请求示例**：

```
GET /api/search?keyword=速度与激情&channels=tgsearchers2&channels=xxx&concurrency=2&force_refresh=true&result_type=all
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
| CONCURRENCY | 默认并发数 | 频道数+10 |
| PORT | 服务端口 | 8080 |
| PROXY | SOCKS5代理 | - |
| CACHE_ENABLED | 是否启用缓存 | true |
| CACHE_PATH | 缓存文件路径 | ./cache |
| CACHE_MAX_SIZE | 最大缓存大小(MB) | 100 |
| CACHE_TTL | 缓存生存时间(分钟) | 60 |
| ENABLE_COMPRESSION | 是否启用压缩 | false |
| MIN_SIZE_TO_COMPRESS | 最小压缩阈值(字节) | 1024 |
| GC_PERCENT | GC触发百分比 | 100 |
| OPTIMIZE_MEMORY | 是否优化内存 | true |

## 性能优化

PanSou 实现了多项性能优化技术：

1. **JSON处理优化**：使用 sonic 高性能 JSON 库
2. **内存优化**：预分配策略、对象池化、GC参数优化
3. **缓存优化**：两级缓存、异步写入、优化键生成
4. **HTTP客户端优化**：连接池、HTTP/2支持
5. **并发优化**：工作池、智能并发控制
6. **传输压缩**：支持 gzip 压缩