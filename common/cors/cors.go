// Package cors 提供 go-zero rest.Server 可用的 HTTP CORS 中间件（浏览器跨域）。
package cors

import (
	"net/http"
	"strconv"
	"strings"
)

// Config 对应各微服务 etc/*.yaml 中的可选 CORS 段。
// Disabled 为 true 时不注入任何 CORS 头（默认 false，即启用）。
// AllowOrigins 为空或包含 "*" 时等同允许任意 Origin（此时忽略 AllowCredentials）。
type Config struct {
	Disabled         bool     `json:",optional"`
	AllowOrigins     []string `json:",optional"`
	AllowMethods     []string `json:",optional"`
	AllowHeaders     []string `json:",optional"`
	ExposeHeaders    []string `json:",optional"`
	MaxAge           int      `json:",optional"`
	AllowCredentials bool     `json:",optional"`
}

type resolved struct {
	wildcard         bool
	originSet        map[string]struct{}
	allowCredentials bool
	methods          string
	headers          string
	expose           string
	maxAge           string
}

func resolve(cfg Config) resolved {
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	if len(cfg.AllowMethods) > 0 {
		methods = cfg.AllowMethods
	}
	headers := []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"}
	if len(cfg.AllowHeaders) > 0 {
		headers = cfg.AllowHeaders
	}
	maxAge := cfg.MaxAge
	if maxAge <= 0 {
		maxAge = 86400
	}

	origins := cfg.AllowOrigins
	wildcard := len(origins) == 0
	set := make(map[string]struct{})
	for _, o := range origins {
		o = strings.TrimSpace(o)
		if o == "*" {
			wildcard = true
			set = nil
			break
		}
		if o != "" {
			set[o] = struct{}{}
		}
	}

	allowCred := cfg.AllowCredentials
	if wildcard {
		allowCred = false
	}

	return resolved{
		wildcard:         wildcard,
		originSet:        set,
		allowCredentials: allowCred,
		methods:          strings.Join(methods, ", "),
		headers:          strings.Join(headers, ", "),
		expose:           strings.Join(cfg.ExposeHeaders, ", "),
		maxAge:           strconv.Itoa(maxAge),
	}
}

func (r resolved) allowsOrigin(origin string) bool {
	if r.wildcard {
		return true
	}
	if origin == "" {
		return false
	}
	_, ok := r.originSet[origin]
	return ok
}

// Middleware 与 go-zero rest.Server.Use 所需的签名一致（func(http.HandlerFunc) http.HandlerFunc）。
func Middleware(cfg Config) func(http.HandlerFunc) http.HandlerFunc {
	if cfg.Disabled {
		return func(next http.HandlerFunc) http.HandlerFunc {
			return next
		}
	}

	r := resolve(cfg)

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request) {
			origin := req.Header.Get("Origin")

			writeHeaders := func() {
				w.Header().Set("Access-Control-Allow-Methods", r.methods)
				w.Header().Set("Access-Control-Allow-Headers", r.headers)
				w.Header().Set("Access-Control-Max-Age", r.maxAge)
				if r.expose != "" {
					w.Header().Set("Access-Control-Expose-Headers", r.expose)
				}
				if r.wildcard {
					w.Header().Set("Access-Control-Allow-Origin", "*")
					return
				}
				if origin != "" && r.allowsOrigin(origin) {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Add("Vary", "Origin")
					if r.allowCredentials {
						w.Header().Set("Access-Control-Allow-Credentials", "true")
					}
				}
			}

			if req.Method == http.MethodOptions {
				if !r.wildcard && origin != "" && !r.allowsOrigin(origin) {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				writeHeaders()
				w.WriteHeader(http.StatusNoContent)
				return
			}

			writeHeaders()
			next(w, req)
		}
	}
}
