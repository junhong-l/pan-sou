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
	clxiongBaseURL   = "https://www.cilixiong.org"
	clxiongSearchURL = "https://www.cilixiong.org/e/search/index.php"
	clxiongUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	clxiongMaxRetry  = 3
	clxiongRetryDelay = 2 * time.Second
)

// ClxiongPlugin 磁力熊插件
type ClxiongPlugin struct{}

func init() {
	plugin.Register("clxiong", &ClxiongPlugin{})
}

func (p *ClxiongPlugin) Name() string {
	return "磁力熊"
}

func (p *ClxiongPlugin) Description() string {
	return "磁力熊 - 磁力链接搜索引擎"
}

func (p *ClxiongPlugin) Search(ctx context.Context, keyword string) ([]plugin.SearchResult, error) {
	// 第一步：POST搜索获取searchid
	searchID, err := p.getSearchID(keyword)
	if err != nil {
		return nil, fmt.Errorf("获取searchid失败: %w", err)
	}

	// 第二步：GET搜索结果
	results, err := p.getSearchResults(searchID, keyword)
	if err != nil {
		return nil, fmt.Errorf("获取搜索结果失败: %w", err)
	}

	return results, nil
}

// getSearchID 第一步：POST搜索获取searchid
func (p *ClxiongPlugin) getSearchID(keyword string) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // 不自动跟随重定向
		},
	}

	// 准备POST数据
	formData := url.Values{}
	formData.Set("classid", "1,2")    // 1=电影，2=剧集
	formData.Set("show", "title")     // 搜索字段
	formData.Set("tempid", "1")       // 模板ID
	formData.Set("keyboard", keyword) // 搜索关键词

	req, err := http.NewRequest("POST", clxiongSearchURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", clxiongUserAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", clxiongBaseURL+"/")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")

	var resp *http.Response
	var lastErr error

	// 重试机制
	for i := 0; i < clxiongMaxRetry; i++ {
		resp, lastErr = client.Do(req)
		if lastErr == nil && (resp.StatusCode == 302 || resp.StatusCode == 301) {
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
		if i < clxiongMaxRetry-1 {
			time.Sleep(clxiongRetryDelay)
		}
	}

	if lastErr != nil {
		return "", lastErr
	}
	defer resp.Body.Close()

	// 检查重定向响应
	if resp.StatusCode != 302 && resp.StatusCode != 301 {
		return "", fmt.Errorf("期望302重定向，但得到状态码: %d", resp.StatusCode)
	}

	// 从Location头部提取searchid
	location := resp.Header.Get("Location")
	if location == "" {
		return "", fmt.Errorf("重定向响应中没有Location头部")
	}

	// 解析searchid
	searchID := p.extractSearchIDFromLocation(location)
	if searchID == "" {
		return "", fmt.Errorf("无法从Location中提取searchid: %s", location)
	}

	return searchID, nil
}

// extractSearchIDFromLocation 从Location头部提取searchid
func (p *ClxiongPlugin) extractSearchIDFromLocation(location string) string {
	re := regexp.MustCompile(`searchid=(\d+)`)
	matches := re.FindStringSubmatch(location)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// getSearchResults 第二步：GET搜索结果
func (p *ClxiongPlugin) getSearchResults(searchID, keyword string) ([]plugin.SearchResult, error) {
	resultURL := fmt.Sprintf("%s/e/search/result/?searchid=%s", clxiongBaseURL, searchID)

	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("GET", resultURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", clxiongUserAgent)
	req.Header.Set("Referer", clxiongBaseURL+"/")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")

	var resp *http.Response
	var lastErr error

	// 重试机制
	for i := 0; i < clxiongMaxRetry; i++ {
		resp, lastErr = client.Do(req)
		if lastErr == nil && resp.StatusCode == 200 {
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
		if i < clxiongMaxRetry-1 {
			time.Sleep(clxiongRetryDelay)
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("搜索结果请求失败，状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return p.parseSearchResults(string(body), keyword)
}

// parseSearchResults 解析搜索结果页面
func (p *ClxiongPlugin) parseSearchResults(htmlData, keyword string) ([]plugin.SearchResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlData))
	if err != nil {
		return nil, fmt.Errorf("HTML解析失败: %w", err)
	}

	var results []plugin.SearchResult

	// 查找搜索结果项
	doc.Find(".list-group-item").Each(func(i int, s *goquery.Selection) {
		// 提取标题
		titleEl := s.Find("h5.card-title a, .title a")
		title := strings.TrimSpace(titleEl.Text())
		detailURL, _ := titleEl.Attr("href")

		if title == "" {
			return
		}

		// 确保URL是绝对路径
		if strings.HasPrefix(detailURL, "/") {
			detailURL = clxiongBaseURL + detailURL
		}

		// 提取磁力链接（如果页面直接显示）
		magnetLink := ""
		s.Find("a[href^='magnet:']").Each(func(j int, link *goquery.Selection) {
			if href, exists := link.Attr("href"); exists {
				magnetLink = href
			}
		})

		// 提取时间
		timeStr := strings.TrimSpace(s.Find(".text-muted, .time").Text())
		publishTime := p.parseTime(timeStr)

		// 提取大小
		sizeStr := ""
		s.Find(".badge, .size").Each(func(j int, badge *goquery.Selection) {
			text := strings.TrimSpace(badge.Text())
			if strings.Contains(text, "MB") || strings.Contains(text, "GB") || strings.Contains(text, "KB") {
				sizeStr = text
			}
		})

		results = append(results, plugin.SearchResult{
			Title:     title,
			Url:       magnetLink, // 磁力链接
			Password:  "",
			CloudType: "magnet", // 磁力链接类型
			Size:      sizeStr,
			CreatedAt: publishTime,
		})
	})

	// 关键词过滤
	return p.filterByKeyword(results, keyword), nil
}

// parseTime 解析时间字符串
func (p *ClxiongPlugin) parseTime(timeStr string) string {
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
func (p *ClxiongPlugin) filterByKeyword(results []plugin.SearchResult, keyword string) []plugin.SearchResult {
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
