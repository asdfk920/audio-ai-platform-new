package router

import (
	"os"

	"github.com/gin-gonic/gin"
	log "github.com/go-admin-team/go-admin-core/logger"
	"github.com/go-admin-team/go-admin-core/sdk"
	contentRouter "go-admin/app/admin/content/router"
	deviceRouter "go-admin/app/admin/device/router"
	userRouter "go-admin/app/admin/user/router"
	common "go-admin/common/middleware"
)

// InitRouter 路由初始化，不要怀疑，这里用到了
func InitRouter() {
	var r *gin.Engine
	h := sdk.Runtime.GetEngine()
	if h == nil {
		log.Fatal("not found engine...")
		os.Exit(-1)
	}
	switch h.(type) {
	case *gin.Engine:
		r = h.(*gin.Engine)
	default:
		log.Fatal("not support other engine")
		os.Exit(-1)
	}

	// the jwt middleware
	authMiddleware, err := common.AuthInit()
	if err != nil {
		log.Fatalf("JWT Init Error, %s", err.Error())
	}

	// 注册系统路由
	InitSysRouter(r, authMiddleware)

	// 平台业务：用户 / 设备 / 内容
	userRouter.InitUserRouter(r, authMiddleware)
	deviceRouter.InitDeviceRouter(r, authMiddleware)
	contentRouter.InitContentRouter(r, authMiddleware)

	// 注册业务路由（演示/示例）
	InitExamplesRouter(r, authMiddleware)
}
