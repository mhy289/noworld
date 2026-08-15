// Package handler 提供 HTTP 接口处理器。
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"myworld-backend/internal/logger"
	"myworld-backend/internal/service"
)

// HandleVote 投票：POST /api/vote  请求体: { "option": <任意值> }
func HandleVote(w http.ResponseWriter, r *http.Request) {
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
