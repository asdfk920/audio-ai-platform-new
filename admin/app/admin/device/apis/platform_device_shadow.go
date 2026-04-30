package apis

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth/user"

	"go-admin/app/admin/device/service"
)

// ShadowGet 查询设备影子（Redis 实时快照 + PG 持久化 reported/desired，并计算 delta）
// @Summary 设备影子查询
// @Tags 平台设备-影子
// @Param sn query string true "设备 SN"
// @Router /api/v1/platform-device/shadow [get]
// @Security Bearer
func (e PlatformDevice) ShadowGet(c *gin.Context) {
	sn := strings.TrimSpace(c.Query("sn"))
	if sn == "" {
		e.Error(400, errors.New("sn required"), "请传 sn")
		return
	}
	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	out, err := svc.GetDeviceShadow(sn)
	if err != nil {
		if errors.Is(err, service.ErrPlatformDeviceNotFound) {
			e.Error(404, err, "设备不存在")
			return
		}
		if errors.Is(err, service.ErrPlatformDeviceInvalid) {
			e.Error(400, err, "参数无效")
			return
		}
		e.Logger.Error(err)
		e.Error(500, err, "查询失败")
		return
	}
	e.OK(out, "ok")
}

type shadowDesiredReq struct {
	Sn      string          `json:"sn"`
	Desired json.RawMessage `json:"desired"`
	Merge   bool            `json:"merge"` // 与库中已有 desired 合并；默认 false 为全量覆盖
}

// ShadowPutDesired 更新设备影子期望状态 desired，计算 delta，写 Redis；设备在线时 MQTT 下发 delta
// @Summary 更新设备影子 desired
// @Tags 平台设备-影子
// @Param body body shadowDesiredReq true "sn + desired JSON"
// @Router /api/v1/platform-device/shadow/desired [put]
// @Security Bearer
func (e PlatformDevice) ShadowPutDesired(c *gin.Context) {
	var req shadowDesiredReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Error(400, err, "参数错误")
		return
	}
	if len(req.Desired) == 0 {
		e.Error(400, errors.New("desired empty"), "desired 不能为空")
		return
	}
	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	op := ""
	if uid := user.GetUserId(c); uid > 0 {
		op = c.GetString("user_name")
		if op == "" {
			op = strings.TrimSpace(c.GetString("nickname"))
		}
		if op == "" {
			op = fmt.Sprintf("%d", uid)
		}
	}
	out, err := svc.PutDeviceShadowDesired(&service.PutDeviceShadowDesiredIn{
		Sn:       req.Sn,
		Desired:  req.Desired,
		Merge:    req.Merge,
		Operator: op,
	})
	if err != nil {
		if errors.Is(err, service.ErrPlatformDeviceNotFound) {
			e.Error(404, err, "设备不存在")
			return
		}
		if errors.Is(err, service.ErrPlatformDeviceInvalid) {
			e.Error(400, err, "参数无效")
			return
		}
		e.Logger.Error(err)
		e.Error(500, err, err.Error())
		return
	}
	e.OK(out, "ok")
}

// NormalizedShadowGet 查询规范化设备影子（四表 + 合并现有 JSON/Redis 影子热字段）
// @Summary 规范化设备影子查询
// @Tags 平台设备-影子
// @Param sn path string true "设备 SN"
// @Router /api/v1/platform-device/devices/{sn}/shadow [get]
// @Security Bearer
func (e PlatformDevice) NormalizedShadowGet(c *gin.Context) {
	sn := strings.TrimSpace(c.Param("sn"))
	if sn == "" {
		e.Error(400, errors.New("sn required"), "请传设备 SN")
		return
	}
	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	out, err := svc.GetNormalizedShadow(sn)
	if err != nil {
		if errors.Is(err, service.ErrPlatformDeviceNotFound) {
			e.Error(404, err, "设备不存在")
			return
		}
		if errors.Is(err, service.ErrPlatformDeviceInvalid) {
			e.Error(400, err, "参数无效")
			return
		}
		e.Logger.Error(err)
		e.Error(500, err, "查询失败")
		return
	}
	e.OK(out, "ok")
}

// NormalizedShadowPut 更新规范化设备影子（事务 upsert；不写 device_shadow JSONB）
// @Summary 规范化设备影子更新
// @Tags 平台设备-影子
// @Param sn path string true "设备 SN"
// @Param body body service.NormalizedShadowPutIn true "profile/battery/location/configs 按需提交"
// @Router /api/v1/platform-device/devices/{sn}/shadow [put]
// @Security Bearer
func (e PlatformDevice) NormalizedShadowPut(c *gin.Context) {
	sn := strings.TrimSpace(c.Param("sn"))
	if sn == "" {
		e.Error(400, errors.New("sn required"), "请传设备 SN")
		return
	}
	var req service.NormalizedShadowPutIn
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Error(400, err, "参数错误")
		return
	}
	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	op := ""
	if uid := user.GetUserId(c); uid > 0 {
		op = c.GetString("user_name")
		if op == "" {
			op = strings.TrimSpace(c.GetString("nickname"))
		}
		if op == "" {
			op = fmt.Sprintf("%d", uid)
		}
	}
	req.Operator = op
	if err := svc.PutNormalizedShadow(sn, &req); err != nil {
		if errors.Is(err, service.ErrPlatformDeviceNotFound) {
			e.Error(404, err, "设备不存在")
			return
		}
		if errors.Is(err, service.ErrPlatformDeviceInvalid) {
			e.Error(400, err, "参数无效")
			return
		}
		e.Logger.Error(err)
		e.Error(500, err, err.Error())
		return
	}
	e.OK(map[string]string{"sn": sn}, "ok")
}
