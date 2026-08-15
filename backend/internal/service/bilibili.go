// Package service 提供业务逻辑层。
package service

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
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

// ---------------- Wbi 签名支持 ----------------
// B 站公开的 wbi 签名算法：基于 img_key + sub_key 生成 mixin_key，
// 再对参数做字典排序、URL 编码拼接后取前 32 位 MD5 作为 w_rid。
// 缺少 w_rid 会导致 B 站返回 code=-412 (request was banned)。

// mixinKeyEncTab 是 B 站公开的字符表重排索引，用于从 img_key+sub_key 生成 mixin_key。
var mixinKeyEncTab = []int{
	46, 47, 18, 2, 53, 8, 23, 32, 15, 50, 10, 31, 58, 3, 45, 35, 27, 43, 5, 49,
	33, 9, 42, 19, 29, 28, 14, 39, 12, 38, 41, 13, 37, 48, 7, 16, 24, 55, 40,
	61, 26, 17, 0, 1, 60, 51, 30, 4, 22, 25, 54, 21, 56, 59, 6, 63, 57, 62, 11,
	36, 20, 34, 44, 52,
}

// mixinKeyCache 缓存已获取的 mixin_key 与获取时间（有效期内复用，减少请求）。
var (
	mixinKeyCache   string
	mixinKeyFetched time.Time
	wbiLock         sync.Mutex
)

// 响应缓存：对相同 mid 的 B 站响应做短期缓存，减少实际请求频率，规避 -799 限流。
var (
	videoCache    = make(map[string]videoCacheEntry)
	videoCacheMu  sync.Mutex
	videoCacheTTL = 5 * time.Minute
)

type videoCacheEntry struct {
	body       []byte
	statusCode int
	contentType string
	expireAt   time.Time
}

// buvid3 是一类可复用的匿名 Cookie，能显著降低 B 站对无 Cookie 请求的风控概率。
// 生成一个随机 buvid3 供请求使用（B站接受匿名 buvid3）。
var buvid3 string

func init() {
	buvid3 = randomBuvid3()
}

// randomBuvid3 生成形如 xxxx-xxxx-xxxx-xxxx-xxxxinfoc 的匿名 buvid3。
func randomBuvid3() string {
	const hexChars = "0123456789abcdef"
	randHex := func(n int) string {
		b := make([]byte, n)
		for i := range b {
			b[i] = hexChars[rand.Intn(len(hexChars))]
		}
		return string(b)
	}
	return randHex(8) + "-" + randHex(4) + "-" + randHex(4) + "-" + randHex(4) + "-" + randHex(12) + "infoc"
}

const wbiKeyTTL = 12 * time.Hour

// getMixinKey 获取当前有效的 mixin_key，必要时从 B 站 nav 接口拉取 img_key/sub_key。
func getMixinKey(client *http.Client) (string, error) {
	wbiLock.Lock()
	defer wbiLock.Unlock()

	// 缓存未过期则直接复用
	if mixinKeyCache != "" && time.Since(mixinKeyFetched) < wbiKeyTTL {
		return mixinKeyCache, nil
	}

	// 从 nav 接口获取 img_key 与 sub_key
	req, err := http.NewRequest(http.MethodGet, "https://api.bilibili.com/x/web-interface/nav", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", getRandomUserAgent())
	req.Header.Set("Referer", "https://www.bilibili.com/")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var nav struct {
		Data struct {
			WbiImg struct {
				ImgURL string `json:"img_url"`
				SubURL string `json:"sub_url"`
			} `json:"wbi_img"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&nav); err != nil {
		return "", err
	}
	imgKey := extractKey(nav.Data.WbiImg.ImgURL)
	subKey := extractKey(nav.Data.WbiImg.SubURL)
	if imgKey == "" || subKey == "" {
		return "", fmt.Errorf("nav接口未返回 wbi keys")
	}

	mixin := imgKey + subKey
	var sb strings.Builder
	for _, idx := range mixinKeyEncTab {
		if idx < len(mixin) {
			sb.WriteByte(mixin[idx])
		}
	}
	mixinKeyCache = sb.String()
	mixinKeyFetched = time.Now()
	logger.Info("BILI", "已获取新的 Wbi mixin_key (%d 位)", len(mixinKeyCache))
	return mixinKeyCache, nil
}

// extractKey 从 img_url/sub_url 中提取文件名（不含扩展名），即 key 本体。
func extractKey(rawURL string) string {
	// url 形如 https://i0.hdslb.com/bfs/wbi/7cd084941338484aae1ad9425b84077c.png
	i := strings.LastIndex(rawURL, "/")
	if i < 0 {
		return ""
	}
	name := rawURL[i+1:]
	dot := strings.LastIndex(name, ".")
	if dot >= 0 {
		name = name[:dot]
	}
	return name
}

// encWbi 对查询参数进行 Wbi 签名，返回加入 wts 与 w_rid 后的参数集。
func encWbi(params url.Values, mixinKey string) url.Values {
	// 加入时间戳
	params.Set("wts", strconv.FormatInt(time.Now().Unix(), 10))

	// 字典排序
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 拼接 query 字符串（保持原始编码，避免 QueryEscape 过度转义）
	var pairs []string
	for _, k := range keys {
		pairs = append(pairs, k+"="+params.Get(k))
	}
	queryStr := strings.Join(pairs, "&")

	// 拼接 mixin_key 后取前 32 位 MD5
	raw := queryStr + mixinKey
	sum := md5.Sum([]byte(raw))
	params.Set("w_rid", hex.EncodeToString(sum[:]))
	return params
}

// ProxyBilibiliVideos 代理 B站用户视频接口，将响应透传给客户端。
func ProxyBilibiliVideos(w http.ResponseWriter, mid string) {
	logger.Info("BILI", "收到视频列表请求, mid=%s", mid)

	// 命中缓存则直接返回，避免对 B 站发起重复请求（规避 -799 限流）
	if entry, ok := getVideoCache(mid); ok {
		logger.Info("BILI", "命中响应缓存, mid=%s (缓存 %d 字节)", mid, len(entry.body))
		w.Header().Set("Content-Type", entry.contentType)
		w.WriteHeader(entry.statusCode)
		_, _ = w.Write(entry.body)
		return
	}

	waitBeforeRequest()
	logger.Info("BILI", "正在请求B站API, mid=%s", mid)

	client := &http.Client{Timeout: 25 * time.Second}

	// 获取 Wbi mixin_key（用于签名，缺失会导致 -412）
	mixinKey, err := getMixinKey(client)
	if err != nil {
		logger.Error("BILI", "获取 Wbi 签名密钥失败: %v", err)
		writeError(w, http.StatusBadGateway, "Failed to get wbi key", "获取B站签名密钥失败，请稍后重试")
		return
	}

	// 构造 B站请求（含 Wbi 签名参数）
	params := url.Values{}
	params.Set("mid", mid)
	params.Set("ps", "30")
	params.Set("pn", "1")
	params.Set("web_location", "1550101")
	signed := encWbi(params, mixinKey)

	req, err := http.NewRequest(http.MethodGet, "https://api.bilibili.com/x/space/arc/search?"+signed.Encode(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Request setup failed", err.Error())
		return
	}

	// 附带匿名 Cookie，降低 B 站风控概率
	req.Header.Set("Cookie", "buvid3="+buvid3+"; b_nut="+strconv.FormatInt(time.Now().Unix(), 10)+"; CURRENT_QUALITY=80")
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

	// 仅缓存成功响应（HTTP 200），风控/限流响应不缓存
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/json; charset=utf-8"
	}
	if resp.StatusCode == http.StatusOK {
		setVideoCache(mid, body, resp.StatusCode, ct)
	}

	// 透传 B站原始响应（状态码 + JSON），与 Node 版行为一致
	w.Header().Set("Content-Type", ct)
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

// getVideoCache 读取 B 站视频响应缓存，未命中或过期返回 false。
func getVideoCache(mid string) (videoCacheEntry, bool) {
	videoCacheMu.Lock()
	defer videoCacheMu.Unlock()
	e, ok := videoCache[mid]
	if !ok || time.Now().After(e.expireAt) {
		delete(videoCache, mid)
		return videoCacheEntry{}, false
	}
	return e, true
}

// setVideoCache 写入 B 站视频响应缓存。
func setVideoCache(mid string, body []byte, status int, ct string) {
	videoCacheMu.Lock()
	defer videoCacheMu.Unlock()
	videoCache[mid] = videoCacheEntry{
		body:        body,
		statusCode:  status,
		contentType: ct,
		expireAt:    time.Now().Add(videoCacheTTL),
	}
	// 简单防止缓存无限增长：超过 500 条时清空
	if len(videoCache) > 500 {
		videoCache = make(map[string]videoCacheEntry)
	}
}

// writeError 统一错误响应输出（service 层内部使用）。
func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, `{"error":%q,"message":%q}`, code, message)
}
