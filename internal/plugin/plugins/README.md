# 插件开发指南

## 插件结构

每个插件需要实现 `Plugin` 接口：

```go
type Plugin interface {
    Name() string                // 插件唯一标识（英文）
    DisplayName() string         // 显示名称（中文）
    Description() string         // 插件描述
    Priority() int              // 优先级（1-3，1最高）
    Search(keyword string, ext map[string]interface{}) ([]model.SearchResult, error)
}
```

## 已实现的插件

1. **miaoso** - 喵搜（优先级3）
2. **xys** - 小云搜索（优先级2）
3. **jutoushe** - 剧透社（优先级1）

## 添加新插件

### 步骤1：创建插件文件

在 `internal/plugin/plugins/` 目录下创建新文件，例如 `alupan.go`：

```go
package plugins

import (
    "context"
    "fmt"
    "net/http"
    "time"
    "pansou-openwrt/internal/model"
)

type AlupanPlugin struct {
    client *http.Client
}

func NewAlupanPlugin(client *http.Client) *AlupanPlugin {
    return &AlupanPlugin{client: client}
}

func (p *AlupanPlugin) Name() string        { return "alupan" }
func (p *AlupanPlugin) DisplayName() string { return "阿鲁盘" }
func (p *AlupanPlugin) Description() string { return "阿鲁盘 - 网盘搜索" }
func (p *AlupanPlugin) Priority() int       { return 2 }

func (p *AlupanPlugin) Search(keyword string, ext map[string]interface{}) ([]model.SearchResult, error) {
    // 实现搜索逻辑
    results := make([]model.SearchResult, 0)
    
    // TODO: 实现具体搜索逻辑
    // 1. 构建搜索URL
    // 2. 发送HTTP请求
    // 3. 解析响应
    // 4. 提取链接
    
    return results, nil
}
```

### 步骤2：注册插件

在 `internal/plugin/manager.go` 的 `registerPlugins()` 方法中注册：

```go
func (m *Manager) registerPlugins() {
    // ... 已有插件
    
    m.Register(plugins.NewAlupanPlugin(m.client))
}
```

### 步骤3：添加配置

在 `config.yaml` 中添加插件配置：

```yaml
plugins:
  list:
    alupan:
      enabled: true
      priority: 2
```

## 工具函数

### detectCloudType

检测网盘链接类型：

```go
linkType := detectCloudType("https://pan.baidu.com/s/xxx")
// 返回: "baidu"
```

支持的类型：
- `baidu` - 百度网盘
- `aliyun` - 阿里云盘
- `quark` - 夸克网盘
- `tianyi` - 天翼云盘
- `xunlei` - 迅雷网盘
- `115` - 115网盘
- `pikpak` - PikPak
- `123` - 123盘
- `magnet` - 磁力链接
- `ed2k` - ed2k链接

### extractPassword

提取提取码：

```go
password := extractPassword("分享链接 提取码：1234")
// 返回: "1234"
```

### parseTime

解析时间字符串：

```go
t := parseTime("2024-01-01 12:00:00")
```

## 参考原项目

原项目地址：https://github.com/fish2018/pansou

可参考的插件实现：
- `plugin/miaoso/miaoso.go`
- `plugin/xys/xys.go`
- `plugin/panyq/panyq.go`
- `plugin/alupan/alupan.go`
- 等等...

## 待移植的插件列表

- [ ] alupan - 阿鲁盘
- [ ] mikuclub - 米酷
- [ ] ypfxw - 云盘分享网
- [ ] kkmao - KK猫
- [ ] clxiong - 磁力熊
- [ ] ash - ASH
- [ ] qingying - 轻影
- [ ] meitizy - 美剧资源
- [ ] wuji - 无极
- [ ] dyyj - 电影院
- [ ] labi - 拉比
- [ ] zxzj - 在线之家
- [ ] ddys - 低端影视
- [ ] lou1 - Lou1
- [ ] panyq - 盘友圈
