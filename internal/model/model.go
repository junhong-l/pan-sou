package model

import "time"

// SearchRequest 搜索请求
type SearchRequest struct {
	Keyword     string                 `json:"keyword" binding:"required"`
	Channels    []string               `json:"channels"`
	Plugins     []string               `json:"plugins"`
	CloudTypes  []string               `json:"cloud_types"`
	Concurrency int                    `json:"concurrency"`
	ForceRefresh bool                  `json:"force_refresh"`
	SourceType  string                 `json:"source_type"` // all, tg, plugin
	ResultType  string                 `json:"result_type"` // all, results, merge
	Ext         map[string]interface{} `json:"ext"`
}

// SearchResponse 搜索响应
type SearchResponse struct {
	Total          int             `json:"total"`
	Results        []SearchResult  `json:"results,omitempty"`
	MergedByType   map[string][]SearchResult `json:"merged_by_type,omitempty"`
	SearchTime     float64         `json:"search_time"`
	CacheHit       bool            `json:"cache_hit"`
}

// SearchResult 搜索结果
type SearchResult struct {
	UniqueID    string    `json:"unique_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Links       []Link    `json:"links"`
	Source      string    `json:"source"`       // 来源: plugin:xxx 或 tg:channel
	Channel     string    `json:"channel,omitempty"`
	PublishTime time.Time `json:"publish_time"`
	Datetime    string    `json:"datetime"`
}

// Link 链接信息
type Link struct {
	Type     string `json:"type"`     // baidu, aliyun, quark, magnet, etc.
	URL      string `json:"url"`
	Password string `json:"password,omitempty"`
	Size     string `json:"size,omitempty"`
}

// PluginSearchResult 插件搜索结果（带IsFinal标记）
type PluginSearchResult struct {
	Results []SearchResult `json:"results"`
	IsFinal bool           `json:"is_final"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status         string   `json:"status"`
	PluginsEnabled bool     `json:"plugins_enabled"`
	PluginCount    int      `json:"plugin_count"`
	PluginNames    []string `json:"plugin_names"`
	ChannelsCount  int      `json:"channels_count"`
	Channels       []string `json:"channels"`
	TelegramEnabled bool    `json:"telegram_enabled"`
}

// ConfigResponse 配置响应
type ConfigResponse struct {
	Server    interface{} `json:"server"`
	Search    interface{} `json:"search"`
	Telegram  interface{} `json:"telegram"`
	Plugins   interface{} `json:"plugins"`
	CloudTypes interface{} `json:"cloud_types"`
}
