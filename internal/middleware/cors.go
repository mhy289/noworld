// Package middleware 提供 HTTP 中间件。
package middleware

import (
	"net/http"
	"os"
	"strings"
)

// defaultAllowedOrigins 默认允许跨域的前端来源（协议无关，http/https 视为等价）。
// 校验时只比对「主机名 + 端口」，因此单个条目即可覆盖 http 与 https 两种协议。
// 支持两种条目：
//   - 精确条目（如 mhy.ink、localhost:5173）：按「主机名+端口」精确匹配
//   - 通配符条目（如 *.mhy.ink）：匹配任意子域名（www.mhy.ink、sub.mhy.ink 等），
//     不包含根域名本身（根域需另加精确条目 mhy.ink）
//
// 部署时可通过环境变量 CORS_ALLOWED_ORIGINS 追加（逗号分隔，同样支持 *. 通配符写
// 部署时可通过环境变量 CORS_ALLOWED_ORIGINS 追加（逗号分隔，同样支持 *. 通配符写法）。
var defaultAllowedOrigins = []string{
	"mhy289.dpdns.org",
	"mhy.ink",
	"*.mhy.ink",
	"localhost:5173", // Vite 本地开发
}

// exactOrigins 缓存规范化后的精确白名单集合，避免每次请求重复解析。
// wildcardSuffixes 缓存通配符条目（*.domain）的域名后缀列表。
var exactOrigins, wildcardSuffixes = buildOriginLists()

func buildOriginLists() (map[string]bool, []string) {
	exact := make(map[string]bool, len(defaultAllowedOrigins))
	var wildcard []string
	add := func(list []string) {
		for _, o := range list {
			o = strings.TrimSpace(o)
			if o == "" {
				continue
			}
			if strings.HasPrefix(o, "*.") {
				if s := strings.TrimPrefix(o, "*."); s != "" {
					wildcard = append(wildcard, s)
				}
				continue
			}
			if h := normalizeOrigin(o); h != "" {
				exact[h] = true
			}
		}
	}
	add(defaultAllowedOrigins)
	// 环境变量追加：CORS_ALLOWED_ORIGINS=origin1,origin2,...
	if env := os.Getenv("CORS_ALLOWED_ORIGINS"); env != "" {
		add(strings.Split(env, ","))
	}
	return exact, wildcard
}

// allowedOriginSet 缓存规范化后的白名单集合，避免每次请求重复解析。
var allowedOriginSet = func() map[string]bool {
	set := make(map[string]bool, len(defaultAllowedOrigins))
	add := func(list []string) {
		for _, o := range list {
			if h := normalizeOrigin(o); h != "" {
				set[h] = true
			}
		}
	}
	add(defaultAllowedOrigins)
	// 环境变量覆盖：CORS_ALLOWED_ORIGINS=origin1,origin2,...
	if env := os.Getenv("CORS_ALLOWED_ORIGINS"); env != "" {
		add(strings.Split(env, ","))
	}
	return set
}()

// normalizeOrigin 提取 Origin 中的「主机名 + 端口」用于协议无关的匹配。
// 输入形如 https://mhy289.dpdns.org 或 https://www.mhy.ink:8443，
// 输出形如 mhy289.dpdns.org 或 www.mhy.ink:8443。
// 协议（http/https 及默认端口）不影响匹配结果。
func normalizeOrigin(origin string) string {
	s := strings.TrimSpace(origin)
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "https://")
	// 去掉路径/查询/锚点（正常 Origin 不含这些，此处兜底）
	if i := strings.IndexAny(s, "/?#"); i >= 0 {
		s = s[:i]
	}
	// 去掉默认端口带来的歧义：443 视为 https、80 视为 http，均剥离开来
	if strings.HasSuffix(s, ":443") {
		s = strings.TrimSuffix(s, ":443")
	} else if strings.HasSuffix(s, ":80") {
		s = strings.TrimSuffix(s, ":80")
	}
	return s
}

// isAllowedOrigin 判断请求来源是否在跨域白名单内（协议无关）。
// 先精确匹配（主机名+端口），再尝试通配符后缀（*.domain 匹配任意子域名）。
// 通配匹配带点前缀（"."+suffix），避免 evil-mhy.ink 之类的域名误命中 mhy.ink 后缀。
// 注意：带自定义端口的子域名（如 sub.mhy.ink:8443）不在通配范围内，正常部署用 80/443 无影响。
func isAllowedOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	host := normalizeOrigin(origin)
	if exactOrigins[host] {
		return true
	}
	for _, suffix := range wildcardSuffixes {
		if strings.HasSuffix(host, "."+suffix) {
			return true
		}
	}
	return false
}

// CORS 中间件：前后端分离部署（不同域名/端口）时允许跨域。
//
// 处理策略：
//  1. 仅对「白名单内」的 Origin 回显 Access-Control-Allow-Origin，
//     其他来源不设置跨域头，由浏览器拦截（防止任意网站跨站调用写接口）。
//  2. OPTIONS 预检请求无条件返回 204（无论路径/方法是否注册），
//     确保浏览器预检不会因「路由未匹配」而失败。
//  3. 动态回显前端请求的方法与自定义 Header，而非写死白名单，
//     避免前端使用 PUT/DELETE/PATCH 或自定义请求头时预检失败（405/跨域报错）。
//  4. 对后端方法路由不匹配返回的 405，统一附加完整 CORS 响应头，
//     让前端能正常读取错误信息，而不是因缺 CORS 头被浏览器拦截成跨域错误。
//
// allowMethods 声明后端各路由实际支持的方法集合，用于 405 时精确提示；
// 传空时使用通用兜底值。
func CORS(next http.Handler, allowMethods ...string) http.Handler {
	methods := "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	if len(allowMethods) > 0 {
		methods = ""
		for i, m := range allowMethods {
			if i > 0 {
				methods += ", "
			}
			methods += m
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := isAllowedOrigin(origin)

		// 仅白名单内的 Origin 才设置跨域响应头（含 Cookie 凭证支持）
		if origin != "" && allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			// 允许携带 Cookie / 认证信息（与具体 Origin 搭配，不能与 "*" 同用）
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Vary", "Origin")
		}
		// 无 Origin（同源/非浏览器）或来源不在白名单：均不设置跨域头，
		// 白名单外的浏览器请求会因缺 Allow-Origin 被 CORS 拦截。

		// 1. 无条件响应 OPTIONS 预检请求
		if r.Method == http.MethodOptions {
			// 仅白名单来源需要响应预检；其他来源直接拒绝，避免暴露 CORS 声明
			if !allowed {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			// 回显前端声明的请求方法；未声明时使用后端允许的方法集合
			if rm := r.Header.Get("Access-Control-Request-Method"); rm != "" {
				w.Header().Set("Access-Control-Allow-Methods", rm)
			} else {
				w.Header().Set("Access-Control-Allow-Methods", methods)
			}
			// 回显前端声明的自定义 Header，确保预检通过
			if rh := r.Header.Get("Access-Control-Request-Headers"); rh != "" {
				w.Header().Set("Access-Control-Allow-Headers", rh)
			} else {
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			}
			// 允许前端读取的响应头
			w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")
			// 预检缓存 5 分钟，减少重复预检
			w.Header().Set("Access-Control-Max-Age", "300")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// 2. 包装 ResponseWriter：即使内层返回 405，也确保带上完整 CORS 头
		cw := &corsResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(cw, r)

		// 3. 若内层是 405（方法路由不匹配），补充明确的方法提示
		if cw.status == http.StatusMethodNotAllowed && allowed {
			w.Header().Set("Access-Control-Allow-Methods", methods)
		}
	})
}

// corsResponseWriter 包装 http.ResponseWriter，记录写入的状态码。
type corsResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *corsResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *corsResponseWriter) Write(b []byte) (int, error) {
	return w.ResponseWriter.Write(b)
}
