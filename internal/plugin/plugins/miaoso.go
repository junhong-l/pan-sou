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

// MiaosoPlugin 喵搜插件
type MiaosoPlugin struct {
	client *http.Client
}

// NewMiaosoPlugin 创建喵搜插件
func NewMiaosoPlugin(client *http.Client) *MiaosoPlugin {
	return &MiaosoPlugin{client: client}
}

func (p *MiaosoPlugin) Name() string        { return "miaoso" }
func (p *MiaosoPlugin) DisplayName() string { return "喵搜" }
func (p *MiaosoPlugin) Description() string { return "喵搜 - 多网盘搜索引擎" }
func (p *MiaosoPlugin) Priority() int       { return 3 }

func (p *MiaosoPlugin) Search(keyword string, ext map[string]interface{}) ([]model.SearchResult, error) {
	results := make([]model.SearchResult, 0)
	
	// 构建搜索URL
	searchURL := fmt.Sprintf("https://miaosou.fun/api/secendsearch?name=%s&pageNo=1", url.QueryEscape(keyword))
	
	// 创建请求
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	
	// 设置请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://miaosou.fun/")
	
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
	
	// 解析HTML（简化版本，实际需要根据API响应格式解析）
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("解析HTML失败: %w", err)
	}
	
	// 提取搜索结果
	doc.Find(".result-item").Each(func(i int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Find(".title").Text())
		description := strings.TrimSpace(s.Find(".description").Text())
		
		// 提取链接
		links := make([]model.Link, 0)
		s.Find("a[href*='pan.']").Each(func(j int, link *goquery.Selection) {
			href, _ := link.Attr("href")
			linkType := detectCloudType(href)
			
			if linkType != "" {
				links = append(links, model.Link{
					Type: linkType,
					URL:  href,
				})
			}
		})
		
		if len(links) > 0 {
			results = append(results, model.SearchResult{
				UniqueID:    fmt.Sprintf("miaoso-%d", i),
				Title:       title,
				Description: description,
				Links:       links,
				Source:      "plugin:miaoso",
				PublishTime: time.Now(),
				Datetime:    time.Now().Format("2006-01-02 15:04:05"),
			})
		}
	})
	
	return results, nil
}

// detectCloudType 检测云盘类型
func detectCloudType(urlStr string) string {
	if strings.Contains(urlStr, "pan.baidu.com") {
		return "baidu"
	} else if strings.Contains(urlStr, "aliyundrive.com") || strings.Contains(urlStr, "alipan.com") {
		return "aliyun"
	} else if strings.Contains(urlStr, "pan.quark.cn") {
		return "quark"
	} else if strings.Contains(urlStr, "cloud.189.cn") {
		return "tianyi"
	} else if strings.Contains(urlStr, "pan.xunlei.com") {
		return "xunlei"
	} else if strings.Contains(urlStr, "115.com") {
		return "115"
	} else if strings.Contains(urlStr, "pikpak.com") {
		return "pikpak"
	} else if strings.Contains(urlStr, "123pan.com") || strings.Contains(urlStr, "123865.com") {
		return "123"
	} else if strings.HasPrefix(urlStr, "magnet:") {
		return "magnet"
	} else if strings.HasPrefix(urlStr, "ed2k:") {
		return "ed2k"
	}
	return ""
}

// extractPassword 提取提取码
func extractPassword(text string) string {
	// 匹配提取码模式：提取码：xxxx 或 密码：xxxx
	re := regexp.MustCompile(`(?:提取码|密码|code)[：:]\s*([a-zA-Z0-9]{4})`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
