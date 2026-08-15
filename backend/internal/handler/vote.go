// Package handler 提供 HTTP 接口处理器。
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"myworld-backend/internal/service"
)

// HandleVote 投票：POST /api/vote  请求体: { "option": <任意值> }
func HandleVote(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid JSON body"})
		return
	}
	option, ok := body["option"]
	if !ok || option == nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "missing option"})
		return
	}
	service.AddVote(fmt.Sprint(option))
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

// HandleVotes 获取投票数据：GET /api/votes
func HandleVotes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, service.GetVotes())
}
