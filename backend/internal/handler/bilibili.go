// Package handler 提供 HTTP 接口处理器。
package handler

import (
	"net/http"

	"myworld-backend/internal/service"
)

// HandleBilibiliVideos B站用户视频代理：GET /api/bilibili/user/videos?mid=xxx
func HandleBilibiliVideos(w http.ResponseWriter, r *http.Request) {
	mid := r.URL.Query().Get("mid")
	if mid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "missing mid param"})
		return
	}
	service.ProxyBilibiliVideos(w, mid)
}
