package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Search    SearchConfig    `yaml:"search"`
	Telegram  TelegramConfig  `yaml:"telegram"`
	Plugins   PluginsConfig   `yaml:"plugins"`
	CloudTypes CloudTypesConfig `yaml:"cloud_types"`
	Logging   LoggingConfig   `yaml:"logging"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port      int  `yaml:"port"`
	Enabled   bool `yaml:"enabled"`
	Autostart bool `yaml:"autostart"`
}

// SearchConfig 搜索配置
type SearchConfig struct {
	Concurrency int `yaml:"concurrency"`
	Timeout     int `yaml:"timeout"`
	CacheTTL    int `yaml:"cache_ttl"`
}

// TelegramConfig Telegram配置
type TelegramConfig struct {
	Enabled      bool     `yaml:"enabled"`
	Channels     []string `yaml:"channels"`
	CheckTimeout int      `yaml:"check_timeout"`
	Proxy        string   `yaml:"proxy"`
}

// PluginsConfig 插件配置
type PluginsConfig struct {
	Enabled bool                      `yaml:"enabled"`
	List    map[string]PluginSettings `yaml:"list"`
}

// PluginSettings 单个插件配置
type PluginSettings struct {
	Enabled  bool `yaml:"enabled"`
	Priority int  `yaml:"priority"`
}

// CloudTypesConfig 网盘类型配置
type CloudTypesConfig struct {
	Enabled []string `yaml:"enabled"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

// Load 加载配置文件
func Load(path string) (*Config, error) {
	// 读取文件
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("无效的端口号: %d", c.Server.Port)
	}

	if c.Search.Timeout <= 0 {
		c.Search.Timeout = 30
	}

	if c.Search.CacheTTL <= 0 {
		c.Search.CacheTTL = 60
	}

	return nil
}

// Save 保存配置文件
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// GetEnabledPlugins 获取启用的插件列表
func (c *Config) GetEnabledPlugins() []string {
	if !c.Plugins.Enabled {
		return []string{}
	}

	plugins := make([]string, 0)
	for name, settings := range c.Plugins.List {
		if settings.Enabled {
			plugins = append(plugins, name)
		}
	}
	return plugins
}
