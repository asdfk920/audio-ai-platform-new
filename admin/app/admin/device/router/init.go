package router

import (
	"go-admin/app/admin/device/apis"
	"go-admin/common/middleware"

	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
)

var (
	deviceRouterNoCheckRole = make([]func(*gin.RouterGroup), 0)
	deviceRouterCheckRole   = make([]func(v1 *gin.RouterGroup, authMiddleware *jwt.GinJWTMiddleware), 0)
)

// InitDeviceRouter 初始化设备相关路由
func InitDeviceRouter(r *gin.Engine, authMiddleware *jwt.GinJWTMiddleware) {
	// 无需认证的路由
	for _, f := range deviceRouterNoCheckRole {
		f(r.Group("/api/v1"))
	}
	// 需要认证的路由
	for _, f := range deviceRouterCheckRole {
		f(r.Group("/api/v1"), authMiddleware)
	}
}

// RegisterPlatformDeviceRouter 注册平台设备路由
func RegisterPlatformDeviceRouter(v1 *gin.RouterGroup, authMiddleware *jwt.GinJWTMiddleware) {
	api := apis.PlatformDevice{}
	iot := apis.IotProduct{}
	// 创建产品短路径：POST /api/v1/product（与 POST /api/v1/platform-device/products 相同）
	v1.POST("/product", authMiddleware.MiddlewareFunc(), middleware.AuthCheckRole(), iot.CreateProduct)
	r := v1.Group("/platform-device").Use(authMiddleware.MiddlewareFunc()).Use(middleware.AuthCheckRole())
	{
		r.GET("/list", api.List)
		// 状态上报历史（固定路径 + query sn，避免部分环境下路径参数 404）
		r.GET("/status-logs", api.StatusLogsList)
		r.POST("/status-logs/manual", api.ManualStatusReport)
		// 无多级 path，避免部分已部署二进制/代理未注册 /status-logs/manual 时出现 HTTP 404
		r.POST("/manual-status-report", api.ManualStatusReport)
		// 产品线（须在 /:sn 之前注册）
		r.GET("/products", iot.ListProducts)
		r.POST("/products", iot.CreateProduct)
		r.GET("/products/:id", iot.GetProduct)
		r.PUT("/products/:id", iot.UpdateProduct)
		r.POST("/products/:id/publish", iot.PublishProduct)
		r.POST("/products/:id/disable", iot.DisableProduct)
		r.POST("", api.Create)
		r.POST("/activate-cloud", api.ActivateCloudAuth)
		r.POST("/activate-cloud-admin", api.ActivateCloudTrusted)
		r.POST("/remote-command", api.RemoteCommand)
		r.POST("/report-status", api.ReportStatusTrigger)
		// 单段 path，与 manual-status-report 同理，避免旧二进制未注册 /report-status 时 HTTP 404
		r.POST("/trigger-report-status", api.ReportStatusTrigger)
		r.POST("/info/update", api.AdminInfoUpdate)
		r.POST("/info/update-batch", api.AdminInfoUpdateBatch)
		r.POST("/diagnosis/start", api.StartDiagnosis)
		r.GET("/diagnosis/result", api.GetDiagnosisResult)
		r.GET("/diagnosis/history", api.GetDiagnosisHistory)
		r.POST("/force-unbind", api.ForceUnbind)
		r.POST("/delete", api.Delete)
		r.GET("/log/list", api.GetDeviceLogList)
		// 设备影子（须在 /:sn 静态路由之前注册）
		r.GET("/shadow", api.ShadowGet)
		r.PUT("/shadow/desired", api.ShadowPutDesired)
		r.GET("/devices/:sn/shadow", api.NormalizedShadowGet)
		r.PUT("/devices/:sn/shadow", api.NormalizedShadowPut)
		r.GET("/devices/:sn/status-logs", api.StatusLogsList)
		// 单条指令执行状态（须在 /:sn 之前）
		r.GET("/instruction/execution", api.InstructionExecution)
		// 指令历史（须在 /:sn 之前）
		r.GET("/instructions", api.InstructionList)
		r.GET("/instructions/:id", api.InstructionDetail)
		r.POST("/instructions/:id/cancel", api.InstructionCancel)
		// 固件包（须在 /:sn 之前；upload 须在 /firmware/:id 之前）
		r.GET("/firmware/list", api.FirmwareList)
		r.POST("/firmware/upload", api.FirmwareUpload)
		r.POST("/firmware/delete", api.FirmwareDelete)
		r.POST("/firmware/update", api.FirmwareUpdate)
		r.GET("/firmware/history", api.FirmwareHistory)
		r.GET("/firmware/:id", api.FirmwareDetail)
		// OTA 任务
		r.GET("/ota-task/list", api.OTATaskList)
		r.GET("/ota-task/detail", api.OTATaskDetail)
		r.POST("/ota-task/cancel", api.OTATaskCancel)
		r.GET("/ota-task/progress", api.OTATaskProgress)
		// 固定路径必须注册在 GET /:sn 之前，否则单段路径会被当成 sn（如 /stats → Detail("stats")）
		r.GET("/stats", api.Summary)
		r.GET("/enums", api.Enum)
		r.GET("/productkeys", api.ProductKeys)
		r.GET("/import/template", api.ImportTemplate)
		r.POST("/import/jobs", api.ImportJobCreate)
		r.GET("/import/jobs/:id", api.ImportJobGet)
		r.GET("/import/jobs/:id/download", api.ImportJobDownload)
		r.POST("/import", api.Import)
		r.PUT("/status", api.BatchStatus)
		// 详情 query 形式（须在 /:sn 之前，否则 /detail 会被当成 sn）
		r.GET("/detail", api.Detail)
		// 须在 GET /:sn 之前：设备定时状态上报历史（与 PUT /:sn/status 同风格路径）
		r.GET("/:sn/status-logs", api.StatusLogsList)
		r.GET("/:sn", api.Detail)
		r.PUT("/:sn/status", api.SetStatus)
		r.POST("/:sn/unbind", api.Unbind)
		r.POST("/:sn/command", api.Command)
		r.POST("/:sn/ota", api.OTA)
	}
}

func init() {
	deviceRouterCheckRole = append(deviceRouterCheckRole, RegisterPlatformDeviceRouter)
}
