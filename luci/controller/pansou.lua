-- LuCI模块定义
module("luci.controller.pansou", package.seeall)

function index()
	-- 创建主菜单入口
	local page = entry({"admin", "services", "pansou"}, 
		firstchild(), _("网盘搜索"), 60)
	page.dependent = false

	-- 概览页面
	entry({"admin", "services", "pansou", "overview"}, 
		template("pansou/overview"), _("概览"), 1)

	-- 搜索页面
	entry({"admin", "services", "pansou", "search"}, 
		template("pansou/search"), _("搜索"), 2)

	-- 配置页面
	entry({"admin", "services", "pansou", "config"}, 
		cbi("pansou/config"), _("设置"), 3)

	-- API接口
	entry({"admin", "services", "pansou", "status"}, 
		call("action_status")).leaf = true
	entry({"admin", "services", "pansou", "start"}, 
		call("action_start")).leaf = true
	entry({"admin", "services", "pansou", "stop"}, 
		call("action_stop")).leaf = true
	entry({"admin", "services", "pansou", "restart"}, 
		call("action_restart")).leaf = true
	entry({"admin", "services", "pansou", "search_api"}, 
		call("action_search")).leaf = true
end

-- 获取服务状态
function action_status()
	local util = require "luci.util"
	local sys = require "luci.sys"
	
	local running = sys.call("pgrep -f pansou-openwrt >/dev/null") == 0
	
	luci.http.prepare_content("application/json")
	luci.http.write_json({
		running = running,
		enabled = sys.init.enabled("pansou"),
		pid = running and util.trim(sys.exec("pgrep -f pansou-openwrt")) or nil
	})
end

-- 启动服务
function action_start()
	local sys = require "luci.sys"
	local result = sys.call("/etc/init.d/pansou start >/dev/null 2>&1")
	
	luci.http.prepare_content("application/json")
	luci.http.write_json({
		success = result == 0,
		message = result == 0 and "服务启动成功" or "服务启动失败"
	})
end

-- 停止服务
function action_stop()
	local sys = require "luci.sys"
	local result = sys.call("/etc/init.d/pansou stop >/dev/null 2>&1")
	
	luci.http.prepare_content("application/json")
	luci.http.write_json({
		success = result == 0,
		message = result == 0 and "服务停止成功" or "服务停止失败"
	})
end

-- 重启服务
function action_restart()
	local sys = require "luci.sys"
	local result = sys.call("/etc/init.d/pansou restart >/dev/null 2>&1")
	
	luci.http.prepare_content("application/json")
	luci.http.write_json({
		success = result == 0,
		message = result == 0 and "服务重启成功" or "服务重启失败"
	})
end

-- 搜索API
function action_search()
	local http = require "luci.http"
	local json = require "luci.jsonc"
	local uci = require "luci.model.uci".cursor()
	
	-- 获取请求参数
	local keyword = http.formvalue("keyword")
	if not keyword or keyword == "" then
		http.prepare_content("application/json")
		http.write_json({
			success = false,
			message = "搜索关键词不能为空"
		})
		return
	end
	
	-- 获取端口配置
	local port = uci:get("pansou", "config", "port") or "8888"
	
	-- 构建API请求
	local api_url = string.format("http://127.0.0.1:%s/api/search", port)
	local curl_cmd = string.format(
		"curl -s -X POST '%s' -H 'Content-Type: application/json' -d '{\"keyword\":\"%s\",\"result_type\":\"merge\"}'",
		api_url, keyword
	)
	
	-- 执行请求
	local result = luci.util.exec(curl_cmd)
	
	-- 返回结果
	http.prepare_content("application/json")
	http.write(result)
end
