package logic

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/jacklau/audio-ai-platform/services/device/internal/device/shadow"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// DeviceDetailLogic 设备详情查询逻辑
// 处理用户查询指定设备详细信息的业务逻辑
type DeviceDetailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDeviceDetailLogic 创建设备详情查询逻辑实例
// 参数 ctx context.Context: 请求上下文
// 参数 svcCtx *svc.ServiceContext: 服务上下文
// 返回 *DeviceDetailLogic: 设备详情查询逻辑实例
func NewDeviceDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceDetailLogic {
	return &DeviceDetailLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeviceDetail 查询设备详细信息
// 流程：
//   1. 从 JWT token 中获取用户 ID
//   2. 校验请求数据格式（SN）
//   3. 查询设备是否存在
//   4. 检查设备是否绑定到当前用户
//   5. 查询设备在线状态
//   6. 查询设备影子信息（电量、存储、运行状态等）
//   7. 组装数据并返回
//
// 参数 req *types.DeviceDetailReq: 设备详情请求
// 返回 *types.DeviceDetailResp: 设备详情响应
// 返回 error: 查询失败时的错误信息
func (l *DeviceDetailLogic) DeviceDetail(req *types.DeviceDetailReq) (*types.DeviceDetailResp, error) {
	// 1. 从 JWT token 中获取用户 ID
	userID, ok := jwt.GetUserIdFromContext(l.ctx)
	if !ok || userID <= 0 {
		return nil, fmt.Errorf("请先登录")
	}

	// 2. 校验请求数据格式
	if err := validateDeviceDetailReq(req); err != nil {
		return nil, fmt.Errorf("数据格式校验失败: %v", err)
	}

	sn := strings.ToUpper(strings.TrimSpace(req.Sn))

	// 3. 查询设备是否存在
	deviceInfo, err := l.svcCtx.DeviceRegister.FindBySn(l.ctx, sn)
	if err != nil {
		return nil, fmt.Errorf("查询设备失败: %v", err)
	}
	if deviceInfo == nil {
		return nil, fmt.Errorf("设备未注册")
	}

	// 4. 检查设备是否绑定到当前用户
	bindInfo, err := l.svcCtx.UserDeviceBindRepo.FindByUserIdAndDeviceId(l.ctx, userID, deviceInfo.ID)
	isBound := bindInfo != nil && bindInfo.UserID == userID
	var boundAt string
	if isBound {
		boundAt = bindInfo.CreatedAt.Format("2006-01-02T15:04:05Z")
	}

	// 5. 查询设备在线状态
	onlineStatus := l.getDeviceOnlineStatus(sn)

	// 6. 查询设备影子信息
	shadowInfo := l.getDeviceShadowInfo(sn)

	logx.Infof("查询设备详情成功: user_id=%d, sn=%s, is_bound=%v", userID, sn, isBound)

	// 7. 组装数据并返回
	return &types.DeviceDetailResp{
		ID:              deviceInfo.ID,
		Sn:              sn,
		Model:           deviceInfo.Model,
		FirmwareVersion: deviceInfo.FirmwareVersion,
		OnlineStatus:    onlineStatus,
		BoundAt:         boundAt,
		IsBound:         isBound,
		Shadow:          shadowInfo,
	}, nil
}

// getDeviceOnlineStatus 获取设备在线状态
// 优先查询 Redis，降级查询数据库
func (l *DeviceDetailLogic) getDeviceOnlineStatus(sn string) int {
	rdb := l.svcCtx.Redis
	if rdb != nil {
		okKey := shadow.OnlineKey(sn)
		val, err := rdb.Get(l.ctx, okKey).Result()
		if err == nil && val == "1" {
			return 1
		}
	}

	isOnline, err := l.svcCtx.DeviceRegister.IsOnline(l.ctx, sn)
	if err != nil {
		return 0
	}

	if isOnline {
		return 1
	}

	return 0
}

// getDeviceShadowInfo 获取设备影子信息
// 从 Redis 设备影子 Hash 中读取实时状态数据
func (l *DeviceDetailLogic) getDeviceShadowInfo(sn string) types.DeviceShadowInfo {
	rdb := l.svcCtx.Redis
	if rdb == nil {
		return types.DeviceShadowInfo{}
	}

	sk := shadow.ShadowKey(sn)
	result, err := rdb.HMGet(l.ctx, sk,
		shadow.FBattery,
		shadow.FRunState,
		shadow.FLastActiveMs,
		shadow.FFirmwareVersion,
		shadow.FIP,
	).Result()
	if err != nil {
		return types.DeviceShadowInfo{}
	}

	var info types.DeviceShadowInfo

	if len(result) >= 1 && result[0] != nil {
		if b, ok := result[0].(string); ok && b != "" {
			info.Battery, _ = strconv.Atoi(b)
		}
	}

	if len(result) >= 2 && result[1] != nil {
		if rs, ok := result[1].(string); ok {
			info.RunState = rs
		}
	}

	if len(result) >= 3 && result[2] != nil {
		if ms, ok := result[2].(string); ok && ms != "" {
			info.LastActiveMs, _ = strconv.ParseInt(ms, 10, 64)
		}
	}

	if len(result) >= 4 && result[3] != nil {
		if fv, ok := result[3].(string); ok {
			info.FirmwareVersion = fv
		}
	}

	if len(result) >= 5 && result[4] != nil {
		if ip, ok := result[4].(string); ok {
			info.IP = ip
		}
	}

	return info
}

// validateDeviceDetailReq 校验设备详情请求数据格式
// 校验规则：
//   - SN: 16 位字母数字，正则 ^[A-Z0-9]{16}$，不区分大小写
//
// 参数 req *types.DeviceDetailReq: 设备详情请求
// 返回 error: 校验失败时的错误信息
func validateDeviceDetailReq(req *types.DeviceDetailReq) error {
	snRegex := regexp.MustCompile(`(?i)^[A-Z0-9]{16}$`)
	if !snRegex.MatchString(req.Sn) {
		return fmt.Errorf("SN 格式错误，必须为 16 位字母数字组合")
	}

	return nil
}
