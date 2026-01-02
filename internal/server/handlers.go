package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"pansou-openwrt/internal/model"
)

// handleHealth 健康检查
func (s *Server) handleHealth(c *gin.Context) {
	plugins := s.pluginManager.GetPlugins()
	pluginNames := make([]string, 0, len(plugins))
	for _, p := range plugins {
		pluginNames = append(pluginNames, p.Name())
	}

	resp := model.HealthResponse{
		Status:          "ok",
		PluginsEnabled:  s.config.Plugins.Enabled,
		PluginCount:     len(plugins),
		PluginNames:     pluginNames,
		ChannelsCount:   len(s.config.Telegram.Channels),
		Channels:        s.config.Telegram.Channels,
		TelegramEnabled: s.config.Telegram.Enabled,
	}

	c.JSON(http.StatusOK, resp)
}

// handleSearch 搜索处理
func (s *Server) handleSearch(c *gin.Context) {
	var req model.SearchRequest

	// 根据请求方法解析参数
	if c.Request.Method == "GET" {
		req.Keyword = c.Query("kw")
		req.SourceType = c.DefaultQuery("src", "all")
		req.ResultType = c.DefaultQuery("res", "merge")
		req.ForceRefresh = c.Query("refresh") == "true"
		
		// 解析数组参数
		if channels := c.QueryArray("channels"); len(channels) > 0 {
			req.Channels = channels
		}
		if plugins := c.QueryArray("plugins"); len(plugins) > 0 {
			req.Plugins = plugins
		}
		if cloudTypes := c.QueryArray("cloud_types"); len(cloudTypes) > 0 {
			req.CloudTypes = cloudTypes
		}
	} else {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, model.ErrorResponse{
				Code:    400,
				Message: "无效的请求参数: " + err.Error(),
			})
			return
		}
	}

	// 验证参数
	if req.Keyword == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "搜索关键词不能为空",
		})
		return
	}

	// 设置默认值
	if req.SourceType == "" {
		req.SourceType = "all"
	}
	if req.ResultType == "" {
		req.ResultType = "merge"
	}

	// 执行搜索
	startTime := time.Now()
	result, err := s.searchService.Search(&req)
	searchTime := time.Since(startTime).Seconds()

	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "搜索失败: " + err.Error(),
		})
		return
	}

	result.SearchTime = searchTime
	c.JSON(http.StatusOK, result)
}

// handleGetConfig 获取配置
func (s *Server) handleGetConfig(c *gin.Context) {
	resp := model.ConfigResponse{
		Server: map[string]interface{}{
			"port":      s.config.Server.Port,
			"enabled":   s.config.Server.Enabled,
			"autostart": s.config.Server.Autostart,
		},
		Search: map[string]interface{}{
			"concurrency": s.config.Search.Concurrency,
			"timeout":     s.config.Search.Timeout,
			"cache_ttl":   s.config.Search.CacheTTL,
		},
		Telegram: map[string]interface{}{
			"enabled":       s.config.Telegram.Enabled,
			"channels":      s.config.Telegram.Channels,
			"check_timeout": s.config.Telegram.CheckTimeout,
		},
		Plugins: map[string]interface{}{
			"enabled": s.config.Plugins.Enabled,
			"list":    s.config.Plugins.List,
		},
		CloudTypes: map[string]interface{}{
			"enabled": s.config.CloudTypes.Enabled,
		},
	}

	c.JSON(http.StatusOK, resp)
}

// handleUpdateConfig 更新配置
func (s *Server) handleUpdateConfig(c *gin.Context) {
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "无效的请求参数: " + err.Error(),
		})
		return
	}

	// TODO: 实现配置更新逻辑
	// 这里需要更新配置并保存到文件

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "配置更新成功",
	})
}

// handleGetPlugins 获取插件列表
func (s *Server) handleGetPlugins(c *gin.Context) {
	plugins := s.pluginManager.GetPlugins()
	pluginList := make([]map[string]interface{}, 0, len(plugins))

	for _, p := range plugins {
		pluginList = append(pluginList, map[string]interface{}{
			"name":         p.Name(),
			"display_name": p.DisplayName(),
			"description":  p.Description(),
			"priority":     p.Priority(),
			"enabled":      true, // TODO: 从配置读取
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"total":   len(pluginList),
		"plugins": pluginList,
	})
}
