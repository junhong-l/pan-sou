package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"pansou-openwrt/internal/config"
	"pansou-openwrt/internal/server"
)

var (
	Version   = "1.0.0"
	BuildTime = "unknown"
)

func main() {
	// 命令行参数
	configPath := flag.String("config", "/etc/pansou/config.yaml", "配置文件路径")
	showVersion := flag.Bool("version", false, "显示版本信息")
	flag.Parse()

	// 显示版本信息
	if *showVersion {
		fmt.Printf("PanSou OpenWrt Version: %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		os.Exit(0)
	}

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化日志
	initLogger(cfg)

	log.Printf("PanSou OpenWrt v%s 启动中...", Version)
	log.Printf("配置文件: %s", *configPath)

	// 创建服务器
	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("创建服务器失败: %v", err)
	}

	// 启动服务器
	if err := srv.Start(); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("正在关闭服务器...")
	srv.Shutdown()
	log.Println("服务器已关闭")
}

func initLogger(cfg *config.Config) {
	// 简单的日志初始化
	// 在OpenWrt环境中，通常使用syslog或文件日志
	logFile := cfg.Logging.File
	if logFile != "" && logFile != "/dev/stdout" {
		f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			log.SetOutput(f)
		}
	}
	
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
