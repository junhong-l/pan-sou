package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"pansou-openwrt/internal/config"
	"pansou-openwrt/internal/plugin"
	"pansou-openwrt/internal/search"
)

// Server HTTP服务器
type Server struct {
	config        *config.Config
	httpServer    *http.Server
	searchService *search.Service
	pluginManager *plugin.Manager
}

// New 创建新服务器
func New(cfg *config.Config) (*Server, error) {
	// 创建插件管理器
	pluginMgr := plugin.NewManager(cfg)

	// 创建搜索服务
	searchSrv := search.NewService(cfg, pluginMgr)

	// 创建服务器
	srv := &Server{
		config:        cfg,
		searchService: searchSrv,
		pluginManager: pluginMgr,
	}

	return srv, nil
}

// Start 启动服务器
func (s *Server) Start() error {
	if !s.config.Server.Enabled {
		return fmt.Errorf("服务器未启用")
	}

	// 设置Gin模式
	gin.SetMode(gin.ReleaseMode)

	// 创建路由
	router := s.setupRouter()

	// 创建HTTP服务器
	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.Server.Port),
		Handler: router,
	}

	// 启动服务器
	go func() {
		log.Printf("HTTP服务器启动在端口 %d", s.config.Server.Port)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP服务器错误: %v", err)
		}
	}()

	return nil
}

// Shutdown 关闭服务器
func (s *Server) Shutdown() {
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.httpServer.Shutdown(ctx)
	}
}

// setupRouter 设置路由
func (s *Server) setupRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())
	r.Use(loggerMiddleware())

	// API路由组
	api := r.Group("/api")
	{
		// 健康检查
		api.GET("/health", s.handleHealth)

		// 搜索接口
		api.POST("/search", s.handleSearch)
		api.GET("/search", s.handleSearch)

		// 配置管理
		api.GET("/config", s.handleGetConfig)
		api.POST("/config", s.handleUpdateConfig)

		// 插件信息
		api.GET("/plugins", s.handleGetPlugins)
	}

	return r
}

// CORS中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// 日志中间件
func loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		log.Printf("[HTTP] %s %s %d %v", c.Request.Method, path, statusCode, latency)
	}
}
