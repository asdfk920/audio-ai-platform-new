package router

import (
	"go-admin/app/admin/user/apis"
	"go-admin/common/middleware"

	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
)

var (
	userRouterNoCheckRole = make([]func(*gin.RouterGroup), 0)
	userRouterCheckRole   = make([]func(v1 *gin.RouterGroup, authMiddleware *jwt.GinJWTMiddleware), 0)
)

// InitUserRouter 初始化用户相关路由
func InitUserRouter(r *gin.Engine, authMiddleware *jwt.GinJWTMiddleware) {
	// 无需认证的路由
	for _, f := range userRouterNoCheckRole {
		f(r.Group("/api/v1"))
	}
	// 需要认证的路由
	for _, f := range userRouterCheckRole {
		f(r.Group("/api/v1"), authMiddleware)
	}
}

// RegisterPlatformUserListRouter 注册平台用户列表路由
func RegisterPlatformUserListRouter(v1 *gin.RouterGroup, authMiddleware *jwt.GinJWTMiddleware) {
	api := apis.PlatformUserList{}
	r := v1.Group("/platform-user").Use(authMiddleware.MiddlewareFunc()).Use(middleware.AuthCheckRole())
	{
		r.GET("/list", api.GetPlatformUserList) // 用户列表
		// 固定路径须注册在 /:userId 之前，否则会被当成 userId（如 "status"）
		r.PUT("/status", api.UpdateUserStatus)       // 禁用/启用
		r.PUT("/vip-level", api.UpdateUserVipLevel) // 会员等级
		r.GET("/:userId", api.GetPlatformUserInfo)   // 用户详情
		r.POST("", api.CreatePlatformUser)           // 创建用户
		r.PUT("/:userId", api.UpdatePlatformUser)       // 更新用户基础信息
		r.PUT("/:userId/roles", api.SetPlatformUserRoles) // 覆盖式设置角色（仅超级管理员）
		r.DELETE("/:userId", api.DeletePlatformUser)    // 删除用户
	}
}

// RegisterPlatformMemberListRouter 注册平台会员列表路由
func RegisterPlatformMemberListRouter(v1 *gin.RouterGroup, authMiddleware *jwt.GinJWTMiddleware) {
	api := apis.PlatformMemberList{}
	stub := apis.PlatformMemberStub{}
	r := v1.Group("/platform-member").Use(authMiddleware.MiddlewareFunc()).Use(middleware.AuthCheckRole())
	{
		r.GET("/list", api.GetPlatformMemberList) // 会员列表
		// 扩展/占位路由须注册在 /:userId 之前，否则会被当成 userId
		r.GET("/user-members/summary", stub.UserMembersSummary)
		r.GET("/user-members", stub.UserMembersList)
		r.GET("/levels", stub.Levels)
		r.POST("/levels", stub.UpsertLevel)
		r.PUT("/levels", stub.UpsertLevel)
		r.DELETE("/levels/:id", stub.DeleteLevel)
		r.POST("/levels/batch", stub.BatchLevels)
		r.POST("/levels/reorder", stub.ReorderLevels)
		r.GET("/benefits", stub.Benefits)
		r.POST("/benefits", stub.UpsertBenefit)
		r.PUT("/benefits", stub.UpsertBenefit)
		r.DELETE("/benefits/:id", stub.DeleteBenefit)
		r.POST("/benefits/batch", stub.BatchBenefits)
		r.GET("/level-benefits", stub.LevelBenefits)
		r.PUT("/level-benefits/:levelCode", stub.SetLevelBenefits)
		r.GET("/user-member", stub.UserMember)
		r.PUT("/user-member", stub.UpsertUserMember)
		r.GET("/user-member/detail", stub.UserMemberDetail)
		r.POST("/user-member/batch", stub.BatchUserMembers)
		r.POST("/freeze", api.FreezePlatformMember)                // 冻结会员
		r.POST("/unfreeze", api.UnfreezePlatformMember)            // 解冻会员
		r.POST("/right-config", api.SavePlatformMemberRightConfig) // 保存权益配置
		r.GET("/:userId", api.GetPlatformMemberDetail)             // 会员详情
		r.PUT("", api.UpdatePlatformMember)                        // 更新会员信息
	}
}

// RegisterUserRealNameAuditRouter 注册实名认证审核路由
func RegisterUserRealNameAuditRouter(v1 *gin.RouterGroup, authMiddleware *jwt.GinJWTMiddleware) {
	api := apis.UserRealNameAudit{}
	r := v1.Group("/user-realname").Use(authMiddleware.MiddlewareFunc()).Use(middleware.AuthCheckRole())
	{
		r.GET("/list", api.GetRealNameList)              // 实名认证列表
		r.GET("/detail/:user_id", api.GetRealNameDetail) // 实名认证详情
		r.POST("/audit", api.AuditRealName)              // 审核实名认证
	}
}

func init() {
	userRouterCheckRole = append(userRouterCheckRole, RegisterPlatformUserListRouter)
	userRouterCheckRole = append(userRouterCheckRole, RegisterPlatformMemberListRouter)
	userRouterCheckRole = append(userRouterCheckRole, RegisterUserRealNameAuditRouter)
}
