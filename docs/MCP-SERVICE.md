# PanSou MCP 服务文档

## 功能介绍

PanSou MCP 服务是一个基于 [Model Context Protocol (MCP)](https://modelcontextprotocol.io) 的工具服务，它将 PanSou 网盘搜索 API 的功能封装为可在支持 MCP 的客户端（如 Claude Desktop）中直接调用的工具。

通过这个 MCP 服务，用户可以直接在 Claude 等 AI 助手中搜索网盘资源，极大地提升了获取网盘资源的便捷性。

### 核心功能

1.  **搜索网盘资源 (`search_netdisk`)**:
    *   支持通过关键词搜索网盘资源。
    *   可指定搜索来源：Telegram 频道、插件或两者结合。
    *   可过滤结果，仅显示特定类型的网盘链接（如百度网盘、阿里云盘、夸克网盘等）。
    *   支持强制刷新缓存以获取最新数据。
    *   支持传递扩展参数给后端插件。
    *   结果可按详细信息或按网盘类型分组展示。

2.  **检查服务健康状态 (`check_service_health`)**:
    *   检查所连接的 PanSou 后端服务是否正常运行。
    *   获取后端服务的配置信息，如可用的 Telegram 频道列表和插件列表。

3.  **启动后端服务 (`start_backend`)**:
    *   自动启动本地的 PanSou Go 后端服务（如果尚未运行）。
    *   等待服务完全启动并可用后才开始处理其他请求。

4.  **获取静态资源信息 (`pansou://` URI scheme)**:
    *   提供可用插件列表、可用频道列表和支持的网盘类型列表等静态信息资源。

### 架构与部署方式

PanSou MCP 服务设计为与 PanSou Go 后端服务分离，通过 HTTP API 进行通信。它本身提供以下几种部署方式：

*   **npx 部署 (TypeScript)**: 这是推荐的方式之一。MCP 服务被打包为一个 npm 包，可以通过 `npx` 命令直接运行。它会自动连接到指定的 PanSou 后端服务。
*   **Docker 部署**: 将 MCP 服务和 PanSou 后端服务打包到一个 Docker 镜像中，实现一体化部署。
*   **uvx 部署 (Python)**: 提供一个 Python 版本的 MCP 服务，可通过 `uvx` (或 `pipx`) 运行。

本文档主要介绍 **npx 部署** 方式。

## 安装与部署

### 前提条件

1.  **Node.js**: 确保您的系统已安装 Node.js (版本 >= 18.0.0)。您可以通过在终端运行 `node -v` 来检查版本。
2.  **PanSou 后端服务**: 您需要有一个正在运行的 PanSou Go 后端服务。您可以通过以下方式之一获得：
    *   从源码构建并运行。
    *   使用 Docker 运行 `ghcr.io/fish2018/pansou:latest` 镜像。
    *   使用 Docker Compose (参考主项目 README)。

### 部署步骤

由于您是基于原作者的后端服务构建的，我们假设您已经有一个运行中的 PanSou 后端服务，地址为 `http://localhost:8888` (默认地址)。

1.  **确保 PanSou 后端服务正在运行**:
    在终端中访问 `http://localhost:8888/api/health`，您应该能看到类似以下的 JSON 响应，表明服务正常：
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
    如果服务未运行，请先启动它。

2.  **通过 npx 运行 MCP 服务**:
    您可以直接使用 `npx` 命令来运行 MCP 服务，而无需显式安装。在您的项目目录下（包含 `typescript` 文件夹），运行：
    ```bash
    npx .\typescript\dist\index.js
    ```
    这将启动 MCP 服务，并默认尝试连接到 `http://localhost:8888` 的 PanSou 后端服务。

    如果您的后端服务运行在不同的地址或端口上，您需要通过环境变量指定：
    ```bash
    set PANSOU_SERVER_URL=http://your-backend-address:port
    npx .\typescript\dist\index.js
    ```
    *(在 Linux/macOS 上使用 `export PANSOU_SERVER_URL=http://...`)*

3.  **(可选) 配置 Claude Desktop**:
    要在 Claude Desktop 中使用此 MCP 服务，您需要将其添加到 Claude 的配置文件中。
    *   找到 Claude 的配置目录（通常在用户主目录下的 `.anthropic` 文件夹中）。
    *   编辑或创建 `claude_desktop_config.json` 文件。
    *   添加或修改 `mcpServers` 部分，加入您的服务配置：
        ```json
        {
          "mcpServers": {
            "pansou-search": {
              "command": "npx",
              "args": [
                "C:\\full\\path\\to\\your\\project\\typescript\\dist\\index.js"
              ],
              "env": {
                "PANSOU_SERVER_URL": "http://localhost:8888"
              }
            }
          }
        }
        ```
        请将 `C:\\full\\path\\to\\your\\project` 替换为您项目实际的完整路径。

4.  **启动 Claude Desktop**:
    重新启动 Claude Desktop，它应该会自动连接到您配置的 MCP 服务。

## 支持的参数

MCP 服务通过工具调用接收参数。以下是主要工具及其支持的参数：

### `search_netdisk` 工具

用于搜索网盘资源。

| 参数名 | 类型 | 必填 | 默认值 | 描述 |
| :--- | :--- | :--- | :--- | :--- |
| `keyword` | string | 是 | - | 搜索关键词，例如 "速度与激情"、"Python教程"。 |
| `channels` | array of string | 否 | 配置默认值 | 要搜索的 Telegram 频道列表，例如 `["tgsearchers3", "another_channel"]`。 |
| `plugins` | array of string | 否 | 配置默认值或所有插件 | 要使用的搜索插件列表，例如 `["pansearch", "panta"]`。 |
| `cloud_types` | array of string | 否 | 无过滤 | 过滤结果，仅返回指定类型的网盘链接。支持的类型有：`baidu`, `aliyun`, `quark`, `tianyi`, `uc`, `mobile`, `115`, `pikpak`, `xunlei`, `123`, `magnet`, `ed2k`, `others`。 |
| `source_type` | string | 否 | `"all"` | 数据来源类型。可选值：`"all"` (全部来源), `"tg"` (仅 Telegram), `"plugin"` (仅插件)。 |
| `force_refresh` | boolean | 否 | `false` | 是否强制刷新缓存，以获取最新数据。 |
| `result_type` | string | 否 | `"merge"` | 返回结果的类型。可选值：`"all"` (返回所有结果), `"results"` (仅返回详细结果), `"merge"` (仅返回按网盘类型分组的结果)。 |
| `concurrency` | number | 否 | 自动计算 | 并发搜索的数量。 |
| `ext_params` | object | 否 | `{}` | 传递给后端插件的自定义扩展参数，例如 `{"title_en": "Fast and Furious", "is_all": true}`。 |

### `check_service_health` 工具

用于检查后端服务健康状态。

*   **参数**: 无

### `start_backend` 工具

用于启动本地 PanSou 后端服务。

| 参数名 | 类型 | 必填 | 默认值 | 描述 |
| :--- | :--- | :--- | :--- | :--- |
| `force_restart` | boolean | 否 | `false` | 是否强制重启后端服务（即使它已经在运行）。 |

### 环境变量配置

您可以通过设置环境变量来配置 MCP 服务的行为：

| 环境变量 | 描述 | 默认值 |
| :--- | :--- | :--- |
| `PANSOU_SERVER_URL` | PanSou 后端服务的 URL 地址。 | `http://localhost:8888` |
| `REQUEST_TIMEOUT` | HTTP 请求超时时间（秒）。 | `30` |
| `MAX_RESULTS` | （内部使用，限制处理结果数量） | `100` |
| `DEFAULT_CHANNELS` | 默认搜索的 Telegram 频道列表（逗号分隔）。 | `""` (使用后端默认) |
| `DEFAULT_PLUGINS` | 默认使用的搜索插件列表（逗号分隔）。 | `""` (使用后端默认或所有) |
| `DEFAULT_CLOUD_TYPES` | 默认的网盘类型过滤器（逗号分隔）。 | `""` (无过滤) |
| `AUTO_START_BACKEND` | 是否在 MCP 服务启动时自动尝试启动后端服务。 | `true` |
| `PROJECT_ROOT_PATH` | PanSou 后端可执行文件所在的目录路径（用于自动启动）。 | 无 |
| `IDLE_TIMEOUT` | 空闲超时时间（毫秒），超过此时间无活动则可能关闭后端服务。 | `300000` (5分钟) |
| `ENABLE_IDLE_SHUTDOWN` | 是否启用空闲超时自动关闭后端服务。 | `true`