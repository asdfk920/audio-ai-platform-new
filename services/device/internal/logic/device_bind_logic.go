package logic

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/model"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// DeviceBindLogic 设备绑定逻辑
// 处理用户通过 App 将设备绑定到当前登录账户的业务逻辑
type DeviceBindLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDeviceBindLogic 创建设备绑定逻辑实例
// 参数 ctx context.Context: 请求上下文
// 参数 svcCtx *svc.ServiceContext: 服务上下文
// 返回 *DeviceBindLogic: 设备绑定逻辑实例
func NewDeviceBindLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceBindLogic {
	return &DeviceBindLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeviceBind 绑定设备到当前登录用户
// 流程：
//  1. 从 JWT token 中获取用户 ID
//  2. 校验请求数据格式（SN）
//  3. 查询设备是否存在且有效
//  4. 检查设备是否在线
//  5. 检查设备是否已被其他用户绑定
//  6. 检查用户是否已达到绑定设备数量上限
//  7. 检查是否已绑定（幂等处理）
//  8. 创建绑定记录
//  9. 返回绑定结果
//
// 参数 req *types.DeviceBindReq: 设备绑定请求
// 返回 *types.DeviceBindResp: 设备绑定响应
// 返回 error: 绑定失败时的错误信息
func (l *DeviceBindLogic) DeviceBind(req *types.DeviceBindReq) (*types.DeviceBindResp, error) {
	// 1. 从 JWT token 中获取用户 ID
	userID, ok := jwt.GetUserIdFromContext(l.ctx)
	if !ok || userID <= 0 {
		return nil, fmt.Errorf("请先登录")
	}

	// 2. 校验请求数据格式
	if err := validateDeviceBindReq(req); err != nil {
		return nil, fmt.Errorf("数据格式校验失败: %v", err)
	}

	sn := strings.ToUpper(strings.TrimSpace(req.Sn))

	// 3. 查询设备是否存在且有效
	deviceInfo, err := l.svcCtx.DeviceRegister.FindBySn(l.ctx, sn)
	if err != nil {
		return nil, fmt.Errorf("查询设备失败: %v", err)
	}
	if deviceInfo == nil {
		return nil, fmt.Errorf("设备未注册")
	}

	// 4. 检查设备是否在线
	if !l.isDeviceOnline(sn) {
		return nil, fmt.Errorf("设备离线，请确保设备已连接网络")
	}

	// 5. 检查设备是否已被其他用户绑定
	existingBind, err := l.svcCtx.UserDeviceBindRepo.FindByDeviceId(l.ctx, deviceInfo.ID)
	if err != nil {
		return nil, fmt.Errorf("查询设备绑定状态失败: %v", err)
	}
	if existingBind != nil && existingBind.UserID != userID {
		return nil, fmt.Errorf("设备已被绑定")
	}

	// 6. 检查用户是否已达到绑定设备数量上限
	maxBinds := l.svcCtx.Config.MaxDeviceBinds
	if maxBinds <= 0 {
		maxBinds = 10
	}
	bindCount, err := l.svcCtx.UserDeviceBindRepo.CountByUserId(l.ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("查询用户绑定设备数量失败: %v", err)
	}
	if bindCount >= int64(maxBinds) {
		return nil, fmt.Errorf("绑定设备数量已达上限")
	}

	// 7. 检查是否已绑定（幂等处理）
	if existingBind != nil && existingBind.UserID == userID {
		logx.Infof("设备已绑定（幂等）: user_id=%d, sn=%s", userID, sn)
		return &types.DeviceBindResp{
			Sn:      sn,
			BoundAt: existingBind.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}, nil
	}

	// 8. 创建绑定记录
	bind := &model.UserDeviceBind{
		UserID:   userID,
		DeviceID: deviceInfo.ID,
		SN:       sn,
		Status:   model.UserDeviceBindStatusNormal,
	}
	if err := l.svcCtx.UserDeviceBindRepo.Create(l.ctx, bind); err != nil {
		return nil, fmt.Errorf("创建设备绑定失败: %v", err)
	}

	logx.Infof("设备绑定成功: user_id=%d, device_id=%d, sn=%s", userID, deviceInfo.ID, sn)

	// 9. 返回绑定结果
	return &types.DeviceBindResp{
		Sn:      sn,
		BoundAt: bind.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

// isDeviceOnline 检查设备是否在线
// 优先查询 Redis 中的设备在线状态键，如果 Redis 中没有则查询数据库
func (l *DeviceBindLogic) isDeviceOnline(sn string) bool {
	rdb := l.svcCtx.Redis
	if rdb != nil {
		okKey := "device:online:" + sn
		val, err := rdb.Get(l.ctx, okKey).Result()
		if err == nil && val == "1" {
			return true
		}
	}

	deviceInfo, err := l.svcCtx.DeviceRegister.FindBySn(l.ctx, sn)
	if err != nil || deviceInfo == nil {
		return false
	}

	onlineRepo := l.svcCtx.DeviceRegister
	isOnline, err := onlineRepo.IsOnline(l.ctx, sn)
	if err != nil {
		return false
	}

	return isOnline
}

// validateDeviceBindReq 校验设备绑定请求数据格式
// 校验规则：
//   - SN: 16 位字母数字，正则 ^[A-Z0-9]{16}$，不区分大小写
//
// 参数 req *types.DeviceBindReq: 设备绑定请求
// 返回 error: 校验失败时的错误信息
func validateDeviceBindReq(req *types.DeviceBindReq) error {
	snRegex := regexp.MustCompile(`(?i)^[A-Z0-9]{16}$`)
	if !snRegex.MatchString(req.Sn) {
		return fmt.Errorf("SN 格式错误，必须为 16 位字母数字组合")
	}

	return nil
}
