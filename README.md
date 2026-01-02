# PanSou OpenWrt - 网盘搜索插件

## 项目简介

专为OpenWrt设计的网盘搜索插件，无需登录即可搜索多个网盘平台的资源。

**项目来源**：基于 [fish2018/pansou](https://github.com/fish2018/pansou) 改造，专门适配OpenWrt环境。

## 主要功能

- 支持多个网盘搜索插件（小云搜索、喵搜、剧透社等）
- 支持Telegram频道搜索（自动检测网络连通性）
- 支持12种网盘类型（百度、阿里云盘、夸克、磁力链等）
- LuCI Web管理界面
- 并发搜索，结果缓存
- 可配置并发数、超时时间、缓存时间

## OpenWrt编译

### 1. 克隆到OpenWrt源码

```bash
# 进入OpenWrt源码目录
cd openwrt/package/

# 克隆项目
git clone https://github.com/junhong-l/pan-sou.git
```

### 2. 配置编译选项

```bash
cd openwrt/

# 更新feeds
./scripts/feeds update -a
./scripts/feeds install -a

# 配置选择
make menuconfig
# 进入: Network -> pansou-openwrt [选中]
# 进入: LuCI -> Applications -> luci-app-pansou [选中]
```

### 3. 编译

```bash
# 编译单个包
make package/pan-sou/compile V=s

# 或编译所有包
make -j$(nproc)
```

### 4. 生成文件

编译成功后，在 `bin/packages/你的架构/packages/` 目录生成：

```
bin/packages/x86_64/packages/
├── pansou-openwrt_1.0.0-1_x86_64.ipk        # 主程序包
└── luci-app-pansou_1.0.0-1_all.ipk         # LuCI界面包
```

## 安装使用

```bash
# 上传到OpenWrt
scp dist/*.ipk root@192.168.1.1:/tmp/

# 安装
opkg install /tmp/pansou-openwrt_*.ipk
opkg install /tmp/luci-app-pansou_*.ipk

# 启动
/etc/init.d/pansou enable
/etc/init.d/pansou start
```

访问：`http://192.168.1.1/cgi-bin/luci/admin/services/pansou`

## API使用

```bash
# 搜索
curl "http://192.168.1.1:8888/api/search?kw=电影"

# 或使用POST
curl -X POST http://192.168.1.1:8888/api/search \
  -H "Content-Type: application/json" \
  -d '{"keyword":"电影","result_type":"merge"}'
```

## 配置文件

位置：`/etc/pansou/config.yaml`

```yaml
server:
  port: 8888
  enabled: true

search:
  concurrency: 5      # 并发数
  timeout: 30        # 超时（秒）
  cache_ttl: 60      # 缓存（分钟）

telegram:
  enabled: true
  channels:
    - tgsearchers3

plugins:
  enabled: true
  list:
    xys:
      enabled: true
```

## 许可证

GPL-2.0 License
