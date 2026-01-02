-- 配置界面模型
local m, s, o

m = Map("pansou", translate("网盘搜索配置"), 
	translate("配置网盘搜索服务的参数"))

-- 服务配置
s = m:section(TypedSection, "config", translate("服务设置"))
s.anonymous = true
s.addremove = false

o = s:option(Flag, "enabled", translate("启用"),
	translate("启用网盘搜索服务"))
o.rmempty = false

o = s:option(Flag, "autostart", translate("开机自启"),
	translate("系统启动时自动启动服务"))
o.rmempty = false

o = s:option(Value, "port", translate("端口"),
	translate("HTTP服务监听端口"))
o.datatype = "port"
o.placeholder = "8888"

-- 搜索配置
s = m:section(TypedSection, "search", translate("搜索设置"))
s.anonymous = true
s.addremove = false

o = s:option(Value, "concurrency", translate("并发数"),
	translate("并发搜索的数量，0为自动"))
o.datatype = "uinteger"
o.placeholder = "5"

o = s:option(Value, "timeout", translate("超时时间"),
	translate("搜索超时时间（秒）"))
o.datatype = "uinteger"
o.placeholder = "30"

o = s:option(Value, "cache_ttl", translate("缓存时间"),
	translate("搜索结果缓存时间（分钟）"))
o.datatype = "uinteger"
o.placeholder = "60"

-- Telegram配置
s = m:section(TypedSection, "telegram", translate("Telegram设置"))
s.anonymous = true
s.addremove = false

o = s:option(Flag, "enabled", translate("启用TG搜索"),
	translate("启用Telegram频道搜索"))
o.rmempty = false

o = s:option(DynamicList, "channels", translate("搜索频道"),
	translate("要搜索的Telegram频道列表"))
o.placeholder = "tgsearchers3"

o = s:option(Value, "check_timeout", translate("网络检测超时"),
	translate("检测Telegram网络连通性的超时时间（秒）"))
o.datatype = "uinteger"
o.placeholder = "5"

o = s:option(Value, "proxy", translate("代理设置"),
	translate("SOCKS5代理，格式：socks5://127.0.0.1:1080"))
o.placeholder = "socks5://127.0.0.1:1080"

-- 插件配置
s = m:section(TypedSection, "plugins", translate("插件设置"))
s.anonymous = true
s.addremove = false

o = s:option(Flag, "enabled", translate("启用插件"),
	translate("全局启用/禁用所有搜索插件"))
o.rmempty = false

-- 插件列表
local plugins = {
	{"xys", "小云搜索", 2},
	{"miaoso", "喵搜", 3},
	{"jutoushe", "剧透社", 1},
	{"alupan", "阿鲁盘", 2},
	{"mikuclub", "米酷", 2},
	{"ypfxw", "云盘分享网", 3},
	{"kkmao", "KK猫", 3},
	{"clxiong", "磁力熊", 2},
	{"ash", "ASH", 2},
	{"qingying", "轻影", 3},
	{"meitizy", "美剧资源", 3},
	{"wuji", "无极", 3},
	{"dyyj", "电影院", 3},
	{"labi", "拉比", 3},
	{"zxzj", "在线之家", 3},
	{"ddys", "低端影视", 3},
	{"lou1", "Lou1", 2},
	{"panyq", "盘友圈", 1},
}

for _, plugin in ipairs(plugins) do
	local name, display, priority = plugin[1], plugin[2], plugin[3]
	
	o = s:option(Flag, "plugin_" .. name, display,
		string.format("优先级: %d", priority))
	o.rmempty = false
end

-- 网盘类型过滤
s = m:section(TypedSection, "cloud_types", translate("网盘类型"))
s.anonymous = true
s.addremove = false
s.description = translate("选择要搜索的网盘类型")

local cloud_types = {
	{"baidu", "百度网盘"},
	{"aliyun", "阿里云盘"},
	{"quark", "夸克网盘"},
	{"tianyi", "天翼云盘"},
	{"uc", "UC网盘"},
	{"mobile", "移动云盘"},
	{"115", "115网盘"},
	{"pikpak", "PikPak"},
	{"xunlei", "迅雷网盘"},
	{"123", "123盘"},
	{"magnet", "磁力链接"},
	{"ed2k", "ed2k链接"},
}

for _, cloud in ipairs(cloud_types) do
	local name, display = cloud[1], cloud[2]
	
	o = s:option(Flag, "type_" .. name, display)
	o.rmempty = false
end

return m
