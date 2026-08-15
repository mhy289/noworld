// Package service 提供业务逻辑层。
package service

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"

	"myworld-backend/internal/logger"
)

// B站代理限流：最小间隔 3 秒 + 额外随机 0-2 秒，避免触发风控
var (
	lastRequestTime time.Time
	rateLock        sync.Mutex
	minInterval     = 3 * time.Second
)

// 用户代理池
var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Edge/120.0.0.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
}

func getRandomUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}

// 请求限流：最小间隔 3 秒 + 额外随机 0-2 秒，避免触发风控
func waitBeforeRequest() {
	rateLock.Lock()
	defer rateLock.Unlock()
	now := time.Now()
	elapsed := now.Sub(lastRequestTime)
	if elapsed < minInterval {
		wait := minInterval - elapsed + time.Duration(rand.Intn(2000))*time.Millisecond
		logger.Warn("BILI", "请求频率限制，等待 %s...", wait.Round(time.Second))
		time.Sleep(wait)
	}
	lastRequestTime = time.Now()
}

// ProxyBilibiliVideos 代理 B站用户视频接口，将响应透传给客户端。
func ProxyBilibiliVideos(w http.ResponseWriter, mid string) {
	logger.Info("BILI", "收到视频列表请求, mid=%s", mid)
	waitBeforeRequest()
	logger.Info("BILI", "正在请求B站API, mid=%s", mid)

	// 构造 B站请求（含 Wbi 签名参数）
	req, err := http.NewRequest(http.MethodGet, "https://api.bilibili.com/x/space/arc/search", nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Request setup failed", err.Error())
		return
	}
	q := req.URL.Query()
	q.Set("mid", mid)
	q.Set("ps", "30")
	q.Set("pn", "1")
	q.Set("wts", strconv.FormatInt(time.Now().Unix(), 10))
	q.Set("web_location", "1550101")
	req.URL.RawQuery = q.Encode()

	req.Header.Set("User-Agent", getRandomUserAgent())
	req.Header.Set("Referer", "https://space.bilibili.com/"+mid)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Origin", "https://www.bilibili.com")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("Sec-Ch-Ua", `"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")

	client := &http.Client{Timeout: 25 * time.Second}
	reqStart := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		// 请求已发出但没有收到响应
		logger.Error("BILI", "请求B站失败: %v", err)
		writeError(w, http.StatusServiceUnavailable, "No response from Bilibili server", "B站服务器无响应，请稍后重试")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("BILI", "读取B站响应失败: %v", err)
		writeError(w, http.StatusBadGateway, "Failed to read Bilibili response", err.Error())
		return
	}
	logger.Info("BILI", "B站响应: HTTP %d, %d 字节, 耗时 %s", resp.StatusCode, len(body), time.Since(reqStart).Round(time.Millisecond))

	// 透传 B站原始响应（状态码 + JSON），与 Node 版行为一致
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/json; charset=utf-8"
	}
	w.Header().Set("Content-Type", ct)
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

// writeError 统一错误响应输出（service 层内部使用）。
func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, `{"error":%q,"message":%q}`, code, message)
}
