// Package handler 提供 HTTP 接口处理器。
package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"myworld-backend/internal/logger"
	"myworld-backend/internal/service"
)

// HandleVisitorReport 访客上报：POST /api/visitor/report
// 请求体: { "ip": "1.2.3.4", "domain": "mhy.ink", "time": "2026-08-16T12:00:00Z" }
func HandleVisitorReport(w http.ResponseWriter, r *http.Request) {
	// 限制请求体大小，防止超大 body 拖垮服务
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		logger.Warn("VISITOR", "无效 JSON 请求体: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid JSON body"})
		return
	}
	ip, _ := body["ip"].(string)
	domain, _ := body["domain"].(string)
	timeStr, _ := body["time"].(string)

	// 解析前端上报时间，非法或缺失则回退到服务器当前时间
	visitedAt := time.Now()
	if timeStr != "" {
		if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
			visitedAt = t
		}
	}

	if err := service.AddVisitorReport(ip, domain, visitedAt); err != nil {
		logger.Error("VISITOR", "访客上报失败 ip=%s domain=%s: %v", ip, domain, err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": "report failed", "detail": err.Error()})
		return
	}
	logger.Info("VISITOR", "访客上报成功 ip=%s domain=%s time=%s", ip, domain, visitedAt.Format(time.RFC3339))
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}
