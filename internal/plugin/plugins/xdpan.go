package plugins

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"pan-sou/internal/plugin"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	xdpanBaseURL = "https://xdpan.com"
	xdpanTimeout = 30 * time.Second
)

// XdpanPlugin 兄弟盘插件
type XdpanPlugin struct{}

func init() {
	plugin.Register("xdpan", &XdpanPlugin{})
}

func (p *XdpanPlugin) Name() string {
	return "兄弟盘"
}

func (p *XdpanPlugin) Description() string {
	return "兄弟盘 - 网盘资源搜索引擎"
}

func (p *XdpanPlugin) Search(ctx context.Context, keyword string) ([]plugin.SearchResult, error) {
	// 构建搜索URL（只获取第一页）
	searchURL := fmt.Sprintf("%s/search?page=1&k=%s", xdpanBaseURL, url.QueryEscape(keyword))

	ctx, cancel := context.WithTimeout(ctx, xdpanTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建GET请求失败: %w", err)
	}

	p.setRequestHeaders(req)

	client := &http.Client{
		Timeout: xdpanTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	resp, err := p.doRequestWithRetry(req, client)
	if err != nil {
		return nil, fmt.Errorf("GET请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("请求返回状态码: %d", resp.StatusCode)
	}

	// 解析HTML
	return p.extractSearchResults(resp.Body, keyword, client)
}

// setRequestHeaders 设置请求头
func (p *XdpanPlugin) setRequestHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", xdpanBaseURL+"/")
}

// doRequestWithRetry 带重试的HTTP请求
func (p *XdpanPlugin) doRequestWithRetry(req *http.Request, client *http.Client) (*http.Response, error) {
	maxRetries := 3
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			time.Sleep(time.Duration(i) * time.Second)
		}

		reqClone := req.Clone(req.Context())
		resp, err := client.Do(reqClone)
		if err == nil && resp.StatusCode == http.StatusOK {
			return resp, nil
		}

		if resp != nil {
			resp.Body.Close()
		}
		lastErr = err
	}

	return nil, fmt.Errorf("重试%d次后失败: %w", maxRetries, lastErr)
}

// extractSearchResults 从搜索页面提取结果
func (p *XdpanPlugin) extractSearchResults(body io.Reader, keyword string, client *http.Client) ([]plugin.SearchResult, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("解析HTML失败: %w", err)
	}

	var results []plugin.SearchResult

	// 查找搜索结果项
	doc.Find("div.search-item, article.post").Each(func(i int, s *goquery.Selection) {
		// 提取标题和链接
		titleEl := s.Find("h2 a, h3 a, .title a")
		title := strings.TrimSpace(titleEl.Text())
		detailURL, exists := titleEl.Attr("href")
		if !exists || title == "" {
			return
		}

		// 确保URL是绝对路径
		if strings.HasPrefix(detailURL, "/") {
			detailURL = xdpanBaseURL + detailURL
		}

		// 提取时间
		timeStr := strings.TrimSpace(s.Find("time, .date, .time").Text())
		publishTime := p.parseTime(timeStr)

		// 获取详情页链接
		links := p.fetchDetailLinks(client, detailURL)
		if len(links) == 0 {
			// 如果没有找到链接，仍然返回基本信息
			results = append(results, plugin.SearchResult{
				Title:     title,
				Url:       detailURL,
				Password:  "",
				CloudType: "unknown",
				Size:      "",
				CreatedAt: publishTime,
			})
			return
		}

		// 为每个链接创建结果
		for _, link := range links {
			results = append(results, plugin.SearchResult{
				Title:     title,
				Url:       link.URL,
				Password:  link.Password,
				CloudType: link.Type,
				Size:      "",
				CreatedAt: publishTime,
			})
		}
	})

	// 关键词过滤
	return p.filterByKeyword(results, keyword), nil
}

// fetchDetailLinks 获取详情页的网盘链接
func (p *XdpanPlugin) fetchDetailLinks(client *http.Client, detailURL string) []struct {
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

	p.setRequestHeaders(req)

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

// parseTime 解析时间字符串
func (p *XdpanPlugin) parseTime(timeStr string) string {
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
func (p *XdpanPlugin) filterByKeyword(results []plugin.SearchResult, keyword string) []plugin.SearchResult {
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
