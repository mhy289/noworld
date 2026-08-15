// myworld-backend：前后端分离后的 Go 后端，功能对齐原 server.js
// 项目结构规范化改造，目录按职责分层：
//
//	main.go                 入口：启动服务器、注册路由
//	internal/middleware     CORS 中间件
//	internal/handler        HTTP 处理器
//	internal/service        业务逻辑（投票 / B站代理）
//	internal/store          数据访问层（MySQL 持久化）
package main

import (
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"myworld-backend/internal/handler"
	"myworld-backend/internal/logger"
	"myworld-backend/internal/middleware"
	"myworld-backend/internal/store"
)

// killPortOwner 查找并结束占用指定端口的进程。
// 无论通过 .bat 脚本、go run 还是编译后的二进制手动启动，程序都能自动清理旧实例。
func killPortOwner(port string) {
	conn, err := net.Listen("tcp", ":"+port)
	if err != nil {
		// 端口被占用，需要清理旧实例
		_ = conn // conn 可能为 nil，忽略即可
	} else {
		// 端口空闲，无需清理
		conn.Close()
		return
	}

	logger.Warn("START", "端口 %s 已被占用，尝试自动关闭旧实例...", port)

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell", "-NoProfile", "-Command",
			`$pids = Get-NetTCPConnection -LocalPort `+port+` -State Listen -ErrorAction SilentlyContinue | Select-Object -ExpandProperty OwningProcess; if ($pids) { $pids | Select-Object -Unique | ForEach-Object { Write-Host "kill PID $_"; Stop-Process -Id $_ -Force } }`)
	} else {
		cmd = exec.Command("sh", "-c", "lsof -ti tcp:"+port+" | xargs -r kill -9")
	}
	if out, err := cmd.CombinedOutput(); err != nil && !strings.Contains(err.Error(), "exit status") {
		logger.Error("START", "清理旧实例时提示: %s (%v)", string(out), err)
	} else if len(out) > 0 {
		logger.Info("START", "%s", string(out))
	}
}

func main() {
	logger.Section("Go Backend 启动流程")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	logger.Info("START", "目标端口: %s", port)

	// 启动前自动清理占用目标端口的旧实例
	killPortOwner(port)

	// 初始化 MySQL 连接
	logger.Info("START", "正在初始化 MySQL 连接...")
	if err := store.InitDB(); err != nil {
		logger.Error("DB", "MySQL 初始化失败: %v", err)
		logger.Warn("DB", "请确认 MySQL 已启动，并通过环境变量配置连接：")
		logger.Warn("DB", "  DB_HOST (默认 127.0.0.1), DB_PORT (默认 3306), DB_USER (默认 root), DB_PASS, DB_NAME (默认 myworld)")
		os.Exit(1)
	}
	defer store.CloseDB()
	logger.OK("DB", "MySQL 连接成功")

	// 注册路由
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", handler.HandleHealth)
	mux.HandleFunc("POST /api/vote", handler.HandleVote)
	mux.HandleFunc("GET /api/votes", handler.HandleVotes)
	mux.HandleFunc("GET /api/bilibili/user/videos", handler.HandleBilibiliVideos)
	logger.Info("START", "路由注册完成 (%d 个接口)", 4)

	logger.Section("服务已就绪")
	logger.Info("HTTP", "监听地址   : http://localhost:%s", port)
	logger.Info("HTTP", "健康检查   : http://localhost:%s/api/health", port)
	logger.Info("HTTP", "B站视频API : http://localhost:%s/api/bilibili/user/videos?mid=165392864", port)
	logger.Info("HTTP", "访问日志已启用 (方法 路径 状态码 耗时)")

	// 启动 HTTP 服务（依次叠加访问日志 + CORS 中间件）
	handler := middleware.AccessLog(middleware.CORS(mux))
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		logger.Error("HTTP", "服务启动失败: %v", err)
		os.Exit(1)
	}
}
