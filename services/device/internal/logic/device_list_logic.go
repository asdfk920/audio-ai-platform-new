package logic

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jacklau/audio-ai-platform/services/device/internal/device/shadow"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// DeviceListLogic 设备列表查询逻辑
// 处理用户查询已绑定设备列表的业务逻辑
type DeviceListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDeviceListLogic 创建设备列表查询逻辑实例
// 参数 ctx context.Context: 请求上下文
// 参数 svcCtx *svc.ServiceContext: 服务上下文
// 返回 *DeviceListLogic: 设备列表查询逻辑实例
func NewDeviceListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceListLogic {
	return &DeviceListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeviceList 查询用户已绑定设备列表
// 流程：
//  1. 从 JWT token 中获取用户 ID
//  2. 查询用户绑定的设备列表
//  3. 查询每个设备的详细信息
//  4. 查询每个设备的在线状态
//  5. 查询每个设备的影子信息（电量、运行状态等）
//  6. 组装数据并返回
//
// 返回 *types.DeviceListResp: 设备列表响应
// 返回 error: 查询失败时的错误信息
func (l *DeviceListLogic) DeviceList() (*types.DeviceListResp, error) {
	// 1. 从 JWT token 中获取用户 ID
	userID, ok := jwt.GetUserIdFromContext(l.ctx)
	if !ok || userID <= 0 {
		return nil, fmt.Errorf("请先登录")
	}

	// 2. 查询用户绑定的设备列表
	bindList, err := l.svcCtx.UserDeviceBindRepo.FindListByUserId(l.ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("查询绑定设备列表失败: %v", err)
	}

	if len(bindList) == 0 {
		return &types.DeviceListResp{
			Total: 0,
			List:  []types.DeviceListItem{},
		}, nil
	}

	// 3. 组装设备列表数据
	list := make([]types.DeviceListItem, 0, len(bindList))
	for _, bind := range bindList {
		sn := strings.ToUpper(strings.TrimSpace(bind.SN))
		if sn == "" {
			continue
		}

		// 查询设备详细信息
		deviceInfo, err := l.svcCtx.DeviceRegister.FindBySn(l.ctx, sn)
		if err != nil || deviceInfo == nil {
			logx.Errorf("查询设备详情失败: sn=%s, err=%v", sn, err)
			continue
		}

		// 查询设备在线状态
		onlineStatus := l.getDeviceOnlineStatus(sn)

		// 查询设备影子信息（电量、运行状态）
		battery, runState := l.getDeviceShadowInfo(sn)

		list = append(list, types.DeviceListItem{
			ID:              deviceInfo.ID,
			Sn:              sn,
			Model:           deviceInfo.Model,
			FirmwareVersion: deviceInfo.FirmwareVersion,
			OnlineStatus:    onlineStatus,
			BoundAt:         bind.CreatedAt.Format("2006-01-02T15:04:05Z"),
			Battery:         battery,
			RunState:        runState,
		})
	}

	logx.Infof("查询设备列表成功: user_id=%d, total=%d", userID, len(list))

	return &types.DeviceListResp{
		Total: len(list),
		List:  list,
	}, nil
}

// getDeviceOnlineStatus 获取设备在线状态
// 优先查询 Redis，降级查询数据库
func (l *DeviceListLogic) getDeviceOnlineStatus(sn string) int {
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

// getDeviceShadowInfo 获取设备影子信息（电量、运行状态）
func (l *DeviceListLogic) getDeviceShadowInfo(sn string) (int, string) {
	rdb := l.svcCtx.Redis
	if rdb == nil {
		return 0, ""
	}

	sk := shadow.ShadowKey(sn)
	result, err := rdb.HMGet(l.ctx, sk, shadow.FBattery, shadow.FRunState).Result()
	if err != nil {
		return 0, ""
	}

	var battery int
	var runState string

	if len(result) >= 1 && result[0] != nil {
		if b, ok := result[0].(string); ok && b != "" {
			battery, _ = strconv.Atoi(b)
		}
	}

	if len(result) >= 2 && result[1] != nil {
		if rs, ok := result[1].(string); ok {
			runState = rs
		}
	}

	return battery, runState
}
