// Package middleware 提供 HTTP 中间件。
package middleware

import (
	"net/http"
)

// CORS 中间件：前后端分离部署（不同域名/端口）时允许跨域。
//
// 处理策略：
//  1. OPTIONS 预检请求无条件返回 204（无论路径/方法是否注册），
//     确保浏览器预检不会因「路由未匹配」而失败。
//  2. 动态回显前端请求的方法与自定义 Header，而非写死白名单，
//     避免前端使用 PUT/DELETE/PATCH 或自定义请求头时预检失败（405/跨域报错）。
//  3. 对后端方法路由不匹配返回的 405，统一附加完整 CORS 响应头，
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
		// 有 Origin 才设置跨域头；无 Origin（同源/非浏览器）保持默认行为
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			// 允许携带 Cookie / 认证信息（与具体 Origin 搭配，不能与 "*" 同用）
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Vary", "Origin")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		// 1. 无条件响应 OPTIONS 预检请求
		if r.Method == http.MethodOptions {
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
		if cw.status == http.StatusMethodNotAllowed {
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
