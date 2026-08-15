// Package handler 提供 HTTP 接口处理器。
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"myworld-backend/internal/logger"
	"myworld-backend/internal/service"
)

// HandleVote 投票：POST /api/vote  请求体: { "option": <字符串|数字> }
func HandleVote(w http.ResponseWriter, r *http.Request) {
	// 限制请求体大小，防止超大 body 拖垮服务
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		logger.Warn("VOTE", "无效 JSON 请求体: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid JSON body"})
		return
	}
	option, ok := body["option"]
	if !ok || option == nil {
		logger.Warn("VOTE", "请求缺少 option 字段")
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "missing option"})
		return
	}
	// 仅接受标量类型的 option，拒绝嵌套对象/数组，避免生成不可控键名
	switch option.(type) {
	case string, float64, bool:
	default:
		logger.Warn("VOTE", "option 类型不合法: %T", option)
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid option type"})
		return
	}
	optStr := fmt.Sprint(option)
	if err := service.AddVote(optStr); err != nil {
		logger.Error("VOTE", "投票失败 option=%s: %v", optStr, err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": "vote failed", "detail": err.Error()})
		return
	}
	logger.Info("VOTE", "投票成功 option=%s", optStr)
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

// HandleVotes 获取投票数据：GET /api/votes
func HandleVotes(w http.ResponseWriter, r *http.Request) {
	votes, err := service.GetVotes()
	if err != nil {
		logger.Error("VOTE", "查询投票数据失败: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": "query failed", "detail": err.Error()})
		return
	}
	logger.Info("VOTE", "查询投票数据成功, 共 %d 个选项", len(votes))
	writeJSON(w, http.StatusOK, votes)
}
