// Package handler 提供 HTTP 接口处理器。
package handler

import (
	"io"
	"net/http"
	"strconv"

	"myworld-backend/internal/logger"
	"myworld-backend/internal/service"
)

const maxUploadBytes = 5 << 20 // 5 MB，超过直接拒绝

// HandlePixelConvert 像素图转换：POST /api/pixel/convert
// 请求: multipart/form-data, 字段 file(图片文件) + size(目标尺寸, 16 或 32, 默认 16)
// 成功: 返回 image/png 二进制（像素点图）
// 失败: 返回 JSON { "error": "..." }
func HandlePixelConvert(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		logger.Warn("PIXEL", "解析表单失败: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid multipart form"})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		logger.Warn("PIXEL", "缺少 file 字段: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "missing file field"})
		return
	}
	defer file.Close()

	// size 仅允许 16 / 32，非法值回退 16
	size := 16
	if s := r.FormValue("size"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && (n == 16 || n == 32) {
			size = n
		}
	}

	data, err := io.ReadAll(io.LimitReader(file, maxUploadBytes))
	if err != nil {
		logger.Warn("PIXEL", "读取文件失败: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "read file failed"})
		return
	}

	out, err := service.ConvertToPixel(data, size)
	if err != nil {
		logger.Warn("PIXEL", "转换失败: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "convert failed", "detail": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	if _, err := w.Write(out); err != nil {
		logger.Warn("PIXEL", "写出响应失败: %v", err)
		return
	}
	logger.Info("PIXEL", "转换成功 size=%d bytes=%d", size, len(out))
}
