# PanSou MCP 服务文档

## 功能介绍

PanSou MCP 服务是一个基于 [Model Context Protocol (MCP)](https://modelcontextprotocol.io) 的工具服务，它将 PanSou 网盘搜索 API 的功能封装为可在支持 MCP 的客户端（如 Claude Desktop）中直接调用的工具。

通过 PanSou MCP 服务，可以直接在 Claude 等 AI 助手中搜索网盘资源，极大地提升了获取网盘资源的便捷性。

### 核心功能

1. **搜索网盘资源 (`search_netdisk`)**:
   - 支持通过关键词搜索网盘资源。
   - 可指定搜索来源：Telegram 频道、插件或两者结合。
   - 可过滤结果，仅显示特定类型的网盘链接（如百度网盘、阿里云盘、夸克网盘等）。
   - 支持强制刷新缓存以获取最新数据。
   - 支持传递扩展参数给后端插件。
   - 结果可按详细信息或按网盘类型分组展示。

2. **检查服务健康状态 (`check_service_health`)**:
   - 检查所连接的 PanSou 后端服务是否正常运行。
   - 获取后端服务的配置信息，如可用的 Telegram 频道列表和插件列表。

3. **启动后端服务 (`start_backend`)**:
   - 自动启动本地的 PanSou Go 后端服务（如果尚未运行）。
   - 等待服务完全启动并可用后才开始处理其他请求。

4. **获取静态资源信息 (`pansou://` URI scheme)**:
   - 提供可用插件列表、可用频道列表和支持的网盘类型列表等静态信息资源。

### 架构与部署方式

PanSou MCP 服务设计为与 PanSou Go 后端服务分离，通过 HTTP API 进行通信。当前支持 **npx 部署 (TypeScript)** 方式。

- **npx 部署 (TypeScript)**: MCP 服务基于 TypeScript 开发，编译后可以通过 `npx` 命令直接运行。它会自动连接到指定的 PanSou 后端服务。

---

## 安装与部署

### 前提条件

1. **Node.js**: 确保您的系统已安装 Node.js (版本 >= 18.0.0)。您可以通过在终端运行 `node -v` 来检查版本。
2. **Go**: 确保您的系统已安装 Go (版本 >= 1.18)。您可以通过在终端运行 `go version` 来检查版本。

### 部署步骤

PanSou 后端服务通常运行在 `http://localhost:8888` (默认地址)。目前，后端服务和 MCP 服务都需要从源码构建。

#### 1. 构建并启动 PanSou 后端服务 (Go)

- 确保系统已安装 Go 1.25.0 或更高版本。
- 克隆或确保已有 PanSou Go 项目源码。
- 在项目根目录下，打开终端并执行以下命令进行构建：

```bash
# Windows (PowerShell/CMD)
go build -o pansou.exe .
```

- 构建完成后，运行生成的可执行文件以启动后端服务：

```bash
# Windows
.\pansou.exe
```

服务默认将在 `http://localhost:8888` 启动。

#### 2. 构建 PanSou MCP 服务 (TypeScript)

- 确保系统已安装 Node.js (版本 >= 18.0.0)。
- 在 `typescript` 目录下，打开终端并执行以下命令来安装依赖并构建项目：

```bash
# 安装 Node.js 依赖
npm install

# 构建 TypeScript 项目
npm run build
```

构建完成后，编译后的 JavaScript 文件将位于 `typescript/dist` 目录下。

- 确保服务成功启动。您可以通过在终端中访问 `http://localhost:8888/api/health` 来检查，应该能看到类似以下的 JSON 响应，表明服务正常：

```json
{
  "status": "ok",
  "plugins_enabled": true,
  "channels_count": 1,
  "channels": ["tgsearchers3"],
  "plugin_count": 16,
  "plugins": ["pansearch", "panta", ...]
}
```

#### 3. 运行 MCP 服务

构建完成后，可以通过以下方式之一运行 MCP 服务：

- **在MCP调用时自动启动** (自动启动):
  直接浏览后续MCP配置的内容，配置好MCP后，调用时会自动启动后端服务器。

- **使用 `node` 直接运行** (手动启动):
  在 PanSou 项目根目录下（包含 `typescript` 文件夹），运行：

  ```bash
  # Windows (CMD/PowerShell)
  node .\\typescript\\dist\\index.js

  ```

服务启动后，将默认尝试连接到 `http://localhost:8888` 的 PanSou 后端服务。

如果想要后端服务运行在不同的地址或端口上，需要通过环境变量指定：

```bash
# Windows (CMD)
set PANSOU_SERVER_URL=http://your-backend-address:port
node .\\typescript\\dist\\index.js

# Windows (PowerShell)
export PANSOU_SERVER_URL=http://your-backend-address:port
node ./typescript/dist/index.js
```

#### 4. 示例配置 Cherry Studio(版本1.5.7)

要在 Cherry Studio 中使用 PanSou MCP 服务，需要将其添加到 Cherry Studio MCP 的配置文件中。

- 找到 设置中的MCP。
- 选择 `添加服务器` 、 `从JSON导入` 。
- 加入服务配置(可以直接复制mcp-config)的内容：

```json
{
  "mcpServers": {
    "pansou": {
      "command": "node",
      "args": [
        "C:\\full\\path\\to\\your\\project\\typescript\\dist\\index.js"
      ],
      "env": {
        "PANSOU_SERVER_URL": "http://localhost:8888",
        "REQUEST_TIMEOUT": "30",
        "MAX_RESULTS": "50",
        "DEFAULT_CLOUD_TYPES": "baidu,aliyun,quark,115",
        "AUTO_START_BACKEND": "true",
        "BACKEND_SHUTDOWN_DELAY": "5000",
        "BACKEND_STARTUP_TIMEOUT": "30000",
        "IDLE_TIMEOUT": "300000",
        "ENABLE_IDLE_SHUTDOWN": "true",
        "PROJECT_ROOT_PATH": "C:\\full\\path\\to\\your\\project"
      }
    }
  }
}
```

请将 `C:\\full\\path\\to\\your\\project` 替换为您项目实际的完整路径。

#### 5. 启动 MCP 服务，并在对话界面启用，开始尝试搜索

---

## 支持的参数

MCP 服务通过工具调用接收参数。以下是主要工具及其支持的参数：

### `search_netdisk` 工具

用于搜索网盘资源。

| 参数名          | 类型            | 必填 | 默认值               | 描述                                                         |
| :-------------- | :-------------- | :--- | :------------------- | :----------------------------------------------------------- |
| `keyword`       | string          | 是   | -                    | 搜索关键词，例如 "速度与激情"、"Python教程"。                |
| `channels`      | array of string | 否   | 配置默认值           | 要搜索的 Telegram 频道列表，例如 `["tgsearchers3", "another_channel"]`。 |
| `plugins`       | array of string | 否   | 配置默认值或所有插件 | 要使用的搜索插件列表，例如 `["pansearch", "panta"]`。        |
| `cloud_types`   | array of string | 否   | 无过滤               | 过滤结果，仅返回指定类型的网盘链接。支持的类型有：`baidu`, `aliyun`, `quark`, `tianyi`, `uc`, `mobile`, `115`, `pikpak`, `xunlei`, `123`, `magnet`, `ed2k`, `others`。 |
| `source_type`   | string          | 否   | `"all"`              | 数据来源类型。可选值：`"all"` (全部来源), `"tg"` (仅 Telegram), `"plugin"` (仅插件)。 |
| `force_refresh` | boolean         | 否   | `false`              | 是否强制刷新缓存，以获取最新数据。                           |
| `result_type`   | string          | 否   | `"merge"`            | 返回结果的类型。可选值：`"all"` (返回所有结果), `"results"` (仅返回详细结果), `"merge"` (仅返回按网盘类型分组的结果)。 |
| `concurrency`   | number          | 否   | 自动计算             | 并发搜索的数量。                                             |
| `ext_params`    | object          | 否   | `{}`                 | 传递给后端插件的自定义扩展参数，例如 `{"title_en": "Fast and Furious", "is_all": true}`。 |

---

### `check_service_health` 工具

用于检查后端服务健康状态。

- **参数**: 无

---

### `start_backend` 工具

用于启动本地 PanSou 后端服务。

| 参数名          | 类型    | 必填 | 默认值  | 描述                                       |
| :-------------- | :------ | :--- | :------ | :----------------------------------------- |
| `force_restart` | boolean | 否   | `false` | 是否强制重启后端服务（即使它已经在运行）。 |

---

### 环境变量配置

您可以通过设置环境变量来配置 MCP 服务的行为：

| 环境变量               | 描述                                                       | 默认值                    |
| :--------------------- | :--------------------------------------------------------- | :------------------------ |
| `PANSOU_SERVER_URL`    | PanSou 后端服务的 URL 地址。                               | `http://localhost:8888`   |
| `REQUEST_TIMEOUT`      | HTTP 请求超时时间（秒）。                                  | `30`                      |
| `MAX_RESULTS`          | （内部使用，限制处理结果数量）                             | `100`                     |
| `DEFAULT_CHANNELS`     | 默认搜索的 Telegram 频道列表（逗号分隔）。                 | `""` (使用后端默认)       |
| `DEFAULT_PLUGINS`      | 默认使用的搜索插件列表（逗号分隔）。                       | `""` (使用后端默认或所有) |
| `DEFAULT_CLOUD_TYPES`  | 默认的网盘类型过滤器（逗号分隔）。                         | `""` (无过滤)             |
| `AUTO_START_BACKEND`   | 是否在 MCP 服务启动时自动尝试启动后端服务。                | `true`                    |
| `PROJECT_ROOT_PATH`    | PanSou 后端可执行文件所在的目录路径（用于自动启动）。      | 无                        |
| `IDLE_TIMEOUT`         | 空闲超时时间（毫秒），超过此时间无活动则可能关闭后端服务。 | `300000` (5分钟)          |
| `ENABLE_IDLE_SHUTDOWN` | 是否启用空闲超时自动关闭后端服务。                         | `true`                    |