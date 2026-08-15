// Package handler 提供 HTTP 接口处理器。
//
// public.go 提供「对外公开接口」（/public/* 前缀），供第三方/外部系统调用，
// 与面向前端 UI 的 /api/* 接口明确区分。
//
// 对外接口原则：
//   - 全部为只读接口，不提供任何写入能力（如投票等写操作仅保留在 /api/*）
//   - 响应内容为精简的公开数据，不暴露数据库连接等内部实现细节
//   - 与前端接口共用同一套 service 层业务逻辑
package handler

import (
	"net/http"
	"time"

	"myworld-backend/internal/logger"
	"myworld-backend/internal/service"
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

// HandlePublicStatsVotes 对外投票统计（只读）：GET /public/stats/votes
func HandlePublicStatsVotes(w http.ResponseWriter, r *http.Request) {
	votes, err := service.GetVotes()
	if err != nil {
		logger.Error("PUBLIC", "查询对外投票统计失败: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error":   "query failed",
			"message": "公开投票数据暂不可用",
		})
		return
	}
	logger.Info("PUBLIC", "对外投票统计查询成功, 共 %d 个选项", len(votes))
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": votes,
	})
}

// HandlePublicBilibiliVideos 对外 B站视频数据（只读）：GET /public/bilibili/videos?mid=xxx
// 与前端 /api/bilibili/user/videos 复用同一套代理逻辑，仅路由与语义不同。
func HandlePublicBilibiliVideos(w http.ResponseWriter, r *http.Request) {
	mid := r.URL.Query().Get("mid")
	if mid == "" {
		logger.Warn("PUBLIC", "对外接口请求缺少 mid 参数")
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":   "missing mid param",
			"message": "缺少 mid 参数",
		})
		return
	}
	// B站 mid 为纯数字 UID，基础校验
	for _, c := range mid {
		if c < '0' || c > '9' {
			logger.Warn("PUBLIC", "对外接口 mid 参数非法: %q", mid)
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{
				"error":   "invalid mid param",
				"message": "mid 参数非法",
			})
			return
		}
	}
	if len(mid) > 20 {
		logger.Warn("PUBLIC", "对外接口 mid 参数过长: %q", mid)
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":   "invalid mid param",
			"message": "mid 参数过长",
		})
		return
	}
	service.ProxyBilibiliVideos(w, mid)
}
