package router

// sys_admin 路由：控制台管理员模块的完整 CRUD + 批量删除 + 重置密码 + 安全策略
// 挂载于 /api/v1/sys-admin/*，均需 JWT 认证并走 Casbin 角色校验；
// 另有 /api/admin/account/change-password 自助改密，在 sys_router.go 手动挂载以与
// 冷启动注册 /api/admin/register 保持同一命名空间。

import (
	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"

	"go-admin/app/admin/apis"
	"go-admin/common/middleware"
)

func init() {
	routerCheckRole = append(routerCheckRole, registerSysAdminRouter)
}

func registerSysAdminRouter(v1 *gin.RouterGroup, authMiddleware *jwt.GinJWTMiddleware) {
	api := apis.SysAdmin{}

	// /api/v1/sys-admin
	r := v1.Group("/sys-admin").
		Use(authMiddleware.MiddlewareFunc()).
		Use(middleware.AuthCheckRole())
	{
		r.GET("", api.AdminList)
		r.GET("/:id", api.AdminDetail)
		r.POST("", api.AdminCreate)
		r.PUT("/:id", api.AdminUpdate)
		r.DELETE("/:id", api.AdminDelete)

		// 批量删除：请求体 {"user_ids":[...]}，最多 100 条
		r.POST("/batch-delete", api.AdminBatchDelete)

		// 状态切换 / 重置密码 / 安全策略 / 强制改密标志
		r.PUT("/:id/status", api.AdminStatus)
		r.PUT("/:id/password", api.AdminResetPassword)
		r.PUT("/:id/security", api.AdminSetSecurity)
		r.PUT("/:id/must-change-password", api.AdminSetForceChange)
	}
}
