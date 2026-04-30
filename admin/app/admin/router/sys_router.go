package router

import (
	"go-admin/app/admin/apis"
	"mime"

	"github.com/go-admin-team/go-admin-core/sdk/config"

	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/ws"
	ginSwagger "github.com/swaggo/gin-swagger"

	swaggerfiles "github.com/swaggo/files"

	"go-admin/common/middleware"
	"go-admin/common/middleware/handler"
	_ "go-admin/docs/admin"
)

func InitSysRouter(r *gin.Engine, authMiddleware *jwt.GinJWTMiddleware) *gin.RouterGroup {
	g := r.Group("")
	sysBaseRouter(g)
	// 静态文件
	sysStaticFileRouter(g)
	// swagger；注意：生产环境可以注释掉
	if config.ApplicationConfig.Mode != "prod" {
		sysSwaggerRouter(g)
	}
	// 需要认证
	sysCheckRoleRouterInit(g, authMiddleware)
	return g
}

func sysBaseRouter(r *gin.RouterGroup) {

	go ws.WebsocketManager.Start()
	go ws.WebsocketManager.SendService()
	go ws.WebsocketManager.SendAllService()

	if config.ApplicationConfig.Mode != "prod" {
		r.GET("/", apis.GoAdmin)
	}
	r.GET("/info", handler.Ping)
}

func sysStaticFileRouter(r *gin.RouterGroup) {
	err := mime.AddExtensionType(".js", "application/javascript")
	if err != nil {
		return
	}
	r.Static("/static", "./static")
	if config.ApplicationConfig.Mode != "prod" {
		r.Static("/form-generator", "./static/form-generator")
	}
}

func sysSwaggerRouter(r *gin.RouterGroup) {
	r.GET("/swagger/admin/*any", ginSwagger.WrapHandler(swaggerfiles.NewHandler(), ginSwagger.InstanceName("admin")))
}

func sysCheckRoleRouterInit(r *gin.RouterGroup, authMiddleware *jwt.GinJWTMiddleware) {
	wss := r.Group("").Use(authMiddleware.MiddlewareFunc())
	{
		wss.GET("/ws/:id/:channel", ws.WebsocketManager.WsClient)
		wss.GET("/wslogout/:id/:channel", ws.WebsocketManager.UnWsClient)
	}

	v1 := r.Group("/api/v1")
	{
		v1.POST("/login", authMiddleware.LoginHandler)
		// Refresh time can be longer than token timeout
		v1.GET("/refresh_token", authMiddleware.RefreshHandler)
	}
	// /api/v1/login 与 /api/admin/login 均只校验 public.sys_admin；sys_user 视图为 sys_admin 的兼容投影（见迁移 081），不读 public.users
	adminAPI := r.Group("/api/admin")
	{
		// 专用后台冷启动注册接口：与 C 端用户注册路径彻底分离（旧 /api/v1/setup/* 已下线）
		setupAPI := apis.Setup{}
		adminAPI.GET("/setup/status", setupAPI.GetStatus)
		adminAPI.POST("/setup/register", setupAPI.PostRegister)

		// 管理员账号注册模块：公开接口，仅写入 public.sys_admin，与 C 端 users 完全隔离
		adminAccountAPI := apis.AdminAccount{}
		adminAPI.POST("/account/register", adminAccountAPI.Register)

		adminAPI.POST("/login", handler.AdminLoginHandler(authMiddleware))
		adminAPI.GET("/refresh_token", authMiddleware.RefreshHandler)
		apiAdmin := apis.SysAdmin{}
		adminAuth := adminAPI.Group("").Use(authMiddleware.MiddlewareFunc()).Use(middleware.AuthCheckRole())
		{
			adminAuth.POST("/register", apiAdmin.AdminRegister)
			// 自助改密：首次登录后或日常修改密码均走此路径，仅需 JWT，不涉及 Casbin 资源粒度控制。
			adminAuth.POST("/account/change-password", apiAdmin.AdminChangePasswordSelf)
			// 当前登录管理员资料：从 Token 解析 userId，不需要前端传 user_id
			adminAuth.GET("/profile", apiAdmin.AdminProfileGet)
			adminAuth.PUT("/profile", apiAdmin.AdminProfileUpdate)
		}
	}
	registerBaseRouter(v1, authMiddleware)
}

func registerBaseRouter(v1 *gin.RouterGroup, authMiddleware *jwt.GinJWTMiddleware) {
	api := apis.SysMenu{}
	api2 := apis.SysDept{}
	v1auth := v1.Group("").Use(authMiddleware.MiddlewareFunc()).Use(middleware.AuthCheckRole())
	{
		v1auth.GET("/roleMenuTreeselect/:roleId", api.GetMenuTreeSelect)
		//v1.GET("/menuTreeselect", api.GetMenuTreeSelect)
		v1auth.GET("/roleDeptTreeselect/:roleId", api2.GetDeptTreeRoleSelect)
		v1auth.POST("/logout", handler.LogOut)
	}
}
