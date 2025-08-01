# Zhizhen API 数据结构分析

## 基本信息
- **数据源类型**: JSON API  
- **API URL格式**: `https://xiaomi666.fun/api.php/provide/vod?ac=detail&wd={关键词}`
- **数据特点**: 视频点播(VOD)系统API，提供结构化影视资源数据
- **特殊说明**: 使用独立域名，网盘标识符与wanou/ouge略有不同

## API响应结构

### 顶层结构
```json
{
    "code": 1,                    // 状态码：1表示成功
    "msg": "数据列表",             // 响应消息
    "page": 1,                    // 当前页码
    "pagecount": 1,               // 总页数
    "limit": 20,                  // 每页限制条数
    "total": 6,                   // 总记录数
    "list": []                    // 数据列表数组
}
```

### `list`数组中的数据项结构
```json
{
    "vod_id": 11455,                    // 资源唯一ID
    "vod_name": "凡人修仙传真人版",      // 资源标题
    "vod_actor": "杨洋,金晨,汪铎...",    // 主演（逗号分隔）
    "vod_director": "杨阳",             // 导演
    "vod_area": "大陆",                 // 地区
    "vod_lang": "国语",                 // 语言
    "vod_year": "2025",                 // 年份
    "vod_remarks": "更新至11集",        // 更新状态/备注
    "vod_pubdate": "",                  // 发布日期（可能为空）
    "vod_blurb": "该剧改编自忘语...",    // 简介
    "vod_content": "该剧改编自忘语...",  // 内容描述
    "vod_pic": "https://...",           // 封面图片URL
    
    // 关键字段：下载链接相关
    "vod_down_from": "kuake$$$BAIDUI$$$kuake",
    "vod_down_url": "https://pan.quark.cn/s/d228bf3a6e44$$$https://pan.baidu.com/s/1kOWHnazfGFe6wJ-tin2pNQ?pwd=b2s4$$$https://pan.quark.cn/s/12e29bdacec4"
}
```

## 插件所需字段映射

| 源字段 | 目标字段 | 说明 |
|--------|----------|------|
| `vod_id` | `UniqueID` | 格式: `zhizhen-{vod_id}` |
| `vod_name` | `Title` | 资源标题 |
| `vod_actor`, `vod_director`, `vod_area`, `vod_year`, `vod_remarks` | `Content` | 组合描述信息 |
| `vod_year`, `vod_area` | `Tags` | 标签数组 |
| `vod_down_from` + `vod_down_url` | `Links` | 解析为Link数组 |
| `""` | `Channel` | 插件搜索结果Channel为空 |
| `time.Now()` | `Datetime` | 当前时间 |

## 下载链接解析

### 分隔符规则
- **多个下载源**: 使用 `$$$` 分隔
- **对应关系**: `vod_down_from`、`vod_down_url` 按相同位置对应

### 下载源标识映射（zhizhen特有）
| API标识 | 网盘类型 | 域名示例 | 备注 |
|---------|----------|----------|------|
| `kuake` | quark (夸克网盘) | `pan.quark.cn` | ⚠️ 使用`kuake`而非`KG` |
| `BAIDUI` | baidu (百度网盘) | `pan.baidu.com` | ⚠️ 使用`BAIDUI`而非`bd` |
| `UC` | uc (UC网盘) | `drive.uc.cn` | 与标准一致 |

### 多源示例数据
```
vod_down_from: "kuake$$$BAIDUI$$$UC$$$BAIDUI"
vod_down_url: "https://pan.quark.cn/s/24afb59cd9ae$$$https://pan.baidu.com/s/1d8bHaARjn60rlY_5mN3phA?pwd=ceda$$$https://drive.uc.cn/s/40f6a8d5c9804?public=1$$$https://pan.baidu.com/s/19CVP2d8_ka901b9myBh68w?pwd=begh"
```

### 链接格式示例
```
夸克网盘: https://pan.quark.cn/s/d228bf3a6e44
百度网盘: https://pan.baidu.com/s/1kOWHnazfGFe6wJ-tin2pNQ?pwd=b2s4
UC网盘: https://drive.uc.cn/s/40f6a8d5c9804?public=1
```

## 支持的网盘类型（16种）

### 主流网盘
- **baidu (百度网盘)**: `https://pan.baidu.com/s/{分享码}?pwd={密码}`
- **quark (夸克网盘)**: `https://pan.quark.cn/s/{分享码}`
- **aliyun (阿里云盘)**: `https://aliyundrive.com/s/{分享码}`, `https://www.alipan.com/s/{分享码}`
- **uc (UC网盘)**: `https://drive.uc.cn/s/{分享码}`
- **xunlei (迅雷网盘)**: `https://pan.xunlei.com/s/{分享码}`

### 运营商网盘
- **tianyi (天翼云盘)**: `https://cloud.189.cn/t/{分享码}`
- **mobile (移动网盘)**: `https://caiyun.feixin.10086.cn/{分享码}`

### 专业网盘
- **115 (115网盘)**: `https://115.com/s/{分享码}`
- **weiyun (微云)**: `https://share.weiyun.com/{分享码}`
- **lanzou (蓝奏云)**: `https://lanzou.com/{分享码}`
- **jianguoyun (坚果云)**: `https://jianguoyun.com/{分享码}`
- **123 (123网盘)**: `https://123pan.com/s/{分享码}`
- **pikpak (PikPak)**: `https://mypikpak.com/s/{分享码}`

### 其他协议
- **magnet (磁力链接)**: `magnet:?xt=urn:btih:{hash}`
- **ed2k (电驴链接)**: `ed2k://|file|{filename}|{size}|{hash}|/`
- **others (其他类型)**: 其他不在上述分类中的链接

## 插件开发指导

### 请求示例
```go
searchURL := fmt.Sprintf("https://xiaomi666.fun/api.php/provide/vod?ac=detail&wd=%s", url.QueryEscape(keyword))
```

### SearchResult构建示例
```go
result := model.SearchResult{
    UniqueID: fmt.Sprintf("zhizhen-%d", item.VodID),
    Title:    item.VodName,
    Content:  buildContent(item),
    Links:    parseDownloadLinks(item.VodDownFrom, item.VodDownURL),
    Tags:     []string{item.VodYear, item.VodArea},
    Channel:  "", // 插件搜索结果Channel为空
    Datetime: time.Now(),
}
```

### 特殊映射函数
```go
func (p *ZhizhenAsyncPlugin) mapCloudType(apiType, url string) string {
    // 优先根据API标识映射（zhizhen特有）
    switch strings.ToUpper(apiType) {
    case "KUAKE":
        return "quark"
    case "BAIDUI":
        return "baidu"
    case "UC":
        return "uc"
    }
    
    // 如果API标识无法识别，则通过URL模式匹配
    return p.determineLinkType(url)
}
```

### 链接解析逻辑
```go
// 按$$$分隔
fromParts := strings.Split(item.VodDownFrom, "$$$")
urlParts := strings.Split(item.VodDownURL, "$$$")

// 遍历对应位置
for i := 0; i < min(len(fromParts), len(urlParts)); i++ {
    linkType := p.mapCloudType(fromParts[i], urlParts[i])
    password := extractPassword(urlParts[i])
    // ...
}
```

## 与其他插件的差异

| 特性 | zhizhen | wanou/ouge | 说明 |
|------|---------|------------|------|
| **API域名** | `xiaomi666.fun` | `woog.nxog.eu.org` | 不同域名 |
| **夸克标识** | `kuake` | `KG` | 标识符不同 |
| **百度标识** | `BAIDUI` | `bd` | 标识符不同 |
| **UC标识** | `UC` | `UC` | 一致 |
| **数据结构** | 相同 | 相同 | JSON结构完全一致 |

## 注意事项
1. **标识符差异**: 需要专门处理`kuake`和`BAIDUI`标识符
2. **数据格式**: 纯JSON API，无需HTML解析  
3. **分隔符处理**: 多个值使用`$$$`分隔，需要split处理
4. **密码提取**: 部分百度网盘链接包含`?pwd=`参数
5. **错误处理**: API可能返回`code != 1`的错误状态
6. **链接验证**: 应过滤无效链接（如`javascript:;`等）

## 开发建议
- **基于wanou改造**: 可以复制wanou插件实现，修改域名和标识符映射
- **映射函数重点**: 关键是正确处理`kuake`→`quark`和`BAIDUI`→`baidu`的映射
- **测试覆盖**: 重点测试多种网盘类型的混合链接解析
- **缓存策略**: 建议使用相同的缓存机制和TTL设置