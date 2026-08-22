// Package handler 提供 HTTP 接口处理器。
package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"myworld-backend/internal/logger"
	"myworld-backend/internal/service"
)

// HandleMessageAdd 新增留言：POST /api/messages
// 请求体: { "nickname": "昵称", "content": "留言内容" }，返回 { "success": true, "floor": 楼层号 }
func HandleMessageAdd(w http.ResponseWriter, r *http.Request) {
	// 限制请求体大小，防止超大 body 拖垮服务
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		logger.Warn("MESSAGE", "无效 JSON 请求体: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid JSON body"})
		return
	}
	nickname, _ := body["nickname"].(string)
	content, _ := body["content"].(string)
	nickname = strings.TrimSpace(nickname)
	content = strings.TrimSpace(content)

	if nickname == "" {
		logger.Warn("MESSAGE", "请求缺少昵称")
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "missing nickname"})
		return
	}
	if content == "" {
		logger.Warn("MESSAGE", "请求缺少留言内容")
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "missing content"})
		return
	}

	floor, err := service.AddMessage(nickname, content)
	if err != nil {
		logger.Error("MESSAGE", "留言保存失败 nickname=%s: %v", nickname, err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": "message save failed", "detail": err.Error()})
		return
	}
	logger.Info("MESSAGE", "留言成功 floor=%d nickname=%s", floor, nickname)
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "floor": floor})
}

// HandleMessages 获取留言列表：GET /api/messages
func HandleMessages(w http.ResponseWriter, r *http.Request) {
	messages, err := service.GetMessages()
	if err != nil {
		logger.Error("MESSAGE", "查询留言列表失败: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": "query failed", "detail": err.Error()})
		return
	}
	logger.Info("MESSAGE", "查询留言列表成功, 共 %d 条", len(messages))
	writeJSON(w, http.StatusOK, messages)
}
