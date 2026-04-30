package router

import (
	"go-admin/app/admin/apis"
	"go-admin/common/middleware"

	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
)

func init() {
	routerCheckRole = append(routerCheckRole, registerPlatformRbacRouter)
}

func registerPlatformRbacRouter(v1 *gin.RouterGroup, authMiddleware *jwt.GinJWTMiddleware) {
	api := apis.PlatformRbac{}
	r := v1.Group("/platform-rbac").Use(authMiddleware.MiddlewareFunc()).Use(middleware.AuthCheckRole())
	{
		r.GET("/matrix", api.Matrix)
		r.GET("/roles", api.Roles)
		r.PUT("/roles/:roleKey", api.UpdateRole)
	}
}
