package plugin

import (
	"pansou/model"
	"sync"
)

// 全局插件注册表
var (
	globalRegistry     = make(map[string]SearchPlugin)
	globalRegistryLock sync.RWMutex
)

// SearchPlugin 搜索插件接口
type SearchPlugin interface {
	// Name 返回插件名称
	Name() string
	
	// Search 执行搜索并返回结果
	Search(keyword string) ([]model.SearchResult, error)
	
	// Priority 返回插件优先级（可选，用于控制结果排序）
	Priority() int
}

// RegisterGlobalPlugin 注册插件到全局注册表
// 这个函数应该在每个插件的init函数中被调用
func RegisterGlobalPlugin(plugin SearchPlugin) {
	if plugin == nil {
		return
	}
	
	globalRegistryLock.Lock()
	defer globalRegistryLock.Unlock()
	
	name := plugin.Name()
	if name == "" {
		return
	}
	
	globalRegistry[name] = plugin
}

// GetRegisteredPlugins 获取所有已注册的插件
func GetRegisteredPlugins() []SearchPlugin {
	globalRegistryLock.RLock()
	defer globalRegistryLock.RUnlock()
	
	plugins := make([]SearchPlugin, 0, len(globalRegistry))
	for _, plugin := range globalRegistry {
		plugins = append(plugins, plugin)
	}
	
	return plugins
}

// PluginManager 插件管理器
type PluginManager struct {
	plugins []SearchPlugin
}

// NewPluginManager 创建新的插件管理器
func NewPluginManager() *PluginManager {
	return &PluginManager{
		plugins: make([]SearchPlugin, 0),
	}
}

// RegisterPlugin 注册插件
func (pm *PluginManager) RegisterPlugin(plugin SearchPlugin) {
	pm.plugins = append(pm.plugins, plugin)
}

// RegisterAllGlobalPlugins 注册所有全局插件
func (pm *PluginManager) RegisterAllGlobalPlugins() {
	for _, plugin := range GetRegisteredPlugins() {
		pm.RegisterPlugin(plugin)
	}
}

// GetPlugins 获取所有注册的插件
func (pm *PluginManager) GetPlugins() []SearchPlugin {
	return pm.plugins
} 