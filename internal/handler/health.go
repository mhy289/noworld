// Package handler 提供 HTTP 接口处理器。
package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"myworld-backend/internal/logger"
	"myworld-backend/internal/store"
)

// writeJSON 统一 JSON 响应输出。
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// HandleHealth 健康检查：GET /api/health
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	dbOK := store.PingDB()
	if dbOK {
		logger.Debug("HTTP", "健康检查通过 (数据库正常)")
	} else {
		logger.Warn("HTTP", "健康检查: 数据库连接异常")
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"message":   "Backend server is running",
		"database":  dbStatus(dbOK),
	})
}

func dbStatus(ok bool) map[string]interface{} {
	status := "ok"
	if !ok {
		status = "down"
	}
	return map[string]interface{}{
		"connected": ok,
		"status":    status,
		"type":      "mysql",
	}
}
