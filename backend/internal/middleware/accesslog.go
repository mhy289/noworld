// Package middleware 提供 HTTP 中间件。
package middleware

import (
	"net/http"
	"time"

	"myworld-backend/internal/logger"
)

// statusWriter 包装 ResponseWriter 以捕获响应状态码。
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

// AccessLog 全局访问日志中间件：记录每个请求的方法、路径、客户端、状态码与耗时。
func AccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(sw, r)

		status := sw.status
		level := "INFO"
		if status >= 500 {
			level = "ERROR"
		} else if status >= 400 {
			level = "WARN"
		}
		dur := time.Since(start)
		switch level {
		case "ERROR":
			logger.Error("HTTP", "%s %s %d %s (%s)", r.Method, r.URL.Path, status, dur.Round(time.Millisecond), r.RemoteAddr)
		case "WARN":
			logger.Warn("HTTP", "%s %s %d %s (%s)", r.Method, r.URL.Path, status, dur.Round(time.Millisecond), r.RemoteAddr)
		default:
			logger.Info("HTTP", "%s %s %d %s (%s)", r.Method, r.URL.Path, status, dur.Round(time.Millisecond), r.RemoteAddr)
		}
	})
}
