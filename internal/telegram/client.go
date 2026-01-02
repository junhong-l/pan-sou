package telegram

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"pansou-openwrt/internal/config"
	"pansou-openwrt/internal/model"
)

// Client Telegram客户端
type Client struct {
	config     *config.TelegramConfig
	httpClient *http.Client
	available  bool
}

// NewClient 创建Telegram客户端
func NewClient(cfg *config.TelegramConfig) *Client {
	client := &Client{
		config:    cfg,
		available: false,
	}

	// 创建HTTP客户端
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     30 * time.Second,
	}

	// 如果配置了代理
	if cfg.Proxy != "" {
		proxyURL, err := url.Parse(cfg.Proxy)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
			log.Printf("[TG] 使用代理: %s", cfg.Proxy)
		} else {
			log.Printf("[TG] 代理配置无效: %v", err)
		}
	}

	client.httpClient = &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	// 检查网络连接
	client.checkAvailability()

	return client
}

// checkAvailability 检查Telegram是否可访问
func (c *Client) checkAvailability() {
	ctx, cancel := context.WithTimeout(context.Background(), 
		time.Duration(c.config.CheckTimeout)*time.Second)
	defer cancel()

	// 尝试连接Telegram API
	testURLs := []string{
		"https://api.telegram.org",
		"https://t.me",
	}

	for _, testURL := range testURLs {
		req, err := http.NewRequestWithContext(ctx, "HEAD", testURL, nil)
		if err != nil {
			continue
		}

		resp, err := c.httpClient.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				c.available = true
				log.Println("[TG] Telegram网络连接正常")
				return
			}
		}
	}

	// 如果没有代理，尝试直接TCP连接
	if c.config.Proxy == "" {
		conn, err := net.DialTimeout("tcp", "api.telegram.org:443", 
			time.Duration(c.config.CheckTimeout)*time.Second)
		if err == nil {
			conn.Close()
			c.available = true
			log.Println("[TG] Telegram网络连接正常（TCP）")
			return
		}
	}

	log.Println("[TG] Telegram网络不可访问，将跳过TG搜索")
	c.available = false
}

// IsAvailable 返回Telegram是否可用
func (c *Client) IsAvailable() bool {
	return c.available
}

// Search 搜索Telegram频道
func (c *Client) Search(keyword string, channels []string) ([]model.SearchResult, error) {
	if !c.available {
		return []model.SearchResult{}, nil
	}

	if len(channels) == 0 {
		channels = c.config.Channels
	}

	results := make([]model.SearchResult, 0)

	// 使用Telegram公开Web API搜索
	for _, channel := range channels {
		channelResults, err := c.searchChannel(keyword, channel)
		if err != nil {
			log.Printf("[TG] 搜索频道 %s 失败: %v", channel, err)
			continue
		}
		results = append(results, channelResults...)
	}

	log.Printf("[TG] 搜索关键词: %s, 频道数: %d, 结果数: %d", keyword, len(channels), len(results))
	
	return results, nil
}

// searchChannel 搜索单个频道
func (c *Client) searchChannel(keyword string, channel string) ([]model.SearchResult, error) {
	// 注意：这是简化实现，使用公开的Telegram Web界面
	// 更完整的实现需要Bot Token或MTProto
	
	// 尝试通过t.me访问频道
	channelURL := fmt.Sprintf("https://t.me/s/%s", channel)
	
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", channelURL, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP状态码: %d", resp.StatusCode)
	}
	
	// 这里需要解析HTML并提取包含关键词的消息
	// 由于Telegram Web的HTML结构复杂，这里返回空结果
	// 完整实现建议使用Bot API或第三方服务
	
	log.Printf("[TG] 频道 %s 搜索完成（简化版）", channel)
	return []model.SearchResult{}, nil
}

// SearchWithBotAPI 使用Bot API搜索（需要Bot Token）
func (c *Client) SearchWithBotAPI(keyword string, channels []string, botToken string) ([]model.SearchResult, error) {
	if !c.available {
		return []model.SearchResult{}, nil
	}

	// TODO: 实现Bot API搜索
	// 1. 使用botToken调用getUpdates或searchMessages
	// 2. 过滤包含keyword的消息
	// 3. 提取网盘链接
	
	return []model.SearchResult{}, fmt.Errorf("Bot API搜索待实现")
}

// SearchWithMTProto 使用MTProto搜索（需要API ID和Hash）
func (c *Client) SearchWithMTProto(keyword string, channels []string, apiID int, apiHash string) ([]model.SearchResult, error) {
	if !c.available {
		return []model.SearchResult{}, nil
	}

	// TODO: 实现MTProto搜索
	// 需要使用第三方库如 github.com/gotd/td
	
	return []model.SearchResult{}, fmt.Errorf("MTProto搜索待实现")
}

// RefreshAvailability 重新检查可用性
func (c *Client) RefreshAvailability() {
	c.checkAvailability()
}
