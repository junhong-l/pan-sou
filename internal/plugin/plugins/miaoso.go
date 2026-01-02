package plugins

import (
	"fmt"
	"net/http"

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
	// TODO: 实现真实的喵搜API调用
	// 当前返回测试数据以验证流程
	results := []model.SearchResult{
		{
			Title:       fmt.Sprintf("【喵搜】测试结果：%s", keyword),
			Description: "这是来自喵搜插件的测试数据，用于验证系统运行正常",
			Links: []model.Link{
				{
					Type:     "baidu",
					URL:      "https://pan.baidu.com/s/test123",
					Password: "1234",
				},
			},
			Source: "plugin:miaoso",
		},
	}
	
	return results, nil
}
