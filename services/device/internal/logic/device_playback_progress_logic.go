package logic

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// DevicePlaybackProgressLogic 设备播放进度查询逻辑
// 处理用户通过 App 查询设备当前播放进度的业务逻辑
// 支持从 Redis 缓存或 MySQL 数据库查询设备播放进度
// 返回轻量级进度信息：当前时间、总时长、百分比、剩余时间

type DevicePlaybackProgressLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDevicePlaybackProgressLogic 创建设备播放进度查询逻辑实例
// 参数 ctx context.Context: 请求上下文
// 参数 svcCtx *svc.ServiceContext: 服务上下文
// 返回 *DevicePlaybackProgressLogic: 设备播放进度查询逻辑实例
func NewDevicePlaybackProgressLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DevicePlaybackProgressLogic {
	return &DevicePlaybackProgressLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DevicePlaybackProgress 查询设备播放进度
// 接收播放进度查询请求，验证用户权限，查询设备播放进度信息
// 参数 req *types.DevicePlaybackProgressReq: 播放进度查询请求
// 返回 *types.DevicePlaybackProgressResp: 播放进度查询响应
// 返回 error: 错误信息
func (l *DevicePlaybackProgressLogic) DevicePlaybackProgress(req *types.DevicePlaybackProgressReq) (*types.DevicePlaybackProgressResp, error) {
	// 1. 校验 Token 是否存在，提取用户身份
	userID, ok := jwt.GetUserIdFromContext(l.ctx)
	if !ok || userID <= 0 {
		return nil, fmt.Errorf("请先登录")
	}

	// 2. 校验请求参数
	if err := validateDevicePlaybackProgressReq(req); err != nil {
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

	// 6. 查询设备播放进度数据（Redis优先，MySQL降级）
	currentTime, duration, timestamp, err := l.queryDeviceProgress(sn)
	if err != nil {
		logx.Errorf("查询播放进度失败: sn=%s, err=%v", sn, err)
	}

	// 7. 计算进度百分比和剩余时间
	percentage := calculatePercentage(currentTime, duration)
	remainingTime := calculateRemainingTime(currentTime, duration)

	// 8. 组装响应数据
	resp := &types.DevicePlaybackProgressResp{
		Sn:            sn,
		Online:        online,
		CurrentTime:   currentTime,
		Duration:      duration,
		Percentage:    percentage,
		RemainingTime: remainingTime,
		Timestamp:     timestamp,
	}

	// 9. 设备离线时添加备注说明
	if !online {
		resp.Note = "设备离线，数据可能不是最新"
	}

	logx.Infof("设备播放进度查询成功: user_id=%d, sn=%s, online=%t, progress=%ds/%ds (%.1f%%)",
		userID, sn, online, currentTime, duration, percentage)

	return resp, nil
}

// isDeviceOnline 检查设备在线状态
// 优先查询 Redis 缓存，未命中则降级查询 MySQL 数据库
// 参数 sn string: 设备序列号
// 返回 bool: 设备在线状态
func (l *DevicePlaybackProgressLogic) isDeviceOnline(sn string) bool {
	online, err := l.queryDeviceOnlineFromRedis(sn)
	if err == nil {
		logx.Infof("isDeviceOnline: Redis查询成功, sn=%s, online=%t", sn, online)
		return online
	}

	logx.Infof("isDeviceOnline: Redis查询失败, sn=%s, error=%v", sn, err)

	online, err = l.queryDeviceOnlineFromMySQL(sn)
	if err == nil {
		logx.Infof("isDeviceOnline: MySQL查询成功, sn=%s, online=%t", sn, online)
		return online
	}

	logx.Infof("isDeviceOnline: MySQL查询失败, sn=%s, error=%v", sn, err)
	return false
}

// queryDeviceOnlineFromRedis 从 Redis 缓存查询设备在线状态
// 键名格式为 device:online:{sn}
// 参数 sn string: 设备序列号
// 返回 bool: 设备在线状态
// 返回 error: 查询错误
func (l *DevicePlaybackProgressLogic) queryDeviceOnlineFromRedis(sn string) (bool, error) {
	key := fmt.Sprintf("device:online:%s", sn)
	logx.Infof("queryDeviceOnlineFromRedis: 查询Redis, key=%s", key)

	val, err := l.svcCtx.Redis.Get(l.ctx, key).Result()
	if err != nil {
		logx.Infof("queryDeviceOnlineFromRedis: Redis Get失败, key=%s, error=%v", key, err)
		return false, fmt.Errorf("Redis查询失败: %v", err)
	}

	logx.Infof("queryDeviceOnlineFromRedis: Redis Get成功, key=%s, val=%s", key, val)
	return val == "online" || val == "1", nil
}

// queryDeviceOnlineFromMySQL 从 MySQL 数据库查询设备在线状态
// 从 device 表查询在线状态
// 参数 sn string: 设备序列号
// 返回 bool: 设备在线状态
// 返回 error: 查询错误
func (l *DevicePlaybackProgressLogic) queryDeviceOnlineFromMySQL(sn string) (bool, error) {
	query := `
		SELECT online_status 
		FROM device 
		WHERE sn = $1 
		LIMIT 1
	`

	logx.Infof("queryDeviceOnlineFromMySQL: 查询MySQL, sn=%s", sn)

	var onlineStatus int16
	err := l.svcCtx.DB.QueryRowContext(l.ctx, query, sn).Scan(&onlineStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			logx.Infof("queryDeviceOnlineFromMySQL: MySQL无记录, sn=%s", sn)
			return false, nil
		}
		logx.Infof("queryDeviceOnlineFromMySQL: MySQL查询失败, sn=%s, error=%v", sn, err)
		return false, fmt.Errorf("MySQL查询失败: %v", err)
	}

	online := onlineStatus == 1
	logx.Infof("queryDeviceOnlineFromMySQL: MySQL查询成功, sn=%s, online_status=%d, online=%t", sn, onlineStatus, online)
	return online, nil
}

// queryDeviceProgress 查询设备播放进度数据
// 优先从 Redis 缓存读取，键名格式为 device:progress:{sn}
// Redis 未命中时查询 MySQL 数据库 device_playback_status 表
// 按 last_update 倒序获取最新的播放进度记录
// 参数 sn string: 设备序列号
// 返回 int: 当前播放位置秒数
// 返回 int: 总时长秒数
// 返回 string: 进度记录时间戳
// 返回 error: 查询错误
func (l *DevicePlaybackProgressLogic) queryDeviceProgress(sn string) (int, int, string, error) {
	// 默认值
	defaultTime := time.Now().Format(time.RFC3339)

	// 1. 尝试从 Redis 缓存获取播放进度
	currentTime, duration, timestamp, err := l.queryProgressFromRedis(sn)
	if err == nil && currentTime > 0 || duration > 0 {
		return currentTime, duration, timestamp, nil
	}

	// 2. Redis 未命中，查询 MySQL 数据库
	currentTime, duration, timestamp, err = l.queryProgressFromMySQL(sn)
	if err == nil && currentTime > 0 || duration > 0 {
		l.cacheProgressToRedis(sn, currentTime, duration)
		return currentTime, duration, timestamp, nil
	}

	// 3. 数据库也无记录，返回默认值
	return 0, 0, defaultTime, nil
}

// queryProgressFromRedis 从 Redis 缓存获取设备播放进度
// 键名格式为 device:progress:{sn}
// 参数 sn string: 设备序列号
// 返回 int: 当前播放位置秒数
// 返回 int: 总时长秒数
// 返回 string: 进度记录时间戳
// 返回 error: 查询错误
func (l *DevicePlaybackProgressLogic) queryProgressFromRedis(sn string) (int, int, string, error) {
	key := fmt.Sprintf("device:progress:%s", sn)

	result, err := l.svcCtx.Redis.HGetAll(l.ctx, key).Result()
	if err != nil || len(result) == 0 {
		return 0, 0, "", fmt.Errorf("Redis无数据")
	}

	var currentTime, duration int
	var timestamp string

	if v, ok := result["current_time"]; ok && v != "" {
		fmt.Sscanf(v, "%d", &currentTime)
	}
	if v, ok := result["duration"]; ok && v != "" {
		fmt.Sscanf(v, "%d", &duration)
	}
	if v, ok := result["timestamp"]; ok && v != "" {
		timestamp = v
	} else {
		timestamp = time.Now().Format(time.RFC3339)
	}

	return currentTime, duration, timestamp, nil
}

// queryProgressFromMySQL 从 MySQL 数据库查询设备播放进度
// 从 device_playback_status 表按 last_update 倒序获取最新记录
// 参数 sn string: 设备序列号
// 返回 int: 当前播放位置秒数
// 返回 int: 总时长秒数
// 返回 string: 进度记录时间戳
// 返回 error: 查询错误
func (l *DevicePlaybackProgressLogic) queryProgressFromMySQL(sn string) (int, int, string, error) {
	query := `
		SELECT current_time, duration, last_update 
		FROM device_playback_status 
		WHERE sn = $1 
		ORDER BY last_update DESC 
		LIMIT 1
	`

	var currentTime, duration int
	var lastUpdate sql.NullTime

	err := l.svcCtx.DB.QueryRowContext(l.ctx, query, sn).Scan(&currentTime, &duration, &lastUpdate)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, 0, "", fmt.Errorf("无播放进度记录")
		}
		return 0, 0, "", fmt.Errorf("MySQL查询失败: %v", err)
	}

	timestamp := time.Now().Format(time.RFC3339)
	if lastUpdate.Valid {
		timestamp = lastUpdate.Time.Format(time.RFC3339)
	}

	return currentTime, duration, timestamp, nil
}

// cacheProgressToRedis 将播放进度缓存到 Redis
// 键名格式为 device:progress:{sn}，使用 Hash 结构存储
// 设置缓存过期时间为 5 分钟
// 参数 sn string: 设备序列号
// 参数 currentTime int: 当前播放位置秒数
// 参数 duration int: 总时长秒数
func (l *DevicePlaybackProgressLogic) cacheProgressToRedis(sn string, currentTime, duration int) {
	key := fmt.Sprintf("device:progress:%s", sn)

	fields := map[string]interface{}{
		"current_time": currentTime,
		"duration":     duration,
		"timestamp":    time.Now().Format(time.RFC3339),
	}

	err := l.svcCtx.Redis.HSet(l.ctx, key, fields).Err()
	if err != nil {
		logx.Errorf("Redis缓存播放进度失败: %v", err)
		return
	}

	err = l.svcCtx.Redis.Expire(l.ctx, key, 5*time.Minute).Err()
	if err != nil {
		logx.Errorf("设置Redis过期时间失败: %v", err)
	}
}

// calculatePercentage 计算进度百分比
// 公式：percentage = (current_time / duration) * 100
// 如果 duration 为 0，则 percentage 为 0
// 保留一位小数点精度
// 参数 currentTime int: 当前播放位置秒数
// 参数 duration int: 总时长秒数
// 返回 float64: 进度百分比
func calculatePercentage(currentTime, duration int) float64 {
	if duration <= 0 {
		return 0
	}

	percentage := float64(currentTime) / float64(duration) * 100
	return math.Round(percentage*10) / 10
}

// calculateRemainingTime 计算剩余时间
// 公式：remaining_time = duration - current_time
// 参数 currentTime int: 当前播放位置秒数
// 参数 duration int: 总时长秒数
// 返回 int: 剩余时间秒数
func calculateRemainingTime(currentTime, duration int) int {
	if duration <= 0 {
		return 0
	}

	remaining := duration - currentTime
	if remaining < 0 {
		return 0
	}

	return remaining
}

// validateDevicePlaybackProgressReq 校验播放进度查询请求参数
// 校验 SN 格式是否为 16 位字母数字组合
// 参数 req *types.DevicePlaybackProgressReq: 播放进度查询请求
// 返回 error: 校验错误
func validateDevicePlaybackProgressReq(req *types.DevicePlaybackProgressReq) error {
	if req == nil {
		return fmt.Errorf("请求不能为空")
	}

	sn := strings.TrimSpace(req.Sn)
	if sn == "" {
		return fmt.Errorf("设备序列号不能为空")
	}

	snRegex := regexp.MustCompile(`^[A-Za-z0-9]{16}$`)
	if !snRegex.MatchString(sn) {
		return fmt.Errorf("设备序列号格式错误，必须为16位字母数字组合")
	}

	return nil
}
