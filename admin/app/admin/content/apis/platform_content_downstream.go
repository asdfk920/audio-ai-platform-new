package apis

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// 以下接口对应 services/content/internal/handler 中的路由，
// 经管理端统一出口转发到内容微服务（需配置 CONTENT_SERVICE_BASE_URL）。
// 需要用户 JWT 的接口：请求头携带 App 侧 Authorization（与内容服务 Auth.AccessSecret 一致），
// 管理端 JWT 仅用于 RBAC，不会自动替换为 App token。

func contentServiceBaseURL() (string, bool) {
	base := strings.TrimSpace(os.Getenv("CONTENT_SERVICE_BASE_URL"))
	if base == "" {
		return "", false
	}
	return strings.TrimRight(base, "/"), true
}

func contentInternalSecret() string {
	return strings.TrimSpace(os.Getenv("CONTENT_INTERNAL_SECRET"))
}

func (e PlatformContent) serveContentReverseProxy(c *gin.Context, upstreamPath string) {
	if err := e.MakeContext(c).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	base, ok := contentServiceBaseURL()
	if !ok {
		e.Error(503, errors.New("CONTENT_SERVICE_BASE_URL empty"), "未配置 CONTENT_SERVICE_BASE_URL，无法转发到内容服务")
		return
	}
	target, err := url.Parse(base + upstreamPath)
	if err != nil {
		e.Error(500, err, "内容服务地址解析失败")
		return
	}

	proxy := &httputil.ReverseProxy{
		Director: func(out *http.Request) {
			out.URL = target
			out.URL.RawQuery = c.Request.URL.RawQuery
			out.Method = c.Request.Method
			out.Host = target.Host
			out.Proto = "HTTP/1.1"
			out.ProtoMajor = 1
			out.ProtoMinor = 1
			out.Header = cloneForwardHeaders(c)
			out.Header.Del("X-Internal-Secret")
			out.ContentLength = c.Request.ContentLength
			out.Body = c.Request.Body
			out.GetBody = c.Request.GetBody
		},
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			ResponseHeaderTimeout: 60 * time.Second,
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"code":502,"msg":"内容服务不可达","data":null}`))
		},
	}
	proxy.ServeHTTP(c.Writer, c.Request)
}

func cloneForwardHeaders(c *gin.Context) http.Header {
	h := make(http.Header)
	for _, k := range []string{
		"Authorization",
		"Content-Type",
		"Accept",
		"Accept-Language",
		"User-Agent",
	} {
		if v := strings.TrimSpace(c.GetHeader(k)); v != "" {
			h.Set(k, v)
		}
	}
	return h
}

// --- 与 content 服务「无 JWT」路由块一致：GET/POST /api/v1/content/... ---

// ProxyContentAppList GET /api/v1/platform-content/content/list
// @Summary 转发：内容分页列表（可选 App Authorization）
// @Tags 平台内容
// @Router /api/v1/platform-content/content/list [get]
// @Security Bearer
func (e PlatformContent) ProxyContentAppList(c *gin.Context) {
	e.serveContentReverseProxy(c, "/api/v1/content/list")
}

// ProxyContentAppRecommend GET /api/v1/platform-content/content/recommend
// @Summary 转发：首页推荐
// @Tags 平台内容
// @Router /api/v1/platform-content/content/recommend [get]
// @Security Bearer
func (e PlatformContent) ProxyContentAppRecommend(c *gin.Context) {
	e.serveContentReverseProxy(c, "/api/v1/content/recommend")
}

// ProxyContentAppAuth POST /api/v1/platform-content/content/auth
// @Summary 转发：统一内容鉴权
// @Tags 平台内容
// @Router /api/v1/platform-content/content/auth [post]
// @Security Bearer
func (e PlatformContent) ProxyContentAppAuth(c *gin.Context) {
	e.serveContentReverseProxy(c, "/api/v1/content/auth")
}

// --- 与 content 服务「JWT」路由块一致（须带 App Bearer）---

// ProxyContentAppStatus GET /api/v1/platform-content/content/status
// @Summary 转发：处理状态
// @Tags 平台内容
// @Router /api/v1/platform-content/content/status [get]
// @Security Bearer
func (e PlatformContent) ProxyContentAppStatus(c *gin.Context) {
	e.serveContentReverseProxy(c, "/api/v1/content/status")
}

// ProxyContentAppDetail GET /api/v1/platform-content/content/detail
// @Summary 转发：内容详情
// @Tags 平台内容
// @Router /api/v1/platform-content/content/detail [get]
// @Security Bearer
func (e PlatformContent) ProxyContentAppDetail(c *gin.Context) {
	e.serveContentReverseProxy(c, "/api/v1/content/detail")
}

// ProxyContentAppAudioURL GET /api/v1/platform-content/content/audio/url
// @Summary 转发：播放地址
// @Tags 平台内容
// @Router /api/v1/platform-content/content/audio/url [get]
// @Security Bearer
func (e PlatformContent) ProxyContentAppAudioURL(c *gin.Context) {
	e.serveContentReverseProxy(c, "/api/v1/content/audio/url")
}

// ProxyContentAppPlayReport POST /api/v1/platform-content/content/play/report
// @Summary 转发：播放上报
// @Tags 平台内容
// @Router /api/v1/platform-content/content/play/report [post]
// @Security Bearer
func (e PlatformContent) ProxyContentAppPlayReport(c *gin.Context) {
	e.serveContentReverseProxy(c, "/api/v1/content/play/report")
}

// ProxyContentAppFavoriteAdd POST /api/v1/platform-content/content/favorite/add
// @Summary 转发：收藏
// @Tags 平台内容
// @Router /api/v1/platform-content/content/favorite/add [post]
// @Security Bearer
func (e PlatformContent) ProxyContentAppFavoriteAdd(c *gin.Context) {
	e.serveContentReverseProxy(c, "/api/v1/content/favorite/add")
}

// ProxyContentAppFavoriteCancel POST /api/v1/platform-content/content/favorite/cancel
// @Summary 转发：取消收藏
// @Tags 平台内容
// @Router /api/v1/platform-content/content/favorite/cancel [post]
// @Security Bearer
func (e PlatformContent) ProxyContentAppFavoriteCancel(c *gin.Context) {
	e.serveContentReverseProxy(c, "/api/v1/content/favorite/cancel")
}

// ProxyContentAppFavoriteList GET /api/v1/platform-content/content/favorite/list
// @Summary 转发：收藏列表
// @Tags 平台内容
// @Router /api/v1/platform-content/content/favorite/list [get]
// @Security Bearer
func (e PlatformContent) ProxyContentAppFavoriteList(c *gin.Context) {
	e.serveContentReverseProxy(c, "/api/v1/content/favorite/list")
}

// ProxyContentAppDownloadApply POST /api/v1/platform-content/content/download/apply
// @Summary 转发：下载申请
// @Tags 平台内容
// @Router /api/v1/platform-content/content/download/apply [post]
// @Security Bearer
func (e PlatformContent) ProxyContentAppDownloadApply(c *gin.Context) {
	e.serveContentReverseProxy(c, "/api/v1/content/download/apply")
}

// ProxyContentAppDownloadRecord GET /api/v1/platform-content/content/download/record
// @Summary 转发：下载记录
// @Tags 平台内容
// @Router /api/v1/platform-content/content/download/record [get]
// @Security Bearer
func (e PlatformContent) ProxyContentAppDownloadRecord(c *gin.Context) {
	e.serveContentReverseProxy(c, "/api/v1/content/download/record")
}

// ProxyContentAppUpload POST /api/v1/platform-content/content/upload
// @Summary 转发：获取上传地址（占位）
// @Tags 平台内容
// @Router /api/v1/platform-content/content/upload [post]
// @Security Bearer
func (e PlatformContent) ProxyContentAppUpload(c *gin.Context) {
	e.serveContentReverseProxy(c, "/api/v1/content/upload")
}

// ProxyContentAppCatalogUpload POST /api/v1/platform-content/content/catalog/upload
// @Summary 转发：运营 catalog 入库（multipart）
// @Tags 平台内容
// @Router /api/v1/platform-content/content/catalog/upload [post]
// @Security Bearer
func (e PlatformContent) ProxyContentAppCatalogUpload(c *gin.Context) {
	e.serveContentReverseProxy(c, "/api/v1/content/catalog/upload")
}

// --- 运维：由管理端注入 X-Internal-Secret，不暴露给浏览器 ---

// OpsBumpContentListCache POST /api/v1/platform-content/ops/bump-list-cache
// @Summary 运维：失效列表/推荐热点缓存（等同 content internal bump）
// @Tags 平台内容
// @Router /api/v1/platform-content/ops/bump-list-cache [post]
// @Security Bearer
func (e PlatformContent) OpsBumpContentListCache(c *gin.Context) {
	e.internalContentPOST(c, "/api/v1/content/internal/bump-list-cache", nil, false)
}

// OpsDelContentDetailCache POST /api/v1/platform-content/ops/del-detail-cache
// @Summary 运维：清理单条详情缓存（JSON body: content_id）
// @Tags 平台内容
// @Router /api/v1/platform-content/ops/del-detail-cache [post]
// @Security Bearer
func (e PlatformContent) OpsDelContentDetailCache(c *gin.Context) {
	if err := e.MakeContext(c).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		e.Error(400, err, "读取请求体失败")
		return
	}
	e.internalContentPOST(c, "/api/v1/content/internal/del-detail-cache", body, true)
}

func (e PlatformContent) internalContentPOST(c *gin.Context, path string, body []byte, setJSON bool) {
	if err := e.MakeContext(c).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	base, ok := contentServiceBaseURL()
	if !ok {
		e.Error(503, errors.New("CONTENT_SERVICE_BASE_URL empty"), "未配置 CONTENT_SERVICE_BASE_URL")
		return
	}
	secret := contentInternalSecret()
	if secret == "" {
		e.Error(503, errors.New("CONTENT_INTERNAL_SECRET empty"), "未配置 CONTENT_INTERNAL_SECRET")
		return
	}
	var rdr io.Reader
	if len(body) > 0 {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, base+path, rdr)
	if err != nil {
		e.Error(500, err, "构造请求失败")
		return
	}
	req.Header.Set("X-Internal-Secret", secret)
	if setJSON {
		req.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		e.Error(502, err, "调用内容服务失败")
		return
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		e.Error(502, err, "读取内容服务响应失败")
		return
	}
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/json; charset=utf-8"
	}
	c.Data(resp.StatusCode, ct, respBody)
}
