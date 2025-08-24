#!/usr/bin/env node

import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import {
  CallToolRequestSchema,
  ErrorCode,
  ListResourcesRequestSchema,
  ListToolsRequestSchema,
  McpError,
  ReadResourceRequestSchema,
} from '@modelcontextprotocol/sdk/types.js';

import { loadConfig } from './utils/config.js';
import { HttpClient } from './utils/http-client.js';
import { searchTool, executeSearchTool } from './tools/search.js';
import { healthTool, executeHealthTool } from './tools/health.js';

/**
 * PanSou MCP服务器
 */
class PanSouMCPServer {
  private server: Server;
  private httpClient: HttpClient;
  private config: any;

  constructor() {
    this.server = new Server(
      {
        name: 'pansou-mcp-server',
        version: '1.0.0',
      },
      {
        capabilities: {
          tools: {},
          resources: {},
        },
      }
    );

    // 加载配置
    this.config = loadConfig();
    this.httpClient = new HttpClient(this.config);

    this.setupHandlers();
  }

  /**
   * 设置请求处理器
   */
  private setupHandlers(): void {
    // 工具列表处理器
    this.server.setRequestHandler(ListToolsRequestSchema, async () => {
      return {
        tools: [searchTool, healthTool],
      };
    });

    // 工具调用处理器
    this.server.setRequestHandler(CallToolRequestSchema, async (request) => {
      const { name, arguments: args } = request.params;

      try {
        switch (name) {
          case 'search_netdisk':
            const searchResult = await executeSearchTool(args, this.httpClient);
            return {
              content: [
                {
                  type: 'text',
                  text: searchResult,
                },
              ],
            };

          case 'check_service_health':
            const healthResult = await executeHealthTool(args, this.httpClient);
            return {
              content: [
                {
                  type: 'text',
                  text: healthResult,
                },
              ],
            };

          default:
            throw new McpError(
              ErrorCode.MethodNotFound,
              `未知工具: ${name}`
            );
        }
      } catch (error) {
        if (error instanceof McpError) {
          throw error;
        }

        throw new McpError(
          ErrorCode.InternalError,
          `工具执行失败: ${error instanceof Error ? error.message : String(error)}`
        );
      }
    });

    // 资源列表处理器
    this.server.setRequestHandler(ListResourcesRequestSchema, async () => {
      return {
        resources: [
          {
            uri: 'pansou://plugins',
            name: '可用插件列表',
            description: '获取当前可用的搜索插件列表',
            mimeType: 'application/json',
          },
          {
            uri: 'pansou://channels',
            name: '可用频道列表',
            description: '获取当前可用的TG频道列表',
            mimeType: 'application/json',
          },
          {
            uri: 'pansou://cloud-types',
            name: '支持的网盘类型',
            description: '获取支持的网盘类型列表',
            mimeType: 'application/json',
          },
        ],
      };
    });

    // 资源读取处理器
    this.server.setRequestHandler(ReadResourceRequestSchema, async (request) => {
      const { uri } = request.params;

      try {
        switch (uri) {
          case 'pansou://plugins':
            return await this.getPluginsResource();

          case 'pansou://channels':
            return await this.getChannelsResource();

          case 'pansou://cloud-types':
            return await this.getCloudTypesResource();

          default:
            throw new McpError(
              ErrorCode.InvalidRequest,
              `未知资源URI: ${uri}`
            );
        }
      } catch (error) {
        if (error instanceof McpError) {
          throw error;
        }

        throw new McpError(
          ErrorCode.InternalError,
          `资源读取失败: ${error instanceof Error ? error.message : String(error)}`
        );
      }
    });
  }

  /**
   * 获取插件资源
   */
  private async getPluginsResource() {
    try {
      const healthData = await this.httpClient.checkHealth();
      
      const plugins = {
        enabled: healthData.plugins_enabled || false,
        count: healthData.plugin_count || 0,
        list: healthData.plugins || [],
      };

      return {
        contents: [
          {
            uri: 'pansou://plugins',
            mimeType: 'application/json',
            text: JSON.stringify(plugins, null, 2),
          },
        ],
      };
    } catch (error) {
      throw new McpError(
        ErrorCode.InternalError,
        `获取插件信息失败: ${error instanceof Error ? error.message : String(error)}`
      );
    }
  }

  /**
   * 获取频道资源
   */
  private async getChannelsResource() {
    try {
      const healthData = await this.httpClient.checkHealth();
      
      const channels = {
        count: healthData.channels_count || 0,
        list: healthData.channels || [],
      };

      return {
        contents: [
          {
            uri: 'pansou://channels',
            mimeType: 'application/json',
            text: JSON.stringify(channels, null, 2),
          },
        ],
      };
    } catch (error) {
      throw new McpError(
        ErrorCode.InternalError,
        `获取频道信息失败: ${error instanceof Error ? error.message : String(error)}`
      );
    }
  }

  /**
   * 获取网盘类型资源
   */
  private async getCloudTypesResource() {
    const cloudTypes = {
      supported: [
        'baidu',      // 百度网盘
        'ali',        // 阿里云盘
        'quark',      // 夸克网盘
        'uc',         // UC网盘
        '115',        // 115网盘
        'lanzou',     // 蓝奏云
        'tianyi',     // 天翼云盘
        'weiyun',     // 微云
        'onedrive',   // OneDrive
        'googledrive',// Google Drive
        'mega',       // MEGA
        'other'       // 其他
      ],
      description: {
        'baidu': '百度网盘',
        'ali': '阿里云盘',
        'quark': '夸克网盘',
        'uc': 'UC网盘',
        '115': '115网盘',
        'lanzou': '蓝奏云',
        'tianyi': '天翼云盘',
        'weiyun': '微云',
        'onedrive': 'OneDrive',
        'googledrive': 'Google Drive',
        'mega': 'MEGA',
        'other': '其他网盘'
      }
    };

    return {
      contents: [
        {
          uri: 'pansou://cloud-types',
          mimeType: 'application/json',
          text: JSON.stringify(cloudTypes, null, 2),
        },
      ],
    };
  }

  /**
   * 启动服务器
   */
  public async start(): Promise<void> {
    const transport = new StdioServerTransport();
    await this.server.connect(transport);
    
    // 输出启动信息到stderr，避免干扰MCP通信
    console.error('🚀 PanSou MCP服务器已启动');
    console.error(`📡 服务地址: ${this.config.serverUrl}`);
    console.error(`⏱️  请求超时: ${this.config.requestTimeout}ms`);
    console.error(`📊 最大结果数: ${this.config.maxResults}`);
  }
}

/**
 * 主函数
 */
async function main(): Promise<void> {
  try {
    const server = new PanSouMCPServer();
    await server.start();
  } catch (error) {
    console.error('❌ 服务器启动失败:', error);
    process.exit(1);
  }
}

// 处理未捕获的异常
process.on('uncaughtException', (error) => {
  console.error('❌ 未捕获的异常:', error);
  process.exit(1);
});

process.on('unhandledRejection', (reason, promise) => {
  console.error('❌ 未处理的Promise拒绝:', reason);
  process.exit(1);
});

// 启动服务器
if (import.meta.url === `file://${process.argv[1]}` || process.argv[1].endsWith('index.js')) {
  main();
}

export { PanSouMCPServer };