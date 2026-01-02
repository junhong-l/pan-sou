package plugins

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"pansou-openwrt/internal/model"
)

const (
	jutousheBaseURL = "https://www.jutoushe.net"
)

type JutoushePlugin struct {
	client *http.Client
}

func NewJutoushePlugin(client *http.Client) *JutoushePlugin {
	return &JutoushePlugin{client: client}
}

func (p *JutoushePlugin) Name() string        { return "jutoushe" }
func (p *JutoushePlugin) DisplayName() string { return "剧透社" }
func (p *JutoushePlugin) Description() string { return "剧透社 - 影视资源搜索" }
func (p *JutoushePlugin) Priority() int       { return 1 }

func (p *JutoushePlugin) Search(keyword string, ext map[string]interface{}) ([]model.SearchResult, error) {
	searchURL := fmt.Sprintf("%s/search.html?wd=%s", jutousheBaseURL, url.QueryEscape(keyword))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP状态码: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	results := make([]model.SearchResult, 0)

	doc.Find(".search-item").Each(func(i int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Find(".title").Text())
		if title == "" {
			return
		}

		description := strings.TrimSpace(s.Find(".description").Text())
		links := make([]model.Link, 0)

		s.Find("a[href]").Each(func(j int, link *goquery.Selection) {
			href, _ := link.Attr("href")
			if href == "" {
				return
			}

			cloudType := detectCloudType(href)
			if cloudType == "" {
				return
			}

			password := extractPassword(link.Text() + " " + description)

			links = append(links, model.Link{
				Type:     cloudType,
				URL:      href,
				Password: password,
			})
		})

		if len(links) > 0 {
			results = append(results, model.SearchResult{
				Title:       title,
				Description: description,
				Links:       links,
				Source:      "plugin:jutoushe",
			})
		}
	})

	return results, nil
}

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
	} else if strings.Contains(urlStr, "123pan.com") {
		return "123"
	} else if strings.HasPrefix(urlStr, "magnet:") {
		return "magnet"
	} else if strings.HasPrefix(urlStr, "ed2k:") {
		return "ed2k"
	}
	return ""
}

func extractPassword(text string) string {
	re := regexp.MustCompile(`(?:提取码|密码|code|Code)[：:]\s*([a-zA-Z0-9]{4})`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
