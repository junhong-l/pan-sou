package plugin

import (
	"net/http"
	"time"

	"pansou-openwrt/internal/config"
	"pansou-openwrt/internal/model"
	"pansou-openwrt/internal/plugin/plugins"
)

// Plugin 插件接口
type Plugin interface {
	Name() string
	DisplayName() string
	Description() string
	Priority() int
	Search(keyword string, ext map[string]interface{}) ([]model.SearchResult, error)
}

// Manager 插件管理器
type Manager struct {
	config  *config.Config
	plugins map[string]Plugin
	client  *http.Client
}

// NewManager 创建插件管理器
func NewManager(cfg *config.Config) *Manager {
	m := &Manager{
		config:  cfg,
		plugins: make(map[string]Plugin),
		client: &http.Client{
			Timeout: time.Duration(cfg.Search.Timeout) * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}

	// 注册所有插件
	m.registerPlugins()

	return m
}

// registerPlugins 注册所有插件（不需要登录的插件）
func (m *Manager) registerPlugins() {
	// 注册已实现的插件
	m.Register(plugins.NewMiaosoPlugin(m.client))
	m.Register(plugins.NewXysPlugin(m.client))
	m.Register(plugins.NewJutoushePlugin(m.client))
	
	// TODO: 继续添加其他插件
	// m.Register(plugins.NewAlupanPlugin(m.client))
	// m.Register(plugins.NewMikuclubPlugin(m.client))
	// m.Register(plugins.NewYpfxwPlugin(m.client))
	// m.Register(plugins.NewKkmaoPlugin(m.client))
	// m.Register(plugins.NewClxiongPlugin(m.client))
	// m.Register(plugins.NewAshPlugin(m.client))
	// m.Register(plugins.NewQingyingPlugin(m.client))
	// m.Register(plugins.NewMeitizyPlugin(m.client))
	// m.Register(plugins.NewWujiPlugin(m.client))
	// m.Register(plugins.NewDyyjPlugin(m.client))
	// m.Register(plugins.NewLabiPlugin(m.client))
	// m.Register(plugins.NewZxzjPlugin(m.client))
	// m.Register(plugins.NewDdysPlugin(m.client))
	// m.Register(plugins.NewLou1Plugin(m.client))
	// m.Register(plugins.NewPanyqPlugin(m.client))
}

// Register 注册插件
func (m *Manager) Register(p Plugin) {
	m.plugins[p.Name()] = p
}

// GetPlugin 获取指定插件
func (m *Manager) GetPlugin(name string) (Plugin, bool) {
	p, ok := m.plugins[name]
	return p, ok
}

// GetPlugins 获取所有插件
func (m *Manager) GetPlugins() []Plugin {
	plugins := make([]Plugin, 0, len(m.plugins))
	for _, p := range m.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

// GetEnabledPlugins 获取启用的插件
func (m *Manager) GetEnabledPlugins() []Plugin {
	if !m.config.Plugins.Enabled {
		return []Plugin{}
	}

	plugins := make([]Plugin, 0)
	for name, settings := range m.config.Plugins.List {
		if settings.Enabled {
			if p, ok := m.plugins[name]; ok {
				plugins = append(plugins, p)
			}
		}
	}
	return plugins
}
