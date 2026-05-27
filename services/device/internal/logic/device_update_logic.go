package logic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

// DeviceUpdateLogic 设备信息更新逻辑
// 处理用户通过 App 更新已绑定设备的基本信息和配置
type DeviceUpdateLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDeviceUpdateLogic 创建设备信息更新逻辑实例
func NewDeviceUpdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceUpdateLogic {
	return &DeviceUpdateLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeviceUpdate 更新设备信息
//
// 完整流程：
//  1. 从 JWT token 中获取用户 ID
//  2. 校验请求数据格式（SN）
//  3. 查询设备是否存在且有效
//  4. 检查设备是否已绑定当前用户
//  5. 更新设备信息（device_name、location、group_name、scene）
//  6. 同时更新 device 表和 user_device_bind 表的 device_name 字段
//  7. 返回更新后的设备信息
//
// 参数 req *types.DeviceUpdateReq: 设备信息更新请求
// 返回 *types.DeviceUpdateResp: 设备信息更新响应
// 返回 error: 更新失败时的错误信息
func (l *DeviceUpdateLogic) DeviceUpdate(req *types.DeviceUpdateReq) (*types.DeviceUpdateResp, error) {
	logx.Infof("====================================")
	logx.Infof("[Device Update] 📝 开始更新设备信息...")

	// 1. 从 JWT token 中获取用户 ID
	userID, ok := jwt.GetUserIdFromContext(l.ctx)
	if !ok || userID <= 0 {
		return nil, fmt.Errorf("请先登录")
	}

	// 2. 校验请求数据格式
	sn := strings.TrimSpace(req.Sn)
	if sn == "" {
		return nil, fmt.Errorf("设备序列号不能为空")
	}
	if err := validateReqSN(sn); err != nil {
		return nil, err
	}

	logx.Infof("[Device Update] 📋 用户ID: %d, 设备SN: %s", userID, sn)

	// 3. 查询设备是否存在且有效
	deviceInfo, err := l.svcCtx.DeviceRegister.FindBySn(l.ctx, sn)
	if err != nil {
		return nil, fmt.Errorf("查询设备失败: %v", err)
	}
	if deviceInfo == nil {
		return nil, fmt.Errorf("设备未注册")
	}

	logx.Infof("[Device Update] ✅ 找到设备: id=%d", deviceInfo.ID)

	// 4. 检查设备是否已绑定当前用户
	bindInfo, err := l.svcCtx.UserDeviceBindRepo.FindByUserIdAndDeviceId(l.ctx, userID, deviceInfo.ID)
	if err != nil {
		return nil, fmt.Errorf("查询绑定关系失败: %v", err)
	}
	if bindInfo == nil {
		return nil, fmt.Errorf("设备未绑定到当前账户，请先绑定设备")
	}

	logx.Infof("[Device Update] ✅ 确认绑定关系: bind_id=%d", bindInfo.ID)

	// 5. 准备更新数据
	now := time.Now()
	deviceName := strings.TrimSpace(req.DeviceName)
	location := strings.TrimSpace(req.Location)
	groupName := strings.TrimSpace(req.GroupName)
	scene := strings.TrimSpace(req.Scene)

	// 如果 device_name 为空，保持原值不更新
	if deviceName == "" {
		deviceName = bindInfo.DeviceName
	}
	// 如果仍为空，使用默认值
	if deviceName == "" {
		deviceName = "未命名设备"
	}

	// 6. 更新 user_device_bind 表
	err = l.svcCtx.UserDeviceBindRepo.UpdateDeviceInfo(
		l.ctx,
		bindInfo.ID,
		deviceName,
		location,
		groupName,
		scene,
	)
	if err != nil {
		logx.Errorf("[Device Update] ❌ 更新用户设备绑定信息失败: %v", err)
		return nil, fmt.Errorf("更新设备信息失败: %w", err)
	}

	// 7. 同步更新 device 表的 device_name 字段（如果提供了新的名称）
	if req.DeviceName != "" {
		err = l.svcCtx.DeviceRepo.UpdateDeviceName(l.ctx, deviceInfo.ID, deviceName)
		if err != nil {
			// 不影响主流程，仅记录日志
			logx.Slowf("[Device Update] sync update device table device_name failed (no effect): %v", err)
		}
	}

	logx.Infof("\n====================================")
	logx.Infof("🎉 [Device Update] ✅✅✅ 设备信息更新成功! ✅✅✅")
	logx.Infof("====================================")

	// 8. 返回更新后的设备信息
	return &types.DeviceUpdateResp{
		Sn:         sn,
		DeviceName: deviceName,
		Location:   location,
		GroupName:  groupName,
		Scene:      scene,
		UpdatedAt:  now.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}
