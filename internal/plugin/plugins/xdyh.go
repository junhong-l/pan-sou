package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"pan-sou/internal/plugin"
	"strings"
	"time"
)

const (
	xdyhAPIURL = "https://ys.66ds.de/search"
	xdyhTimeout = 15 * time.Second
)

// XdyhPlugin XDYH聚合搜索插件
type XdyhPlugin struct{}

// SearchRequest API请求结构体
type SearchRequest struct {
	Keyword    string      `json:"keyword"`
	Sites      interface{} `json:"sites"` // null or []string
	MaxWorkers int         `json:"max_workers"`
	SaveToFile bool        `json:"save_to_file"`
	SplitLinks bool        `json:"split_links"`
}

// APIResponse API响应结构体
type APIResponse struct {
	Status          string             `json:"status"`
	Keyword         string             `json:"keyword"`
	SearchTimestamp string             `json:"search_timestamp"`
	Summary         Summary            `json:"summary"`
	SuccessfulSites []string           `json:"successful_sites"`
	FailedSites     []string           `json:"failed_sites"`
	Data            []SearchResultItem `json:"data"`
	Performance     Performance        `json:"performance"`
}

type Summary struct {
	TotalSitesSearched      int `json:"total_sites_searched"`
	SuccessfulSites         int `json:"successful_sites"`
	FailedSites             int `json:"failed_sites"`
	TotalResults            int `json:"total_results"`
	TotalValidLinks         int `json:"total_valid_links"`
	TotalAliYunLinks        int `json:"total_aliyun_links"`
	TotalBaiDuLinks         int `json:"total_baidu_links"`
	TotalQuarkLinks         int `json:"total_quark_links"`
	TotalUCLinks            int `json:"total_uc_links"`
	TotalXunLeiLinks        int `json:"total_xunlei_links"`
	TotalUnknownTypeLinks   int `json:"total_unknown_type_links"`
}

type SearchResultItem struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Site        string `json:"site"`
	Password    string `json:"password,omitempty"`
	PublishTime string `json:"publish_time,omitempty"`
	Links       []Link `json:"links,omitempty"`
}

type Link struct {
	Type     string `json:"type"`
	URL      string `json:"url"`
	Password string `json:"password,omitempty"`
}

type Performance struct {
	TotalTime   string `json:"total_time"`
	SearchPhase string `json:"search_phase"`
	LinkPhase   string `json:"link_phase"`
}

func init() {
	plugin.Register("xdyh", &XdyhPlugin{})
}

func (p *XdyhPlugin) Name() string {
	return "XDYH聚合搜索"
}

func (p *XdyhPlugin) Description() string {
	return "XDYH - 聚合多个网盘搜索站点的API"
}

func (p *XdyhPlugin) Search(ctx context.Context, keyword string) ([]plugin.SearchResult, error) {
	// 构建请求体
	requestBody := SearchRequest{
		Keyword:    keyword,
		Sites:      nil, // null表示搜索所有站点
		MaxWorkers: 10,  // API默认并发数
		SaveToFile: false,
		SplitLinks: true,
	}

	// JSON序列化
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("JSON序列化失败: %w", err)
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(ctx, xdyhTimeout)
	defer cancel()

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", xdyhAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	p.setRequestHeaders(req)

	client := &http.Client{
		Timeout: xdyhTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 30,
			MaxConnsPerHost:     50,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	// 发送请求
	resp, err := p.doRequestWithRetry(req, client)
	if err != nil {
		return nil, fmt.Errorf("搜索请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("请求返回状态码: %d", resp.StatusCode)
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析JSON响应
	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	// 检查API响应状态
	if apiResp.Status != "success" {
		return nil, fmt.Errorf("API返回错误状态: %s", apiResp.Status)
	}

	// 转换为标准格式
	return p.convertToSearchResults(apiResp, keyword), nil
}

// setRequestHeaders 设置HTTP请求头
func (p *XdyhPlugin) setRequestHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://ys.66ds.de/")
	req.Header.Set("Origin", "https://ys.66ds.de")
}

// doRequestWithRetry 带重试机制的HTTP请求
func (p *XdyhPlugin) doRequestWithRetry(req *http.Request, client *http.Client) (*http.Response, error) {
	maxRetries := 2
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			time.Sleep(time.Duration(i) * time.Second)
		}

		// 需要重新读取body
		bodyBytes, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == 200 {
			return resp, nil
		}

		if resp != nil {
			resp.Body.Close()
		}
		lastErr = err

		// 为下次重试准备
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	return nil, fmt.Errorf("重试%d次后失败: %w", maxRetries, lastErr)
}

// convertToSearchResults 将API响应转换为标准搜索结果
func (p *XdyhPlugin) convertToSearchResults(apiResp APIResponse, keyword string) []plugin.SearchResult {
	var results []plugin.SearchResult

	for _, item := range apiResp.Data {
		// 如果有分割的链接，为每个链接创建一个结果
		if len(item.Links) > 0 {
			for _, link := range item.Links {
				results = append(results, plugin.SearchResult{
					Title:     item.Title,
					Url:       link.URL,
					Password:  link.Password,
					CloudType: p.normalizeCloudType(link.Type),
					Size:      "",
					CreatedAt: p.parseTime(item.PublishTime),
				})
			}
		} else {
			// 没有分割链接，使用主URL
			cloudType := p.determineCloudType(item.URL)
			results = append(results, plugin.SearchResult{
				Title:     item.Title,
				Url:       item.URL,
				Password:  item.Password,
				CloudType: cloudType,
				Size:      "",
				CreatedAt: p.parseTime(item.PublishTime),
			})
		}
	}

	// 关键词过滤
	return p.filterByKeyword(results, keyword)
}

// normalizeCloudType 标准化云盘类型名称
func (p *XdyhPlugin) normalizeCloudType(cloudType string) string {
	cloudType = strings.ToLower(cloudType)
	switch cloudType {
	case "阿里云盘", "aliyun", "aliyundrive":
		return "aliyun"
	case "百度网盘", "baidu", "baidupan":
		return "baidu"
	case "夸克网盘", "quark":
		return "quark"
	case "uc网盘", "uc":
		return "uc"
	case "迅雷云盘", "xunlei":
		return "xunlei"
	default:
		return cloudType
	}
}

// determineCloudType 从URL判断云盘类型
func (p *XdyhPlugin) determineCloudType(url string) string {
	url = strings.ToLower(url)
	if strings.Contains(url, "aliyundrive.com") || strings.Contains(url, "alipan.com") {
		return "aliyun"
	}
	if strings.Contains(url, "pan.baidu.com") {
		return "baidu"
	}
	if strings.Contains(url, "pan.quark.cn") {
		return "quark"
	}
	if strings.Contains(url, "drive.uc.cn") {
		return "uc"
	}
	if strings.Contains(url, "pan.xunlei.com") {
		return "xunlei"
	}
	return "unknown"
}

// parseTime 解析时间字符串
func (p *XdyhPlugin) parseTime(timeStr string) string {
	if timeStr == "" {
		return time.Now().Format("2006-01-02 15:04:05")
	}

	formats := []string{
		"2006-01-02",
		"2006/01/02",
		"2006-01-02 15:04:05",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t.Format("2006-01-02 15:04:05")
		}
	}

	return timeStr
}

// filterByKeyword 过滤搜索结果
func (p *XdyhPlugin) filterByKeyword(results []plugin.SearchResult, keyword string) []plugin.SearchResult {
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
