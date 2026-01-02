package plugins

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"pansou-openwrt/internal/model"
)

// JutoushePlugin 剧透社插件
type JutoushePlugin struct {
	client *http.Client
}

// NewJutoushePlugin 创建剧透社插件
func NewJutoushePlugin(client *http.Client) *JutoushePlugin {
	return &JutoushePlugin{client: client}
}

func (p *JutoushePlugin) Name() string        { return "jutoushe" }
func (p *JutoushePlugin) DisplayName() string { return "剧透社" }
func (p *JutoushePlugin) Description() string { return "剧透社 - 影视资源搜索" }
func (p *JutoushePlugin) Priority() int       { return 1 }

func (p *JutoushePlugin) Search(keyword string, ext map[string]interface{}) ([]model.SearchResult, error) {
	results := make([]model.SearchResult, 0)
	
	// 构建搜索URL
	baseURL := "https://1.star2.cn"
	searchURL := fmt.Sprintf("%s/search/?keyword=%s", baseURL, url.QueryEscape(keyword))
	
	// 创建请求
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	
	// 设置请求头，避免反爬虫检测
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Referer", baseURL)
	
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
	doc.Find(".module-items .module-item").Each(func(i int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Find(".module-item-title").Text())
		detailLink, _ := s.Find("a").Attr("href")
		
		if title != "" && detailLink != "" {
			// 获取详情页链接
			fullDetailLink := detailLink
			if !strings.HasPrefix(detailLink, "http") {
				fullDetailLink = baseURL + detailLink
			}
			
			// 获取详情页内容
			links := p.fetchDetailLinks(fullDetailLink)
			
			if len(links) > 0 {
				results = append(results, model.SearchResult{
					UniqueID:    fmt.Sprintf("jutoushe-%d", i),
					Title:       title,
					Description: "",
					Links:       links,
					Source:      "plugin:jutoushe",
					PublishTime: time.Now(),
					Datetime:    time.Now().Format("2006-01-02 15:04:05"),
				})
			}
		}
	})
	
	return results, nil
}

// fetchDetailLinks 获取详情页链接
func (p *JutoushePlugin) fetchDetailLinks(detailURL string) []model.Link {
	links := make([]model.Link, 0)
	
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", detailURL, nil)
	if err != nil {
		return links
	}
	
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	
	resp, err := p.client.Do(req)
	if err != nil {
		return links
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return links
	}
	
	content := string(body)
	
	// 提取网盘链接
	links = append(links, extractPanLinks(content)...)
	
	// 提取磁力链接
	links = append(links, extractMagnetLinks(content)...)
	
	return links
}

// extractPanLinks 提取网盘链接
func extractPanLinks(content string) []model.Link {
	links := make([]model.Link, 0)
	
	// 百度网盘
	baiduRe := regexp.MustCompile(`https://pan\.baidu\.com/s/[a-zA-Z0-9_-]+`)
	for _, match := range baiduRe.FindAllString(content, -1) {
		password := extractPasswordNearLink(content, match)
		links = append(links, model.Link{
			Type:     "baidu",
			URL:      match,
			Password: password,
		})
	}
	
	// 阿里云盘
	aliyunRe := regexp.MustCompile(`https://(?:www\.)?aliyundrive\.com/s/[a-zA-Z0-9]+`)
	for _, match := range aliyunRe.FindAllString(content, -1) {
		password := extractPasswordNearLink(content, match)
		links = append(links, model.Link{
			Type:     "aliyun",
			URL:      match,
			Password: password,
		})
	}
	
	// 夸克网盘
	quarkRe := regexp.MustCompile(`https://pan\.quark\.cn/s/[a-zA-Z0-9]+`)
	for _, match := range quarkRe.FindAllString(content, -1) {
		links = append(links, model.Link{
			Type: "quark",
			URL:  match,
		})
	}
	
	return links
}

// extractMagnetLinks 提取磁力链接
func extractMagnetLinks(content string) []model.Link {
	links := make([]model.Link, 0)
	
	magnetRe := regexp.MustCompile(`magnet:\?xt=urn:btih:[a-zA-Z0-9]+`)
	for _, match := range magnetRe.FindAllString(content, -1) {
		links = append(links, model.Link{
			Type: "magnet",
			URL:  match,
		})
	}
	
	return links
}

// extractPasswordNearLink 提取链接附近的提取码
func extractPasswordNearLink(content, linkURL string) string {
	// 查找链接位置
	index := strings.Index(content, linkURL)
	if index == -1 {
		return ""
	}
	
	// 提取链接前后100个字符
	start := index - 50
	if start < 0 {
		start = 0
	}
	end := index + len(linkURL) + 50
	if end > len(content) {
		end = len(content)
	}
	
	nearText := content[start:end]
	
	// 匹配提取码
	re := regexp.MustCompile(`(?:提取码|密码|code|pwd)[：:\s]*([a-zA-Z0-9]{4})`)
	matches := re.FindStringSubmatch(nearText)
	if len(matches) > 1 {
		return matches[1]
	}
	
	return ""
}
