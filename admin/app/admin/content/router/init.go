package router

import (
	"go-admin/app/admin/content/apis"
	"go-admin/common/middleware"

	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
)

var (
	contentRouterNoCheckRole = make([]func(*gin.RouterGroup), 0)
	contentRouterCheckRole   = make([]func(v1 *gin.RouterGroup, authMiddleware *jwt.GinJWTMiddleware), 0)
)

// InitContentRouter 初始化内容管理路由
func InitContentRouter(r *gin.Engine, authMiddleware *jwt.GinJWTMiddleware) {
	for _, f := range contentRouterNoCheckRole {
		f(r.Group("/api/v1"))
	}
	for _, f := range contentRouterCheckRole {
		f(r.Group("/api/v1"), authMiddleware)
	}
}

// RegisterPlatformContentRouter 平台内容管理（RBAC: content_mgmt）
func RegisterPlatformContentRouter(v1 *gin.RouterGroup, authMiddleware *jwt.GinJWTMiddleware) {
	api := apis.PlatformContent{}
	r := v1.Group("/platform-content")
	r.Use(authMiddleware.MiddlewareFunc(), middleware.AuthCheckRole())
	{
		// --- 转发 services/content 接口（与 internal/handler 路由对齐；需 CONTENT_SERVICE_BASE_URL）---
		cg := r.Group("/content")
		{
			cg.GET("/list", api.ProxyContentAppList)
			cg.GET("/recommend", api.ProxyContentAppRecommend)
			cg.POST("/auth", api.ProxyContentAppAuth)
			cg.GET("/status", api.ProxyContentAppStatus)
			cg.GET("/detail", api.ProxyContentAppDetail)
			cg.GET("/audio/url", api.ProxyContentAppAudioURL)
			cg.POST("/play/report", api.ProxyContentAppPlayReport)
			cg.POST("/favorite/add", api.ProxyContentAppFavoriteAdd)
			cg.POST("/favorite/cancel", api.ProxyContentAppFavoriteCancel)
			cg.GET("/favorite/list", api.ProxyContentAppFavoriteList)
			cg.POST("/download/apply", api.ProxyContentAppDownloadApply)
			cg.GET("/download/record", api.ProxyContentAppDownloadRecord)
			cg.POST("/upload", api.ProxyContentAppUpload)
			cg.POST("/catalog/upload", api.ProxyContentAppCatalogUpload)
		}
		og := r.Group("/ops")
		{
			og.POST("/bump-list-cache", api.OpsBumpContentListCache)
			og.POST("/del-detail-cache", api.OpsDelContentDetailCache)
		}

		// 后台内容分页列表（可筛选/分页/排序；默认不含已删除）
		r.GET("/list", api.List)
		// 后台内容详情（用于编辑页展示全量字段 + 分类名 + 标签）
		r.GET("/detail", api.Detail)
		// 后台新增内容（multipart/form-data；支持直接传 cover/audio 文件或 cover_url/audio_url）
		r.POST("/add", api.Add)
		// 后台编辑内容（multipart/form-data；支持部分更新与重新上传 cover/audio 文件）
		r.POST("/update", api.Update)
		// 后台上架内容（状态流转：0/2 -> 1；1 幂等）
		r.POST("/online", api.Online)
		// 后台下架内容（状态流转：非删除 -> 2；2 幂等）
		r.POST("/offline", api.Offline)
		// 后台删除内容（软删：is_deleted=1；幂等）
		r.POST("/delete", api.Delete)
		// 后台恢复已删除内容（幂等；需 content:restore 权限）
		r.POST("/restore", api.Restore)
	}
}

func init() {
	contentRouterCheckRole = append(contentRouterCheckRole, RegisterPlatformContentRouter)
}
