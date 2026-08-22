// Package handler 提供 HTTP 接口处理器。
//
// public.go 提供「对外公开接口」（/public/* 前缀）。
// 与 /api/* 接口功能完全同构（读写一致），仅路由前缀不同：
//   - /api/*    面向前端 UI，适用于前后端同服务器/同域名部署（同源）
//   - /public/* 面向跨域/第三方调用，适用于前后端分离部署
//
// 两者共享同一套 handler 与 service 层逻辑，响应格式完全
// 两者共享同一套 handler 与 service 层逻辑，响应格式完全一致，
// 由前端 .env 的 VITE_API_MODE 选择使用哪一套（见 main.go 路由注册）。
package handler

import (
	"net/http"
	"time"
)

// HandlePublicHealth 对外服务状态：GET /public/health
// 与前端 /api/health 相比，仅返回服务是否在线及基本信息，不暴露内部依赖细节。
func HandlePublicHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"service": "myworld-backend",
		"status":  "ok",
		"version": "public/v1",
		"time":    time.Now().Format(time.RFC3339),
	})
}
