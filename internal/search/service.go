package search

import (
	"fmt"
	"log"
	"sync"
	"time"

	"pansou-openwrt/internal/config"
	"pansou-openwrt/internal/model"
	"pansou-openwrt/internal/plugin"
	"pansou-openwrt/internal/telegram"
)

// Service 搜索服务
type Service struct {
	config        *config.Config
	pluginManager *plugin.Manager
	tgClient      *telegram.Client
	cache         *cache
}

// NewService 创建搜索服务
func NewService(cfg *config.Config, pm *plugin.Manager) *Service {
	// 创建Telegram客户端
	var tgClient *telegram.Client
	if cfg.Telegram.Enabled {
		tgClient = telegram.NewClient(&cfg.Telegram)
	}

	return &Service{
		config:        cfg,
		pluginManager: pm,
		tgClient:      tgClient,
		cache:         newCache(time.Duration(cfg.Search.CacheTTL) * time.Minute),
	}
}

// Search 执行搜索
func (s *Service) Search(req *model.SearchRequest) (*model.SearchResponse, error) {
	// 检查缓存
	cacheKey := s.buildCacheKey(req)
	if !req.ForceRefresh {
		if cached, ok := s.cache.Get(cacheKey); ok {
			log.Printf("缓存命中: %s", cacheKey)
			result := cached.(*model.SearchResponse)
			result.CacheHit = true
			return result, nil
		}
	}

	// 执行搜索
	allResults := make([]model.SearchResult, 0)
	var wg sync.WaitGroup
	var mu sync.Mutex
	
	concurrency := req.Concurrency
	if concurrency <= 0 {
		concurrency = s.config.Search.Concurrency
	}
	
	// 使用信号量控制并发
	sem := make(chan struct{}, concurrency)

	// 插件搜索
	if req.SourceType == "all" || req.SourceType == "plugin" {
		plugins := s.getPluginsForSearch(req.Plugins)
		
		for _, p := range plugins {
			wg.Add(1)
			sem <- struct{}{} // 获取信号量
			
			go func(plug plugin.Plugin) {
				defer wg.Done()
				defer func() { <-sem }() // 释放信号量
				
				results, err := plug.Search(req.Keyword, req.Ext)
				if err != nil {
					log.Printf("插件 %s 搜索失败: %v", plug.Name(), err)
					return
				}
				
				mu.Lock()
				allResults = append(allResults, results...)
				mu.Unlock()
			}(p)
		}
	}

	// Telegram搜索
	if req.SourceType == "all" || req.SourceType == "tg" {
		if s.config.Telegram.Enabled && s.tgClient != nil && s.tgClient.IsAvailable() {
			wg.Add(1)
			sem <- struct{}{}
			
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				
				results, err := s.searchTelegram(req)
				if err != nil {
					log.Printf("Telegram搜索失败: %v", err)
					return
				}
				
				mu.Lock()
				allResults = append(allResults, results...)
				mu.Unlock()
			}()
		} else if s.config.Telegram.Enabled {
			log.Println("[TG] Telegram未启用或网络不可达，跳过TG搜索")
		}
	}

	wg.Wait()

	// 过滤网盘类型
	if len(req.CloudTypes) > 0 {
		allResults = s.filterByCloudType(allResults, req.CloudTypes)
	}

	// 构建响应
	resp := &model.SearchResponse{
		Total:    len(allResults),
		CacheHit: false,
	}

	// 根据结果类型返回不同格式
	switch req.ResultType {
	case "all":
		resp.Results = allResults
		resp.MergedByType = s.mergeByType(allResults)
	case "results":
		resp.Results = allResults
	case "merge":
		resp.MergedByType = s.mergeByType(allResults)
	default:
		resp.MergedByType = s.mergeByType(allResults)
	}

	// 缓存结果
	s.cache.Set(cacheKey, resp)

	return resp, nil
}

// getPluginsForSearch 获取用于搜索的插件
func (s *Service) getPluginsForSearch(requestedPlugins []string) []plugin.Plugin {
	if len(requestedPlugins) > 0 {
		// 使用指定的插件
		plugins := make([]plugin.Plugin, 0)
		for _, name := range requestedPlugins {
			if p, ok := s.pluginManager.GetPlugin(name); ok {
				plugins = append(plugins, p)
			}
		}
		return plugins
	}
	
	// 使用所有启用的插件
	return s.pluginManager.GetEnabledPlugins()
}

// searchTelegram 搜索Telegram
func (s *Service) searchTelegram(req *model.SearchRequest) ([]model.SearchResult, error) {
	if s.tgClient == nil {
		return []model.SearchResult{}, nil
	}

	channels := req.Channels
	if len(channels) == 0 {
		channels = s.config.Telegram.Channels
	}

	return s.tgClient.Search(req.Keyword, channels)
}

// filterByCloudType 按网盘类型过滤
func (s *Service) filterByCloudType(results []model.SearchResult, cloudTypes []string) []model.SearchResult {
	typeMap := make(map[string]bool)
	for _, t := range cloudTypes {
		typeMap[t] = true
	}

	filtered := make([]model.SearchResult, 0)
	for _, result := range results {
		hasMatchingLink := false
		filteredLinks := make([]model.Link, 0)
		
		for _, link := range result.Links {
			if typeMap[link.Type] {
				filteredLinks = append(filteredLinks, link)
				hasMatchingLink = true
			}
		}
		
		if hasMatchingLink {
			result.Links = filteredLinks
			filtered = append(filtered, result)
		}
	}

	return filtered
}

// mergeByType 按网盘类型合并结果
func (s *Service) mergeByType(results []model.SearchResult) map[string][]model.SearchResult {
	merged := make(map[string][]model.SearchResult)

	for _, result := range results {
		for _, link := range result.Links {
			if _, ok := merged[link.Type]; !ok {
				merged[link.Type] = make([]model.SearchResult, 0)
			}
			
			// 创建只包含当前类型链接的结果
			singleResult := result
			singleResult.Links = []model.Link{link}
			merged[link.Type] = append(merged[link.Type], singleResult)
		}
	}

	return merged
}

// buildCacheKey 构建缓存键
func (s *Service) buildCacheKey(req *model.SearchRequest) string {
	return fmt.Sprintf("search:%s:%s:%s", req.Keyword, req.SourceType, req.ResultType)
}

// 简单的内存缓存
type cache struct {
	data map[string]*cacheItem
	mu   sync.RWMutex
	ttl  time.Duration
}

type cacheItem struct {
	value      interface{}
	expireTime time.Time
}

func newCache(ttl time.Duration) *cache {
	c := &cache{
		data: make(map[string]*cacheItem),
		ttl:  ttl,
	}
	
	// 启动清理协程
	go c.cleanup()
	
	return c
}

func (c *cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.data[key]
	if !ok {
		return nil, false
	}

	if time.Now().After(item.expireTime) {
		return nil, false
	}

	return item.value, true
}

func (c *cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = &cacheItem{
		value:      value,
		expireTime: time.Now().Add(c.ttl),
	}
}

func (c *cache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.data {
			if now.After(item.expireTime) {
				delete(c.data, key)
			}
		}
		c.mu.Unlock()
	}
}
