package plugins

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"pansou-openwrt/internal/model"
)

// XysPlugin 小云搜索插件
type XysPlugin struct {
	client *http.Client
}

// NewXysPlugin 创建小云搜索插件
func NewXysPlugin(client *http.Client) *XysPlugin {
	return &XysPlugin{client: client}
}

func (p *XysPlugin) Name() string        { return "xys" }
func (p *XysPlugin) DisplayName() string { return "小云搜索" }
func (p *XysPlugin) Description() string { return "小云搜索 - 阿里云盘、夸克网盘、百度网盘等多网盘搜索引擎" }
func (p *XysPlugin) Priority() int       { return 2 }

func (p *XysPlugin) Search(keyword string, ext map[string]interface{}) ([]model.SearchResult, error) {
	results := make([]model.SearchResult, 0)
	
	// 第一步：获取token
	token, err := p.getToken(keyword)
	if err != nil {
		return nil, fmt.Errorf("获取token失败: %w", err)
	}
	
	// 第二步：执行搜索
	searchURL := "https://www.yunso.net/api/validate/searchX2"
	
	// 构建表单数据
	formData := url.Values{}
	formData.Set("keyword", keyword)
	formData.Set("token", token)
	formData.Set("page", "1")
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "POST", searchURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	
	// 设置请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "https://www.yunso.net/")
	req.Header.Set("Origin", "https://www.yunso.net")
	
	// 发送请求
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP状态码: %d", resp.StatusCode)
	}
	
	// 解析响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	
	// 解析HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("解析HTML失败: %w", err)
	}
	
	// 提取搜索结果
	doc.Find(".search-result-item").Each(func(i int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Find(".title").Text())
		description := strings.TrimSpace(s.Find(".description").Text())
		timeStr := strings.TrimSpace(s.Find(".time").Text())
		
		// 提取链接
		links := make([]model.Link, 0)
		s.Find("a[data-url]").Each(func(j int, link *goquery.Selection) {
			href, _ := link.Attr("data-url")
			password, _ := link.Attr("data-password")
			linkType := detectCloudType(href)
			
			if linkType != "" {
				links = append(links, model.Link{
					Type:     linkType,
					URL:      href,
					Password: password,
				})
			}
		})
		
		if len(links) > 0 {
			results = append(results, model.SearchResult{
				UniqueID:    fmt.Sprintf("xys-%d", i),
				Title:       title,
				Description: description,
				Links:       links,
				Source:      "plugin:xys",
				PublishTime: parseTime(timeStr),
				Datetime:    timeStr,
			})
		}
	})
	
	return results, nil
}

// getToken 获取搜索token
func (p *XysPlugin) getToken(keyword string) (string, error) {
	tokenURL := "https://www.yunso.net/index/user/s"
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", tokenURL, nil)
	if err != nil {
		return "", err
	}
	
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://www.yunso.net/")
	
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	
	// 从响应中提取token（简化版本）
	// 实际需要根据页面结构解析
	token := strings.TrimSpace(string(body))
	
	return token, nil
}

// parseTime 解析时间字符串
func parseTime(timeStr string) time.Time {
	// 尝试多种时间格式
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02",
		"2006/01/02 15:04:05",
		"2006/01/02",
	}
	
	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t
		}
	}
	
	return time.Now()
}
