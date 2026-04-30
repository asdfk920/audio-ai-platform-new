package logic

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/services/device/internal/device/shadow"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// DeviceStatusUpdateLogic 设备状态更新逻辑
// 处理设备通过 HTTP 接口主动上报在线状态的业务逻辑
type DeviceStatusUpdateLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDeviceStatusUpdateLogic 创建设备状态更新逻辑实例
// 参数 ctx context.Context: 请求上下文
// 参数 svcCtx *svc.ServiceContext: 服务上下文
// 返回 *DeviceStatusUpdateLogic: 设备状态更新逻辑实例
func NewDeviceStatusUpdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceStatusUpdateLogic {
	return &DeviceStatusUpdateLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeviceStatusUpdate 更新设备在线状态
// 流程：
//  1. 校验请求数据格式（SN、OnlineStatus）
//  2. 查询设备是否存在且有效
//  3. 更新 Redis 缓存中的设备在线状态
//  4. 更新设备影子 Hash 中的在线状态字段
//  5. 记录状态变更日志
//  6. 返回更新结果
//
// 参数 req *types.DeviceStatusUpdateReq: 设备状态更新请求
// 返回 *types.DeviceStatusUpdateResp: 设备状态更新响应
// 返回 error: 更新失败时的错误信息
func (l *DeviceStatusUpdateLogic) DeviceStatusUpdate(req *types.DeviceStatusUpdateReq) (*types.DeviceStatusUpdateResp, error) {
	// 1. 校验请求数据格式
	if err := validateDeviceStatusUpdateReq(req); err != nil {
		return nil, fmt.Errorf("数据格式校验失败: %v", err)
	}

	sn := strings.ToUpper(strings.TrimSpace(req.Sn))
	onlineStatus := req.OnlineStatus

	// 2. 查询设备是否存在且有效
	deviceInfo, err := l.svcCtx.DeviceRegister.FindBySn(l.ctx, sn)
	if err != nil {
		return nil, fmt.Errorf("查询设备失败: %v", err)
	}
	if deviceInfo == nil {
		return nil, fmt.Errorf("设备未注册")
	}

	// 3. 更新 Redis 缓存中的设备在线状态
	rdb := l.svcCtx.Redis
	if rdb != nil {
		ttl := time.Duration(l.svcCtx.Config.DeviceShadow.HeartbeatTTLSeconds) * time.Second
		if ttl <= 0 {
			ttl = 300 * time.Second
		}

		okKey := shadow.OnlineKey(sn)
		sk := shadow.ShadowKey(sn)

		pipe := rdb.Pipeline()
		if onlineStatus == 1 {
			pipe.Set(l.ctx, okKey, "1", ttl)
			pipe.HSet(l.ctx, sk, map[string]interface{}{
				shadow.FOnline:       "1",
				shadow.FLastActiveMs: time.Now().UnixMilli(),
			})
			pipe.Expire(l.ctx, sk, ttl)
			if l.svcCtx.Config.DeviceShadow.EnableOnlineSet {
				pipe.SAdd(l.ctx, shadow.KeyOnlineAll, sn)
			}
		} else {
			pipe.Set(l.ctx, okKey, "0", ttl)
			pipe.HSet(l.ctx, sk, map[string]interface{}{
				shadow.FOnline:       "0",
				shadow.FLastActiveMs: time.Now().UnixMilli(),
			})
			pipe.Expire(l.ctx, sk, ttl)
			if l.svcCtx.Config.DeviceShadow.EnableOnlineSet {
				pipe.SRem(l.ctx, shadow.KeyOnlineAll, sn)
			}
		}

		if _, err := pipe.Exec(l.ctx); err != nil {
			logx.Errorf("更新设备在线状态 Redis 失败: sn=%s, err=%v", sn, err)
		}
	}

	now := time.Now()
	logx.Infof("设备状态更新成功: sn=%s, online_status=%d, updated_at=%s", sn, onlineStatus, now.Format("2006-01-02T15:04:05Z"))

	// 4. 返回更新结果
	return &types.DeviceStatusUpdateResp{
		Sn:           sn,
		OnlineStatus: onlineStatus,
		UpdatedAt:    now.Format("2006-01-02T15:04:05Z"),
	}, nil
}

// validateDeviceStatusUpdateReq 校验设备状态更新请求数据格式
// 校验规则：
//   - SN: 16 位字母数字，正则 ^[A-Z0-9]{16}$，不区分大小写
//   - OnlineStatus: 必须为 0 或 1
//
// 参数 req *types.DeviceStatusUpdateReq: 设备状态更新请求
// 返回 error: 校验失败时的错误信息
func validateDeviceStatusUpdateReq(req *types.DeviceStatusUpdateReq) error {
	snRegex := regexp.MustCompile(`(?i)^[A-Z0-9]{16}$`)
	if !snRegex.MatchString(req.Sn) {
		return fmt.Errorf("SN 格式错误，必须为 16 位字母数字组合")
	}

	if req.OnlineStatus != 0 && req.OnlineStatus != 1 {
		return fmt.Errorf("在线状态值错误，必须为 0（离线）或 1（在线）")
	}

	return nil
}
