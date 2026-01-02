package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"pan-sou/internal/plugin"
	"regexp"
	"strings"
	"time"
)

const (
	yunsouSearchURL = "https://yunsou.xyz/s/%s.html"
	yunsouTimeout   = 30 * time.Second
)

var (
	// 提取JSON数据的正则表达式
	jsonDataRegex = regexp.MustCompile(`var jsonData = '(.+?)';`)

	// 提取pwd参数的正则表达式
	pwdParamRegex = regexp.MustCompile(`[?&]pwd=([0-9a-zA-Z]+)`)

	// 控制字符清理正则
	controlCharsRegex = regexp.MustCompile(`[\x00-\x1F\x7F]`)
)

// YunsouPlugin 云搜插件
type YunsouPlugin struct{}

// YunsouData JSON数据结构
type YunsouData struct {
	ID               int             `json:"id"`
	IsType           int             `json:"is_type"` // 0=夸克, 1=阿里, 2=百度, 3=UC, 4=迅雷
	Code             *string         `json:"code"`    // 提取码，可能为null
	URL              string          `json:"url"`
	IsTime           int             `json:"is_time"`
	Name             string          `json:"name"`
	Times            string          `json:"times"` // 发布时间 "2025-07-27"
	Category         YunsouCategory  `json:"category"`
}

type YunsouCategory struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func init() {
	plugin.Register("yunsou", &YunsouPlugin{})
}

func (p *YunsouPlugin) Name() string {
	return "云搜"
}

func (p *YunsouPlugin) Description() string {
	return "云搜 - 网盘资源搜索引擎"
}

func (p *YunsouPlugin) Search(ctx context.Context, keyword string) ([]plugin.SearchResult, error) {
	// 构建搜索URL
	searchURL := fmt.Sprintf(yunsouSearchURL, url.QueryEscape(keyword))

	ctx, cancel := context.WithTimeout(ctx, yunsouTimeout)
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
	req.Header.Set("Referer", "https://yunsou.xyz/")

	client := &http.Client{
		Timeout: yunsouTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("请求返回状态码: %d", resp.StatusCode)
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析结果
	return p.parseResults(string(body), keyword)
}

// parseResults 解析搜索结果
func (p *YunsouPlugin) parseResults(htmlData, keyword string) ([]plugin.SearchResult, error) {
	// 提取JSON数据
	matches := jsonDataRegex.FindStringSubmatch(htmlData)
	if len(matches) < 2 {
		return []plugin.SearchResult{}, nil // 没有找到结果
	}

	jsonStr := matches[1]

	// 清理控制字符
	jsonStr = controlCharsRegex.ReplaceAllString(jsonStr, "")

	// 解析JSON
	var dataList []YunsouData
	if err := json.Unmarshal([]byte(jsonStr), &dataList); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	// 转换为标准格式
	var results []plugin.SearchResult
	for _, data := range dataList {
		cloudType := p.mapCloudType(data.IsType)

		// 提取密码
		password := ""
		if data.Code != nil {
			password = *data.Code
		}

		// 如果JSON中没有密码，尝试从URL提取
		if password == "" {
			if matches := pwdParamRegex.FindStringSubmatch(data.URL); len(matches) > 1 {
				password = matches[1]
			}
		}

		results = append(results, plugin.SearchResult{
			Title:     data.Name,
			Url:       data.URL,
			Password:  password,
			CloudType: cloudType,
			Size:      "",
			CreatedAt: p.parseTime(data.Times),
		})
	}

	// 关键词过滤
	return p.filterByKeyword(results, keyword), nil
}

// mapCloudType 映射云盘类型
func (p *YunsouPlugin) mapCloudType(isType int) string {
	switch isType {
	case 0:
		return "quark"
	case 1:
		return "aliyun"
	case 2:
		return "baidu"
	case 3:
		return "uc"
	case 4:
		return "xunlei"
	default:
		return "unknown"
	}
}

// parseTime 解析时间字符串
func (p *YunsouPlugin) parseTime(timeStr string) string {
	if timeStr == "" {
		return time.Now().Format("2006-01-02 15:04:05")
	}

	// 尝试解析常见格式
	formats := []string{
		"2006-01-02",
		"2006/01/02",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t.Format("2006-01-02 15:04:05")
		}
	}

	return time.Now().Format("2006-01-02 15:04:05")
}

// filterByKeyword 过滤搜索结果
func (p *YunsouPlugin) filterByKeyword(results []plugin.SearchResult, keyword string) []plugin.SearchResult {
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
