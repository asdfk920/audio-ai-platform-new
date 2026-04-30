package logic

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// DevicePlaybackStatusLogic 设备播放状态查询逻辑
// 处理用户通过 App 查询设备当前播放状态的业务逻辑
// 支持从 Redis 缓存或 MySQL 数据库查询设备播放状态
// 包含用户身份验证、设备权限校验、播放状态查询等

type DevicePlaybackStatusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDevicePlaybackStatusLogic 创建设备播放状态查询逻辑实例
// 参数 ctx context.Context: 请求上下文
// 参数 svcCtx *svc.ServiceContext: 服务上下文
// 返回 *DevicePlaybackStatusLogic: 设备播放状态查询逻辑实例
func NewDevicePlaybackStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DevicePlaybackStatusLogic {
	return &DevicePlaybackStatusLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DevicePlaybackStatus 查询设备播放状态
// 接收播放状态查询请求，验证用户权限，查询设备播放状态信息
// 参数 req *types.DevicePlaybackStatusReq: 播放状态查询请求
// 返回 *types.DevicePlaybackStatusResp: 播放状态查询响应
// 返回 error: 错误信息
func (l *DevicePlaybackStatusLogic) DevicePlaybackStatus(req *types.DevicePlaybackStatusReq) (*types.DevicePlaybackStatusResp, error) {
	// 1. 校验 Token 是否存在
	userID, ok := jwt.GetUserIdFromContext(l.ctx)
	if !ok || userID <= 0 {
		return nil, fmt.Errorf("请先登录")
	}

	// 2. 校验请求参数
	if err := validateDevicePlaybackStatusReq(req); err != nil {
		return nil, fmt.Errorf("参数校验失败: %v", err)
	}

	sn := strings.ToUpper(strings.TrimSpace(req.Sn))

	// 3. 查询设备是否存在
	deviceInfo, err := l.svcCtx.DeviceRegister.FindBySn(l.ctx, sn)
	if err != nil {
		return nil, fmt.Errorf("查询设备失败: %v", err)
	}
	if deviceInfo == nil {
		return nil, fmt.Errorf("设备不存在: %s", sn)
	}

	// 4. 验证用户权限：查询 user_device_bind 绑定表
	bindInfo, err := l.svcCtx.UserDeviceBindRepo.FindByUserIdAndDeviceId(l.ctx, userID, deviceInfo.ID)
	if err != nil {
		return nil, fmt.Errorf("查询绑定关系失败: %v", err)
	}
	if bindInfo == nil {
		return nil, fmt.Errorf("无权限查询该设备")
	}

	// 5. 查询设备在线状态
	online := l.isDeviceOnline(sn)

	// 6. 查询设备播放状态
	playbackStatus, err := l.queryDevicePlaybackStatus(sn, online)
	if err != nil {
		return nil, fmt.Errorf("查询播放状态失败: %v", err)
	}

	// 7. 组装响应
	resp := &types.DevicePlaybackStatusResp{
		Sn:         sn,
		Online:     online,
		Playback:   playbackStatus,
		LastUpdate: time.Now().Format(time.RFC3339),
	}

	logx.Infof("设备播放状态查询成功: user_id=%d, sn=%s, online=%t, state=%s",
		userID, sn, online, playbackStatus.State)

	return resp, nil
}

// isDeviceOnline 检查设备在线状态
// 参数 sn string: 设备序列号
// 返回 bool: 设备在线状态
func (l *DevicePlaybackStatusLogic) isDeviceOnline(sn string) bool {
	// 1. 首先尝试从 Redis 缓存查询设备在线状态
	online, err := l.queryDeviceOnlineFromRedis(sn)
	if err == nil {
		return online
	}

	// 2. 如果 Redis 未命中，查询 MySQL 数据库
	online, err = l.queryDeviceOnlineFromMySQL(sn)
	if err == nil {
		return online
	}

	// 3. 如果查询失败，默认返回离线状态
	return false
}

// queryDeviceOnlineFromRedis 从 Redis 缓存查询设备在线状态
// 参数 sn string: 设备序列号
// 返回 bool: 设备在线状态
// 返回 error: 查询错误
func (l *DevicePlaybackStatusLogic) queryDeviceOnlineFromRedis(sn string) (bool, error) {
	// Redis 键名格式：device:online:{sn}
	key := fmt.Sprintf("device:online:%s", sn)

	val, err := l.svcCtx.Redis.Get(l.ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("Redis查询失败: %v", err)
	}

	// 解析在线状态
	online := val == "online"
	logx.Infof("queryDeviceOnlineFromRedis: Redis hit, sn=%s, online=%t", sn, online)

	return online, nil
}

// queryDeviceOnlineFromMySQL 从 MySQL 数据库查询设备在线状态
// 参数 sn string: 设备序列号
// 返回 bool: 设备在线状态
// 返回 error: 查询错误
func (l *DevicePlaybackStatusLogic) queryDeviceOnlineFromMySQL(sn string) (bool, error) {
	// 查询 device_status 表获取最新状态
	query := `
		SELECT online_status 
		FROM device_status 
		WHERE sn = $1 
		ORDER BY last_active_at DESC 
		LIMIT 1
	`

	var onlineStatus int16
	err := l.svcCtx.DB.QueryRowContext(l.ctx, query, sn).Scan(&onlineStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			logx.Infof("queryDeviceOnlineFromMySQL: No rows found, sn=%s", sn)
			return false, nil
		}
		return false, fmt.Errorf("MySQL查询失败: %v", err)
	}

	online := onlineStatus == 1
	logx.Infof("queryDeviceOnlineFromMySQL: MySQL result, sn=%s, online_status=%d, online=%t", sn, onlineStatus, online)
	return online, nil
}

// queryDevicePlaybackStatus 查询设备播放状态
// 参数 sn string: 设备序列号
// 参数 online bool: 设备在线状态
// 返回 *types.DevicePlaybackStatus: 播放状态信息
// 返回 error: 查询错误
func (l *DevicePlaybackStatusLogic) queryDevicePlaybackStatus(sn string, online bool) (*types.DevicePlaybackStatus, error) {
	// 1. 首先尝试从 Redis 缓存查询播放状态
	playbackStatus, err := l.queryPlaybackStatusFromRedis(sn)
	if err == nil && playbackStatus != nil {
		return playbackStatus, nil
	}

	// 2. 如果 Redis 未命中，查询 MySQL 数据库
	playbackStatus, err = l.queryPlaybackStatusFromMySQL(sn)
	if err == nil && playbackStatus != nil {
		// 将查询结果缓存到 Redis
		l.cachePlaybackStatusToRedis(sn, playbackStatus)
		return playbackStatus, nil
	}

	// 3. 如果数据库也无记录，返回默认播放状态
	return l.getDefaultPlaybackStatus(online), nil
}

// queryPlaybackStatusFromRedis 从 Redis 缓存查询播放状态
// 参数 sn string: 设备序列号
// 返回 *types.DevicePlaybackStatus: 播放状态信息
// 返回 error: 查询错误
func (l *DevicePlaybackStatusLogic) queryPlaybackStatusFromRedis(sn string) (*types.DevicePlaybackStatus, error) {
	// Redis 键名格式：device:playback:{sn}
	key := fmt.Sprintf("device:playback:%s", sn)

	result, err := l.svcCtx.Redis.Get(l.ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("Redis查询失败: %v", err)
	}

	var playbackStatus types.DevicePlaybackStatus
	err = json.Unmarshal([]byte(result), &playbackStatus)
	if err != nil {
		return nil, fmt.Errorf("JSON解析失败: %v", err)
	}

	return &playbackStatus, nil
}

// queryPlaybackStatusFromMySQL 从 MySQL 数据库查询播放状态
// 参数 sn string: 设备序列号
// 返回 *types.DevicePlaybackStatus: 播放状态信息
// 返回 error: 查询错误
func (l *DevicePlaybackStatusLogic) queryPlaybackStatusFromMySQL(sn string) (*types.DevicePlaybackStatus, error) {
	// 由于当前项目中没有专门的播放状态表，这里模拟查询
	// 实际项目中应该查询 device_playback_status 表

	// 模拟查询：这里返回空结果，让逻辑返回默认状态
	return nil, fmt.Errorf("播放状态记录不存在")
}

// cachePlaybackStatusToRedis 将播放状态缓存到 Redis
// 参数 sn string: 设备序列号
// 参数 playbackStatus *types.DevicePlaybackStatus: 播放状态信息
func (l *DevicePlaybackStatusLogic) cachePlaybackStatusToRedis(sn string, playbackStatus *types.DevicePlaybackStatus) {
	// Redis 键名格式：device:playback:{sn}
	key := fmt.Sprintf("device:playback:%s", sn)

	data, err := json.Marshal(playbackStatus)
	if err != nil {
		logx.Errorf("JSON序列化失败: %v", err)
		return
	}

	// 缓存过期时间：5分钟
	err = l.svcCtx.Redis.Set(l.ctx, key, data, 5*time.Minute).Err()
	if err != nil {
		logx.Errorf("Redis缓存失败: %v", err)
	}
}

// getDefaultPlaybackStatus 获取默认播放状态
// 参数 online bool: 设备在线状态
// 返回 *types.DevicePlaybackStatus: 默认播放状态信息
func (l *DevicePlaybackStatusLogic) getDefaultPlaybackStatus(online bool) *types.DevicePlaybackStatus {
	// 根据设备在线状态返回不同的默认状态
	if online {
		// 设备在线时返回播放中的默认状态
		return &types.DevicePlaybackStatus{
			State: "stopped",
			CurrentSong: &types.PlaybackCurrentSong{
				MediaName: "",
				Artist:    "",
				Duration:  0,
			},
			Progress: &types.PlaybackProgress{
				CurrentTime: 0,
				Duration:    0,
				Percentage:  0,
			},
			Volume: 50,
			Mode: &types.PlaybackMode{
				Loop:    "off",
				Shuffle: false,
			},
			QueueLength: 0,
		}
	} else {
		// 设备离线时返回已暂停的默认状态
		return &types.DevicePlaybackStatus{
			State: "paused",
			CurrentSong: &types.PlaybackCurrentSong{
				MediaName: "",
				Artist:    "",
				Duration:  0,
			},
			Progress: &types.PlaybackProgress{
				CurrentTime: 0,
				Duration:    0,
				Percentage:  0,
			},
			Volume: 50,
			Mode: &types.PlaybackMode{
				Loop:    "off",
				Shuffle: false,
			},
			QueueLength: 0,
		}
	}
}

// validateDevicePlaybackStatusReq 校验播放状态查询请求参数
// 参数 req *types.DevicePlaybackStatusReq: 播放状态查询请求
// 返回 error: 校验错误
func validateDevicePlaybackStatusReq(req *types.DevicePlaybackStatusReq) error {
	if req == nil {
		return fmt.Errorf("请求不能为空")
	}

	// 校验 SN 格式：16位字母数字组合
	sn := strings.TrimSpace(req.Sn)
	if sn == "" {
		return fmt.Errorf("设备序列号不能为空")
	}

	matched, _ := regexp.MatchString(`^[A-Za-z0-9]{16}$`, sn)
	if !matched {
		return fmt.Errorf("设备序列号格式错误，必须为16位字母数字组合")
	}

	return nil
}
