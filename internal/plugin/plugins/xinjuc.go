package plugins

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"pan-sou/internal/plugin"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	xinjucSiteURL = "https://www.xinjuclub.com"
	xinjucTimeout = 30 * time.Second
)

// XinjucPlugin 新剧坊插件
type XinjucPlugin struct{}

func init() {
	plugin.Register("xinjuc", &XinjucPlugin{})
}

func (p *XinjucPlugin) Name() string {
	return "新剧坊"
}

func (p *XinjucPlugin) Description() string {
	return "新剧坊 - 影视资源搜索平台"
}

func (p *XinjucPlugin) Search(ctx context.Context, keyword string) ([]plugin.SearchResult, error) {
	// 构建搜索URL
	searchURL := fmt.Sprintf("%s/?s=%s", xinjucSiteURL, url.QueryEscape(keyword))

	ctx, cancel := context.WithTimeout(ctx, xinjucTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", xinjucSiteURL)

	client := &http.Client{
		Timeout: xinjucTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	// 带重试机制发送请求
	resp, err := p.doRequestWithRetry(req, client)
	if err != nil {
		return nil, fmt.Errorf("搜索请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("搜索请求返回状态码: %d", resp.StatusCode)
	}

	// 解析搜索结果页面
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("解析搜索页面失败: %w", err)
	}

	// 提取搜索结果
	var results []plugin.SearchResult

	// 查找搜索结果列表
	postList := doc.Find("div.row-xs.post-list article.post-item")
	if postList.Length() == 0 {
		return []plugin.SearchResult{}, nil // 没有搜索结果
	}

	// 解析每个搜索结果项
	postList.Each(func(i int, s *goquery.Selection) {
		result := p.parseSearchItem(s, keyword, client)
		if result != nil {
			results = append(results, *result)
		}
	})

	// 关键词过滤
	return p.filterByKeyword(results, keyword), nil
}

// parseSearchItem 解析单个搜索结果项
func (p *XinjucPlugin) parseSearchItem(s *goquery.Selection, keyword string, client *http.Client) *plugin.SearchResult {
	// 提取标题和链接
	titleEl := s.Find("h2.entry-title a")
	title := strings.TrimSpace(titleEl.Text())
	detailURL, exists := titleEl.Attr("href")
	if !exists || title == "" {
		return nil
	}

	// 提取时间
	timeStr := strings.TrimSpace(s.Find("time.entry-date").Text())
	publishTime := p.parseTime(timeStr)

	// 提取摘要
	content := strings.TrimSpace(s.Find("div.entry-excerpt").Text())

	// 获取详情页信息（包含网盘链接）
	links := p.fetchDetailLinks(client, detailURL)
	if len(links) == 0 {
		// 如果没有找到链接，仍然返回基本信息
		return &plugin.SearchResult{
			Title:     title,
			Url:       detailURL,
			Password:  "",
			CloudType: "unknown",
			Size:      "",
			CreatedAt: publishTime,
		}
	}

	// 返回第一个链接（如果有多个可以优化为返回所有）
	link := links[0]
	return &plugin.SearchResult{
		Title:     title,
		Url:       link.URL,
		Password:  link.Password,
		CloudType: link.Type,
		Size:      "",
		CreatedAt: publishTime,
	}
}

// fetchDetailLinks 获取详情页的网盘链接
func (p *XinjucPlugin) fetchDetailLinks(client *http.Client, detailURL string) []struct {
	Type     string
	URL      string
	Password string
} {
	var links []struct {
		Type     string
		URL      string
		Password string
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", detailURL, nil)
	if err != nil {
		return links
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Referer", xinjucSiteURL)

	resp, err := client.Do(req)
	if err != nil {
		return links
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return links
	}

	bodyStr := string(body)

	// 查找网盘链接
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(bodyStr))
	if err != nil {
		return links
	}

	// 提取所有链接
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		cloudType := detectCloudType(href)
		if cloudType != "unknown" {
			password := extractPassword(bodyStr)
			links = append(links, struct {
				Type     string
				URL      string
				Password string
			}{
				Type:     cloudType,
				URL:      href,
				Password: password,
			})
		}
	})

	return links
}

// doRequestWithRetry 带重试机制的HTTP请求
func (p *XinjucPlugin) doRequestWithRetry(req *http.Request, client *http.Client) (*http.Response, error) {
	maxRetries := 3
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			time.Sleep(time.Duration(i) * time.Second)
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

	return nil, fmt.Errorf("重试%d次后失败: %w", maxRetries, lastErr)
}

// parseTime 解析时间字符串
func (p *XinjucPlugin) parseTime(timeStr string) string {
	if timeStr == "" {
		return time.Now().Format("2006-01-02 15:04:05")
	}

	formats := []string{
		"2006-01-02",
		"2006/01/02",
		"2006-01-02 15:04:05",
		"2006年01月02日",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t.Format("2006-01-02 15:04:05")
		}
	}

	return time.Now().Format("2006-01-02 15:04:05")
}

// filterByKeyword 过滤搜索结果
func (p *XinjucPlugin) filterByKeyword(results []plugin.SearchResult, keyword string) []plugin.SearchResult {
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
