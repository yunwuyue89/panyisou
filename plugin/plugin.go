package plugin

import (
	"net/http"
	"strings"
	"sync"

	"pansou/model"
)

// 全局异步插件注册表
var (
	globalRegistry     = make(map[string]AsyncSearchPlugin)
	globalRegistryLock sync.RWMutex
)

// AsyncSearchPlugin 异步搜索插件接口
type AsyncSearchPlugin interface {
	// Name 返回插件名称
	Name() string
	
	// Priority 返回插件优先级
	Priority() int
	
	// AsyncSearch 异步搜索方法
	AsyncSearch(keyword string, searchFunc func(*http.Client, string, map[string]interface{}) ([]model.SearchResult, error), mainCacheKey string, ext map[string]interface{}) ([]model.SearchResult, error)
	
	// SetMainCacheKey 设置主缓存键
	SetMainCacheKey(key string)
	
	// SetCurrentKeyword 设置当前搜索关键词（用于日志显示）
	SetCurrentKeyword(keyword string)
	
	// Search 兼容性方法（内部调用AsyncSearch）
	Search(keyword string, ext map[string]interface{}) ([]model.SearchResult, error)
	
	// SkipServiceFilter 返回是否跳过Service层的关键词过滤
	// 对于磁力搜索等需要宽泛结果的插件，应返回true
	SkipServiceFilter() bool
}

// RegisterGlobalPlugin 注册异步插件到全局注册表
func RegisterGlobalPlugin(plugin AsyncSearchPlugin) {
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

// GetRegisteredPlugins 获取所有已注册的异步插件
func GetRegisteredPlugins() []AsyncSearchPlugin {
	globalRegistryLock.RLock()
	defer globalRegistryLock.RUnlock()
	
	plugins := make([]AsyncSearchPlugin, 0, len(globalRegistry))
	for _, plugin := range globalRegistry {
		plugins = append(plugins, plugin)
	}
	
	return plugins
}

// GetPluginByName 根据名称获取已注册的插件
func GetPluginByName(name string) (AsyncSearchPlugin, bool) {
	globalRegistryLock.RLock()
	defer globalRegistryLock.RUnlock()
	
	plugin, exists := globalRegistry[name]
	return plugin, exists
}

// PluginManager 异步插件管理器
type PluginManager struct {
	plugins []AsyncSearchPlugin
}

// NewPluginManager 创建新的异步插件管理器
func NewPluginManager() *PluginManager {
	return &PluginManager{
		plugins: make([]AsyncSearchPlugin, 0),
	}
}

// RegisterPlugin 注册异步插件
func (pm *PluginManager) RegisterPlugin(plugin AsyncSearchPlugin) {
	pm.plugins = append(pm.plugins, plugin)
}

// RegisterAllGlobalPlugins 注册所有全局异步插件
func (pm *PluginManager) RegisterAllGlobalPlugins() {
	allPlugins := GetRegisteredPlugins()
	for _, plugin := range allPlugins {
		pm.RegisterPlugin(plugin)
	}
}

// RegisterGlobalPluginsWithFilter 根据过滤器注册全局异步插件
// enabledPlugins: nil表示未设置（不启用任何插件），空切片表示设置为空（不启用任何插件），具体列表表示启用指定插件
func (pm *PluginManager) RegisterGlobalPluginsWithFilter(enabledPlugins []string) {
	allPlugins := GetRegisteredPlugins()
	
	// nil 表示未设置环境变量，不启用任何插件
	if enabledPlugins == nil {
		return
	}
	
	// 空切片表示设置为空字符串，也不启用任何插件
	if len(enabledPlugins) == 0 {
		return
	}
	
	// 创建启用插件名称的映射表，用于快速查找
	enabledMap := make(map[string]bool)
	for _, name := range enabledPlugins {
		enabledMap[name] = true
	}
	
	// 只注册在启用列表中的插件
	for _, plugin := range allPlugins {
		if enabledMap[plugin.Name()] {
			pm.RegisterPlugin(plugin)
		}
	}
}

// GetPlugins 获取所有注册的异步插件
func (pm *PluginManager) GetPlugins() []AsyncSearchPlugin {
	return pm.plugins
}

// FilterResultsByKeyword 根据关键词过滤搜索结果的全局辅助函数
func FilterResultsByKeyword(results []model.SearchResult, keyword string) []model.SearchResult {
	if keyword == "" {
		return results
	}
	
	// 预估过滤后会保留80%的结果
	filteredResults := make([]model.SearchResult, 0, len(results)*8/10)

	// 将关键词转为小写，用于不区分大小写的比较
	lowerKeyword := strings.ToLower(keyword)

	// 将关键词按空格分割，用于支持多关键词搜索
	keywords := strings.Fields(lowerKeyword)

	for _, result := range results {
		// 将标题和内容转为小写
		lowerTitle := strings.ToLower(result.Title)
		lowerContent := strings.ToLower(result.Content)

		// 检查每个关键词是否在标题或内容中
		matched := true
		for _, kw := range keywords {
			// 对于所有关键词，检查是否在标题或内容中
			if !strings.Contains(lowerTitle, kw) && !strings.Contains(lowerContent, kw) {
				matched = false
				break
			}
		}

		if matched {
			filteredResults = append(filteredResults, result)
		}
	}

	return filteredResults
} 