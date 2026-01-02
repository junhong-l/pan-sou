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
	ypfxwSearchURL = "https://ypfxw.com/search.php?q=%s"
	ypfxwTimeout   = 30 * time.Second
	ypfxwMaxRetry  = 3
)

// YpfxwPlugin 云盘分享网插件
type YpfxwPlugin struct{}

func init() {
	plugin.Register("ypfxw", &YpfxwPlugin{})
}

func (p *YpfxwPlugin) Name() string {
	return "云盘分享网"
}

func (p *YpfxwPlugin) Description() string {
	return "云盘分享网 - 网盘资源分享平台"
}

func (p *YpfxwPlugin) Search(ctx context.Context, keyword string) ([]plugin.SearchResult, error) {
	searchURL := fmt.Sprintf(ypfxwSearchURL, url.QueryEscape(keyword))

	ctx, cancel := context.WithTimeout(ctx, ypfxwTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", "https://ypfxw.com/")

	client := &http.Client{
		Timeout: ypfxwTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	// 带重试的请求
	resp, err := p.doRequestWithRetry(req, client)
	if err != nil {
		return nil, fmt.Errorf("搜索请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("请求返回状态码: %d", resp.StatusCode)
	}

	// 解析HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("解析HTML失败: %w", err)
	}

	// 提取搜索结果
	return p.parseSearchResults(doc, keyword, client)
}

// doRequestWithRetry 带重试的HTTP请求
func (p *YpfxwPlugin) doRequestWithRetry(req *http.Request, client *http.Client) (*http.Response, error) {
	var lastErr error

	for i := 0; i < ypfxwMaxRetry; i++ {
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

	return nil, fmt.Errorf("重试%d次后失败: %w", ypfxwMaxRetry, lastErr)
}

// parseSearchResults 解析搜索结果
func (p *YpfxwPlugin) parseSearchResults(doc *goquery.Document, keyword string, client *http.Client) ([]plugin.SearchResult, error) {
	var results []plugin.SearchResult

	// 查找搜索结果项
	doc.Find("article.post").Each(func(i int, s *goquery.Selection) {
		// 提取标题和链接
		titleEl := s.Find("h2.entry-title a")
		title := strings.TrimSpace(titleEl.Text())
		detailURL, exists := titleEl.Attr("href")
		if !exists || title == "" {
			return
		}

		// 提取时间
		timeStr := strings.TrimSpace(s.Find("time.entry-date").Text())
		publishTime := p.parseTime(timeStr)

		// 提取文章ID用于获取详情
		articleID := p.extractArticleID(detailURL)

		// 获取详情页链接（异步）
		links := p.fetchDetailLinks(client, detailURL, articleID)

		// 为每个链接创建一个结果
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

// extractArticleID 从URL提取文章ID
func (p *YpfxwPlugin) extractArticleID(detailURL string) string {
	// 提取URL中的ID，例如：https://ypfxw.com/12345.html
	re := regexp.MustCompile(`/(\d+)\.html`)
	matches := re.FindStringSubmatch(detailURL)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// fetchDetailLinks 获取详情页的网盘链接
func (p *YpfxwPlugin) fetchDetailLinks(client *http.Client, detailURL, articleID string) []struct {
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

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

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
func (p *YpfxwPlugin) parseTime(timeStr string) string {
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
func (p *YpfxwPlugin) filterByKeyword(results []plugin.SearchResult, keyword string) []plugin.SearchResult {
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
