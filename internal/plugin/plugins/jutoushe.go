package plugins

import (
	"fmt"
	"net/http"

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
	// TODO: 实现真实的剧透社API调用
	results := []model.SearchResult{
		{
			Title:       fmt.Sprintf("【剧透社】测试结果：%s", keyword),
			Description: "这是来自剧透社插件的测试数据",
			Links: []model.Link{
				{
					Type:     "quark",
					URL:      "https://pan.quark.cn/s/test789",
					Password: "xyz1",
				},
			},
			Source: "plugin:jutoushe",
		},
	}
	
	return results, nil
}
