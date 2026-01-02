package plugins

import (
	"context"
	"encoding/base64"
	encoding_json "encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"pan-sou/internal/plugin"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	xysBaseURL    = "https://www.yunso.net"
	xysTokenPath  = "/index/user/s"
	xysSearchPath = "/api/validate/searchX2"
	xysUserAgent  = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36"
	xysMaxResults = 50
)

// XysPlugin 小云搜索插件
type XysPlugin struct {
	debugMode  bool
	tokenCache sync.Map
	cacheTTL   time.Duration
}

// TokenCache token缓存结构
type TokenCache struct {
	Token     string
	Timestamp time.Time
}

// SearchResponse API响应结构
type SearchResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Time string `json:"time"`
	Data string `json:"data"`
}

func init() {
	plugin.Register("xys", &XysPlugin{
		debugMode: false,
		cacheTTL:  30 * time.Minute,
	})
}

func (p *XysPlugin) Name() string {
	return "小云搜索"
}

func (p *XysPlugin) Description() string {
	return "小云搜索 - 阿里云盘、夸克网盘、百度网盘等多网盘搜索引擎"
}

func (p *XysPlugin) Search(ctx context.Context, keyword string) ([]plugin.SearchResult, error) {
	if p.debugMode {
		log.Printf("[XYS] 开始搜索: %s", keyword)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	// 第一步：获取token
	token, err := p.getToken(client, keyword)
	if err != nil {
		return nil, fmt.Errorf("获取token失败: %w", err)
	}

	if p.debugMode {
		log.Printf("[XYS] 获取到token: %s", token[:10]+"...")
	}

	// 第二步：执行搜索
	results, err := p.executeSearch(client, token, keyword)
	if err != nil {
		return nil, fmt.Errorf("执行搜索失败: %w", err)
	}

	if p.debugMode {
		log.Printf("[XYS] 搜索完成，获取到 %d 个结果", len(results))
	}

	return results, nil
}

// getToken 获取搜索token
func (p *XysPlugin) getToken(client *http.Client, keyword string) (string, error) {
	// 检查缓存
	cacheKey := "token"
	if cached, found := p.tokenCache.Load(cacheKey); found {
		if tokenCache, ok := cached.(TokenCache); ok {
			if time.Since(tokenCache.Timestamp) < p.cacheTTL {
				if p.debugMode {
					log.Printf("[XYS] 使用缓存的token")
				}
				return tokenCache.Token, nil
			}
		}
	}

	// 构建请求URL
	tokenURL := fmt.Sprintf("%s%s?wd=%s&mode=undefined&stype=undefined",
		xysBaseURL, xysTokenPath, url.QueryEscape(keyword))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", tokenURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建token请求失败: %w", err)
	}

	// 设置完整的请求头
	req.Header.Set("User-Agent", xysUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Referer", xysBaseURL+"/")

	resp, err := p.doRequestWithRetry(req, client)
	if err != nil {
		return "", fmt.Errorf("token请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("token请求HTTP状态错误: %d", resp.StatusCode)
	}

	// 解析HTML提取token
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("解析token页面HTML失败: %w", err)
	}

	// 查找script标签中的DToken定义
	var token string
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		scriptContent := s.Text()
		if strings.Contains(scriptContent, "DToken") {
			re := regexp.MustCompile(`const\s+DToken\s*=\s*"([^"]+)"`)
			matches := re.FindStringSubmatch(scriptContent)
			if len(matches) > 1 {
				token = matches[1]
				if p.debugMode {
					log.Printf("[XYS] 从script中提取到token: %s", token[:10]+"...")
				}
			}
		}
	})

	if token == "" {
		return "", fmt.Errorf("未找到DToken")
	}

	// 缓存token
	p.tokenCache.Store(cacheKey, TokenCache{
		Token:     token,
		Timestamp: time.Now(),
	})

	return token, nil
}

// doRequestWithRetry 带重试机制的HTTP请求
func (p *XysPlugin) doRequestWithRetry(req *http.Request, client *http.Client) (*http.Response, error) {
	maxRetries := 3
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			backoff := time.Duration(1<<uint(i-1)) * 200 * time.Millisecond
			time.Sleep(backoff)
		}

		reqClone := req.Clone(req.Context())
		resp, err := client.Do(reqClone)
		if err == nil && resp.StatusCode == 200 {
			return resp, nil
		}

		if resp != nil {
			resp.Body.Close()
		}
		lastErr = err
	}

	return nil, fmt.Errorf("重试 %d 次后仍然失败: %w", maxRetries, lastErr)
}

// executeSearch 执行搜索请求
func (p *XysPlugin) executeSearch(client *http.Client, token, keyword string) ([]plugin.SearchResult, error) {
	// 构建搜索URL
	searchURL := fmt.Sprintf("%s%s?DToken2=%s&requestID=undefined&mode=90002&stype=undefined&scope_content=0&wd=%s&uk=&page=1&limit=20&screen_filetype=",
		xysBaseURL, xysSearchPath, token, url.QueryEscape(keyword))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建搜索请求失败: %w", err)
	}

	// 设置完整的请求头
	req.Header.Set("User-Agent", xysUserAgent)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", xysBaseURL+"/")
	req.Header.Set("Origin", xysBaseURL)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := p.doRequestWithRetry(req, client)
	if err != nil {
		return nil, fmt.Errorf("搜索请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("搜索请求HTTP状态错误: %d", resp.StatusCode)
	}

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	// 解析JSON响应
	var searchResp SearchResponse
	if err := encoding_json.Unmarshal(respBody, &searchResp); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	if searchResp.Code != 0 {
		return nil, fmt.Errorf("搜索API返回错误: %s", searchResp.Msg)
	}

	if p.debugMode {
		log.Printf("[XYS] 搜索API响应成功，data长度: %d", len(searchResp.Data))
	}

	// 解析HTML内容
	return p.parseSearchResults(searchResp.Data, keyword)
}

// parseSearchResults 解析搜索结果HTML
func (p *XysPlugin) parseSearchResults(htmlData, keyword string) ([]plugin.SearchResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlData))
	if err != nil {
		return nil, fmt.Errorf("解析搜索结果HTML失败: %w", err)
	}

	var results []plugin.SearchResult

	// 查找搜索结果项
	doc.Find(".layui-card[data-qid]").Each(func(i int, s *goquery.Selection) {
		if len(results) >= xysMaxResults {
			return
		}

		result := p.parseResultItem(s, i+1)
		if result != nil {
			results = append(results, *result)
		}
	})

	if p.debugMode {
		log.Printf("[XYS] 解析到 %d 个原始结果", len(results))
	}

	// 关键词过滤
	filteredResults := filterResultsByKeyword(results, keyword)

	if p.debugMode {
		log.Printf("[XYS] 关键词过滤后剩余 %d 个结果", len(filteredResults))
	}

	return filteredResults, nil
}

// parseResultItem 解析单个搜索结果项
func (p *XysPlugin) parseResultItem(s *goquery.Selection, index int) *plugin.SearchResult {
	// 提取QID
	qid, _ := s.Attr("data-qid")
	if qid == "" {
		return nil
	}

	// 提取标题和链接
	linkEl := s.Find(`a[onclick="open_sid(this)"]`)
	if linkEl.Length() == 0 {
		return nil
	}

	// 提取标题
	title := p.cleanTitle(linkEl.Text())
	if title == "" {
		return nil
	}

	// 提取链接URL
	href, _ := linkEl.Attr("href")
	if href == "" {
		urlAttr, _ := linkEl.Attr("url")
		if urlAttr != "" {
			if decoded, err := base64.StdEncoding.DecodeString(urlAttr); err == nil {
				href = string(decoded)
			}
		}
	}

	if href == "" {
		if p.debugMode {
			log.Printf("[XYS] 跳过无链接的结果: %s", title)
		}
		return nil
	}

	// 提取密码
	password, _ := linkEl.Attr("pa")

	// 提取时间
	timeStr := strings.TrimSpace(s.Find(".layui-icon-time").Parent().Text())
	publishTime := p.parseTime(timeStr)

	// 提取网盘类型
	platform := p.extractPlatform(s, href)

	return &plugin.SearchResult{
		Title:     title,
		Url:       href,
		Password:  password,
		CloudType: platform,
		Size:      "",
		CreatedAt: publishTime,
	}
}

// cleanTitle 清理标题
func (p *XysPlugin) cleanTitle(title string) string {
	if title == "" {
		return ""
	}

	// 移除HTML标签
	re := regexp.MustCompile(`<[^>]*>`)
	cleaned := re.ReplaceAllString(title, "")

	// 移除@符号
	cleaned = strings.ReplaceAll(cleaned, "@", "")

	// 清理多余的空格
	cleaned = strings.TrimSpace(cleaned)
	re = regexp.MustCompile(`\s+`)
	cleaned = re.ReplaceAllString(cleaned, " ")

	return cleaned
}

// parseTime 解析时间字符串
func (p *XysPlugin) parseTime(timeStr string) string {
	timeStr = strings.TrimSpace(timeStr)
	if timeStr == "" {
		return time.Now().Format("2006-01-02 15:04:05")
	}

	// 常见格式
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02",
		"2006/01/02",
		"01-02 15:04",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t.Format("2006-01-02 15:04:05")
		}
	}

	return time.Now().Format("2006-01-02 15:04:05")
}

// extractPlatform 提取网盘平台类型
func (p *XysPlugin) extractPlatform(s *goquery.Selection, href string) string {
	// 检查显示的平台标签
	platformText := strings.TrimSpace(s.Find(".layui-badge-rim").Text())
	if platformText != "" {
		switch {
		case strings.Contains(platformText, "阿里"):
			return "aliyun"
		case strings.Contains(platformText, "夸克"):
			return "quark"
		case strings.Contains(platformText, "百度"):
			return "baidu"
		case strings.Contains(platformText, "迅雷"):
			return "xunlei"
		case strings.Contains(platformText, "UC"):
			return "uc"
		}
	}

	// 从URL判断
	return detectCloudType(href)
}

// filterResultsByKeyword 过滤搜索结果
func filterResultsByKeyword(results []plugin.SearchResult, keyword string) []plugin.SearchResult {
	if keyword == "" {
		return results
	}

	keyword = strings.ToLower(keyword)
	var filtered []plugin.SearchResult

	for _, result := range results {
		title := strings.ToLower(result.Title)
		if strings.Contains(title, keyword) {
			filtered = append(filtered, result)
		}
	}

	return filtered
}
