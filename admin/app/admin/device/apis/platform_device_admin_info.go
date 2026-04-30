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

type singleDeviceInfoUpdateReq struct {
	DeviceID json.RawMessage                    `json:"device_id"`
	Sn       string                             `json:"sn"`
	Updates  service.DeviceInfoUpdatesPayload `json:"updates"`
}

type batchDeviceInfoUpdateReq struct {
	Items []service.AdminUpdateDeviceInfoBatchItem `json:"items"`
}

func parseDeviceIDFlexible(raw json.RawMessage) (int64, string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, "", nil
	}
	var n int64
	if err := json.Unmarshal(raw, &n); err == nil && n > 0 {
		return n, "", nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return 0, strings.TrimSpace(s), nil
	}
	return 0, "", fmt.Errorf("device_id 须为正整数或设备 SN 字符串")
}

// AdminInfoUpdate POST /api/v1/platform-device/info/update 单台设备扩展信息更新
// @Summary 管理员更新设备扩展信息
// @Tags 平台设备
// @Router /api/v1/platform-device/info/update [post]
// @Security Bearer
func (e PlatformDevice) AdminInfoUpdate(c *gin.Context) {
	var req singleDeviceInfoUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Error(400, err, "参数错误")
		return
	}
	did, sn, err := parseDeviceIDFlexible(req.DeviceID)
	if err != nil {
		e.Error(400, err, err.Error())
		return
	}
	if did <= 0 && strings.TrimSpace(sn) == "" && strings.TrimSpace(req.Sn) == "" {
		e.Error(400, errors.New("device_id or sn required"), "请提供 device_id 或 sn")
		return
	}
	if strings.TrimSpace(sn) == "" {
		sn = strings.TrimSpace(req.Sn)
	}

	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	uid := int64(user.GetUserId(c))
	op := strings.TrimSpace(c.GetString("user_name"))
	if op == "" {
		op = "admin"
	}
	out, err := svc.AdminUpdateDeviceInfo(&service.AdminUpdateDeviceInfoIn{
		DeviceID: did,
		Sn:       sn,
		Updates:  req.Updates,
		Operator: op,
		UserID:   uid,
		ClientIP: c.ClientIP(),
	})
	if err != nil {
		e.handleAdminInfoUpdateErr(err)
		return
	}
	e.OK(out, "设备信息更新成功")
}

// AdminInfoUpdateBatch POST /api/v1/platform-device/info/update-batch
// @Summary 批量更新设备扩展信息
// @Tags 平台设备
// @Router /api/v1/platform-device/info/update-batch [post]
// @Security Bearer
func (e PlatformDevice) AdminInfoUpdateBatch(c *gin.Context) {
	var req batchDeviceInfoUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Error(400, err, "参数错误")
		return
	}
	if len(req.Items) == 0 {
		e.Error(400, errors.New("empty"), "items 不能为空")
		return
	}
	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	uid := int64(user.GetUserId(c))
	op := strings.TrimSpace(c.GetString("user_name"))
	if op == "" {
		op = "admin"
	}
	out := svc.AdminUpdateDeviceInfoBatch(req.Items, op, uid, c.ClientIP())
	e.OK(out, "批量处理完成")
}

func (e PlatformDevice) handleAdminInfoUpdateErr(err error) {
	if errors.Is(err, service.ErrPlatformDeviceNotFound) {
		e.Error(400, err, "设备不存在")
		return
	}
	if errors.Is(err, service.ErrPlatformDeviceInvalid) {
		e.Error(400, err, "参数无效")
		return
	}
	if errors.Is(err, service.ErrDeviceAdminInfoInvalid) {
		e.Error(400, err, "参数校验失败")
		return
	}
	e.Logger.Error(err)
	e.Error(500, err, "更新失败")
}
