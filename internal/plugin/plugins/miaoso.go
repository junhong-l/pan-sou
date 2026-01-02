package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"pansou-openwrt/internal/model"
)

const (
	miaosoBaseURL = "https://miaosou.fun/api/secendsearch"
)

type MiaosouResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		List []struct {
			Title       string `json:"title"`
			Type        int    `json:"type"` // 1=百度, 2=阿里, 3=夸克等
			Url         string `json:"url"`
			Password    string `json:"password"`
			Description string `json:"description"`
		} `json:"list"`
	} `json:"data"`
}

type MiaosoPlugin struct {
	client *http.Client
}

func NewMiaosoPlugin(client *http.Client) *MiaosoPlugin {
	return &MiaosoPlugin{client: client}
}

func (p *MiaosoPlugin) Name() string        { return "miaoso" }
func (p *MiaosoPlugin) DisplayName() string { return "喵搜" }
func (p *MiaosoPlugin) Description() string { return "喵搜 - 多网盘搜索引擎" }
func (p *MiaosoPlugin) Priority() int       { return 3 }

func (p *MiaosoPlugin) Search(keyword string, ext map[string]interface{}) ([]model.SearchResult, error) {
	searchURL := fmt.Sprintf("%s?name=%s&pageNo=1", miaosoBaseURL, url.QueryEscape(keyword))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://miaosou.fun/")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp MiaosouResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	if apiResp.Code != 200 {
		return nil, fmt.Errorf("API错误: %s", apiResp.Msg)
	}

	results := make([]model.SearchResult, 0)
	for _, item := range apiResp.Data.List {
		cloudType := mapMiaosoType(item.Type)
		if cloudType == "" {
			continue
		}

		results = append(results, model.SearchResult{
			Title:       item.Title,
			Description: item.Description,
			Links: []model.Link{
				{
					Type:     cloudType,
					URL:      item.Url,
					Password: item.Password,
				},
			},
			Source: "plugin:miaoso",
		})
	}

	return results, nil
}

func mapMiaosoType(t int) string {
	switch t {
	case 1:
		return "baidu"
	case 2:
		return "aliyun"
	case 3:
		return "quark"
	case 4:
		return "tianyi"
	case 5:
		return "xunlei"
	case 6:
		return "115"
	case 7:
		return "pikpak"
	case 8:
		return "123"
	default:
		return ""
	}
}
