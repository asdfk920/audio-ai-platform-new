package middleware

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/audio-ai-platform/gateway/internal/config"
	"github.com/gin-gonic/gin"
)

// RouteProxy 路由代理中间件
type RouteProxy struct {
	routes []config.RouteConfig
}

// NewRouteProxy 创建路由代理
func NewRouteProxy(routes []config.RouteConfig) *RouteProxy {
	return &RouteProxy{
		routes: routes,
	}
}

// Handler 路由代理处理函数
func (rp *RouteProxy) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// 查找匹配的路由
		var targetRoute *config.RouteConfig
		for _, route := range rp.routes {
			if strings.HasPrefix(path, route.PathPrefix) {
				targetRoute = &route
				break
			}
		}

		if targetRoute == nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "路由未找到",
				"path":  path,
			})
			c.Abort()
			return
		}

		// 设置服务名称到上下文中
		c.Set("service_name", targetRoute.Name)

		// 创建反向代理
		targetURL, err := url.Parse(targetRoute.Target)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":  "目标服务地址无效",
				"target": targetRoute.Target,
			})
			c.Abort()
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(targetURL)

		// 设置超时
		if targetRoute.Timeout > 0 {
			httpClient := &http.Client{
				Timeout: targetRoute.Timeout,
			}
			proxy.Transport = httpClient.Transport
		}

		// 修改请求头
		proxy.ModifyResponse = func(resp *http.Response) error {
			// 添加网关标识
			resp.Header.Set("X-Gateway", "audio-ai-platform-gateway")
			resp.Header.Set("X-Gateway-Service", targetRoute.Name)
			return nil
		}

		// 执行代理
		proxy.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
}

// RouteInfo 路由信息结构
type RouteInfo struct {
	PathPrefix string        `json:"path_prefix"`
	Target     string        `json:"target"`
	Timeout    time.Duration `json:"timeout"`
	Name       string        `json:"name"`
}

// GetRoutesInfo 获取路由信息（用于健康检查）
func (rp *RouteProxy) GetRoutesInfo() []RouteInfo {
	var routes []RouteInfo
	for _, route := range rp.routes {
		routes = append(routes, RouteInfo{
			PathPrefix: route.PathPrefix,
			Target:     route.Target,
			Timeout:    route.Timeout,
			Name:       route.Name,
		})
	}
	return routes
}

// HealthCheck 健康检查处理函数
func (rp *RouteProxy) HealthCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"version": "1.0.0",
			"routes":  rp.GetRoutesInfo(),
		})
	}
}
