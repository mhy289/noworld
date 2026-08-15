// Package handler 提供 HTTP 接口处理器。
package handler

import (
	"net/http"

	"myworld-backend/internal/logger"
	"myworld-backend/internal/service"
)

// HandleBilibiliVideos B站用户视频代理：GET /api/bilibili/user/videos?mid=xxx
func HandleBilibiliVideos(w http.ResponseWriter, r *http.Request) {
	mid := r.URL.Query().Get("mid")
	if mid == "" {
		logger.Warn("BILI", "请求缺少 mid 参数")
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "missing mid param"})
		return
	}
	// B站 mid 为纯数字 UID，做基础校验防止非法参数
	for _, c := range mid {
		if c < '0' || c > '9' {
			logger.Warn("BILI", "mid 参数非法: %q", mid)
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid mid param"})
			return
		}
	}
	if len(mid) > 20 {
		logger.Warn("BILI", "mid 参数过长: %q", mid)
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid mid param"})
		return
	}
	service.ProxyBilibiliVideos(w, mid)
}
