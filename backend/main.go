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
	"fmt"
	"net/http"
	"os"

	"myworld-backend/internal/handler"
	"myworld-backend/internal/middleware"
	"myworld-backend/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 初始化 MySQL 连接
	if err := store.InitDB(); err != nil {
		fmt.Fprintln(os.Stderr, "Database init failed:", err)
		fmt.Fprintln(os.Stderr, "连接 MySQL 失败。请确认 MySQL 已启动，并通过环境变量配置连接：")
		fmt.Fprintln(os.Stderr, "  DB_HOST (默认 127.0.0.1), DB_PORT (默认 3306), DB_USER (默认 root), DB_PASS, DB_NAME (默认 myworld)")
		os.Exit(1)
	}
	defer store.CloseDB()

	// 注册路由
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", handler.HandleHealth)
	mux.HandleFunc("POST /api/vote", handler.HandleVote)
	mux.HandleFunc("GET /api/votes", handler.HandleVotes)
	mux.HandleFunc("GET /api/bilibili/user/videos", handler.HandleBilibiliVideos)

	fmt.Println("=================================")
	fmt.Printf("Go backend server running on port %s\n", port)
	fmt.Printf("Health check: http://localhost:%s/api/health\n", port)
	fmt.Printf("Bilibili API: http://localhost:%s/api/bilibili/user/videos?mid=165392864\n", port)
	fmt.Println("=================================")

	if err := http.ListenAndServe(":"+port, middleware.CORS(mux)); err != nil {
		fmt.Fprintln(os.Stderr, "Server error:", err)
		os.Exit(1)
	}
}
