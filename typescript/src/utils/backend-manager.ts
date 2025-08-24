import { spawn, ChildProcess } from 'child_process';
import { promises as fs } from 'fs';
import path from 'path';
import { HttpClient } from './http-client.js';
import { Config } from './config.js';
import { ActivityMonitor } from './activity-monitor.js';

/**
 * 后端服务管理器
 * 负责自动启动、停止和监控PanSou Go后端服务
 */
export class BackendManager {
  private process: ChildProcess | null = null;
  private config: Config;
  private httpClient: HttpClient;
  private shutdownTimeout: NodeJS.Timeout | null = null;
  private isShuttingDown = false;
  private readonly SHUTDOWN_DELAY = 5000; // 5秒延迟关闭
  private readonly STARTUP_TIMEOUT = 30000; // 30秒启动超时
  private readonly HEALTH_CHECK_INTERVAL = 1000; // 1秒健康检查间隔
  private activityMonitor: ActivityMonitor | null = null;

  constructor(config: Config, httpClient: HttpClient) {
    this.config = config;
    this.httpClient = httpClient;
    
    // 初始化活动监控器
    if (this.config.enableIdleShutdown) {
      this.activityMonitor = new ActivityMonitor(
        this.config.idleTimeout,
        this.config.enableIdleShutdown
      );
      
      // 设置空闲监控回调
      this.activityMonitor.setOnIdleCallback(async () => {
        console.error('⏰ 检测到空闲超时，自动关闭后端服务');
        await this.stopBackend();
        // 退出整个进程
        process.exit(0);
      });
      console.error(`⏱️  空闲监控已启用，超时时间: ${this.config.idleTimeout / 1000} 秒`);
    }
  }

  /**
   * 检查后端服务是否正在运行
   */
  async isBackendRunning(): Promise<boolean> {
    try {
      return await this.httpClient.testConnection();
    } catch (error) {
      return false;
    }
  }

  /**
   * 查找Go可执行文件路径
   */
  private async findGoExecutable(): Promise<string | null> {
    // 优先使用配置中的项目根目录
    const configProjectRoot = this.config.projectRootPath;
    
    const possiblePaths: string[] = [];
    
    // 如果配置了项目根目录，直接在该目录下查找
    if (configProjectRoot) {
      possiblePaths.push(
        path.join(configProjectRoot, 'pansou.exe'),
        path.join(configProjectRoot, 'main.exe')
      );
    } else {
      // 仅在没有配置项目根目录时才使用备用路径
      possiblePaths.push(
        // 当前工作目录
        path.join(process.cwd(), 'pansou.exe'),
        path.join(process.cwd(), 'main.exe'),
        // 上级目录（如果MCP在子目录中）
        path.join(process.cwd(), '..', 'pansou.exe'),
        path.join(process.cwd(), '..', 'main.exe')
      );
    }

    console.error('🔍 查找后端可执行文件...');
    if (configProjectRoot) {
      console.error(`📂 使用配置的项目根目录: ${configProjectRoot}`);
    } else {
      console.error(`📂 当前工作目录: ${process.cwd()}`);
    }
    
    for (const execPath of possiblePaths) {
      try {
        await fs.access(execPath);
        console.error(`✅ 找到可执行文件: ${execPath}`);
        return execPath;
      } catch {
        // 静默跳过未找到的路径
      }
    }

    console.error('❌ 未找到可执行文件');
    return null;
  }

  /**
   * 启动后端服务
   */
  async startBackend(): Promise<boolean> {
    if (this.process) {
      console.error('⚠️  后端服务已在运行中');
      return true;
    }

    // 首先检查是否已有服务在运行
    this.httpClient.setSilentMode(true);
    const isRunning = await this.isBackendRunning();
    this.httpClient.setSilentMode(false);
    
    if (isRunning) {
      console.error('✅ 检测到后端服务已在运行');
      return true;
    }

    // 查找Go可执行文件
    const execPath = await this.findGoExecutable();
    if (!execPath) {
      console.error('❌ 未找到PanSou后端可执行文件');
      console.error('请确保在项目根目录下存在以下文件之一：');
      console.error('  - pansou.exe / pansou');
      console.error('  - main.exe / main');
      return false;
    }

    console.error(`🚀 启动后端服务: ${execPath}`);

    try {
      // 启动Go服务
      this.process = spawn(execPath, [], {
        cwd: path.dirname(execPath),
        stdio: ['ignore', 'pipe', 'pipe'],
        detached: false,
        windowsHide: true
      });

      // 监听进程事件
      this.process.on('error', (error) => {
        console.error('❌ 后端服务启动失败:', error.message);
        console.error('错误详情:', error);
        this.process = null;
      });

      this.process.on('exit', (code, signal) => {
        if (!this.isShuttingDown) {
          console.error(`⚠️  后端服务意外退出 (code: ${code}, signal: ${signal})`);
        }
        this.process = null;
      });

      // 添加进程启动确认
      console.error(`📋 进程PID: ${this.process.pid}`);
      console.error(`📂 工作目录: ${path.dirname(execPath)}`);
      console.error(`⚙️  启动参数: ${execPath}`);
      
      // 给进程一点时间启动
      await new Promise(resolve => setTimeout(resolve, 1000));

      // 捕获输出（用于调试）
      if (this.process.stdout) {
        this.process.stdout.on('data', (data) => {
          console.error('Backend stdout:', data.toString().trim());
        });
      }

      if (this.process.stderr) {
        this.process.stderr.on('data', (data) => {
          console.error('Backend stderr:', data.toString().trim());
        });
      }

      // 等待服务启动
      const started = await this.waitForBackendReady();
      if (started) {
        console.error('✅ 后端服务启动成功');
        
        // 空闲监控已在构造函数中设置
        
        return true;
      } else {
        console.error('❌ 后端服务启动超时');
        await this.stopBackend();
        return false;
      }
    } catch (error) {
      console.error('❌ 启动后端服务时发生错误:', error);
      return false;
    }
  }

  /**
   * 等待后端服务就绪
   */
  private async waitForBackendReady(): Promise<boolean> {
    const startTime = Date.now();
    
    // 在等待期间启用静默模式，避免输出网络错误
    const originalSilentMode = this.httpClient.isSilentMode();
    this.httpClient.setSilentMode(true);
    
    try {
      while (Date.now() - startTime < this.STARTUP_TIMEOUT) {
        if (await this.isBackendRunning()) {
          return true;
        }
        
        // 检查进程是否还在运行
        if (!this.process || this.process.killed) {
          return false;
        }
        
        // 等待一段时间后重试
        await new Promise(resolve => setTimeout(resolve, this.HEALTH_CHECK_INTERVAL));
      }
      
      return false;
    } finally {
      // 恢复原始静默模式状态
      this.httpClient.setSilentMode(originalSilentMode);
    }
  }

  /**
   * 停止后端服务
   */
  async stopBackend(): Promise<void> {
    if (!this.process) {
      return;
    }

    console.error('🛑 正在停止后端服务...');
    this.isShuttingDown = true;

    try {
      // 尝试优雅关闭
      this.process.kill('SIGTERM');
      
      // 等待进程退出
      await new Promise<void>((resolve) => {
        if (!this.process) {
          resolve();
          return;
        }

        const timeout = setTimeout(() => {
          // 强制杀死进程
          if (this.process && !this.process.killed) {
            console.error('⚠️  强制终止后端服务');
            this.process.kill('SIGKILL');
          }
          resolve();
        }, 5000);

        this.process.on('exit', () => {
          clearTimeout(timeout);
          resolve();
        });
      });

      console.error('✅ 后端服务已停止');
    } catch (error) {
      console.error('❌ 停止后端服务时发生错误:', error);
    } finally {
      this.process = null;
      this.isShuttingDown = false;
    }
  }

  /**
   * 延迟停止后端服务
   */
  scheduleShutdown(): void {
    if (this.shutdownTimeout) {
      clearTimeout(this.shutdownTimeout);
    }

    console.error(`⏰ 将在 ${this.SHUTDOWN_DELAY / 1000} 秒后关闭后端服务`);
    
    this.shutdownTimeout = setTimeout(async () => {
      await this.stopBackend();
      this.shutdownTimeout = null;
    }, this.SHUTDOWN_DELAY);
  }

  /**
   * 取消计划的关闭
   */
  cancelShutdown(): void {
    if (this.shutdownTimeout) {
      clearTimeout(this.shutdownTimeout);
      this.shutdownTimeout = null;
      console.error('⏸️  取消后端服务关闭计划');
    }
  }

  /**
   * 获取后端服务状态
   */
  getStatus(): {
    processRunning: boolean;
    serviceReachable: boolean;
    pid?: number;
  } {
    return {
      processRunning: this.process !== null && !this.process.killed,
      serviceReachable: false, // 需要异步检查
      pid: this.process?.pid
    };
  }

  /**
   * 记录活动（重置空闲计时器）
   */
  recordActivity(): void {
    if (this.activityMonitor) {
      this.activityMonitor.recordActivity();
    }
  }

  /**
   * 获取活动监控状态
   */
  getActivityStatus(): any {
    return this.activityMonitor ? this.activityMonitor.getStatus() : null;
  }

  /**
   * 清理资源
   */
  async cleanup(): Promise<void> {
    this.cancelShutdown();
    if (this.activityMonitor) {
      this.activityMonitor.stop();
      this.activityMonitor = null;
    }
    await this.stopBackend();
  }
}

/**
 * 创建后端管理器实例
 */
export function createBackendManager(config: Config, httpClient: HttpClient): BackendManager {
  return new BackendManager(config, httpClient);
}