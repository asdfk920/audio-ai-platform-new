package logic

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/services/device/internal/device/shadow"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

// DeviceShadowQueryLogic 设备影子查询逻辑
// 处理用户查询指定设备最新状态数据的业务逻辑
type DeviceShadowQueryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDeviceShadowQueryLogic 创建设备影子查询逻辑实例
// 参数 ctx context.Context: 请求上下文
// 参数 svcCtx *svc.ServiceContext: 服务上下文
// 返回 *DeviceShadowQueryLogic: 设备影子查询逻辑实例
func NewDeviceShadowQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceShadowQueryLogic {
	return &DeviceShadowQueryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeviceShadowQuery 查询设备影子最新状态
// 流程：
//  1. 从 URL Query 参数中获取 sn
//  2. 校验 sn 参数格式
//  3. 从 JWT token 中获取用户 ID
//  4. 验证用户是否有权限查询该设备（查询绑定关系）
//  5. 优先查询 Redis 缓存
//  6. Redis 未命中则查询 MySQL 数据库
//  7. MySQL 有数据则回种 Redis 缓存
//  8. 组装响应数据并返回
//
// 参数 sn string: 设备序列号
// 返回 *types.DeviceShadowQueryResp: 设备影子查询响应
// 返回 error: 查询失败时的错误信息
func (l *DeviceShadowQueryLogic) DeviceShadowQuery(sn string) (*types.DeviceShadowQueryResp, error) {
	// 1. 校验 sn 参数格式
	if err := validateShadowQuerySn(sn); err != nil {
		return nil, fmt.Errorf("参数校验失败: %v", err)
	}

	sn = strings.ToUpper(strings.TrimSpace(sn))

	// 2. 从 JWT token 中获取用户 ID
	userID, ok := jwt.GetUserIdFromContext(l.ctx)
	if !ok || userID <= 0 {
		return nil, fmt.Errorf("请先登录")
	}

	// 3. 验证用户是否有权限查询该设备
	deviceInfo, err := l.svcCtx.DeviceRegister.FindBySn(l.ctx, sn)
	if err != nil {
		return nil, fmt.Errorf("查询设备失败: %v", err)
	}
	if deviceInfo == nil {
		return nil, fmt.Errorf("设备未注册")
	}

	bindInfo, err := l.svcCtx.UserDeviceBindRepo.FindByUserIdAndDeviceId(l.ctx, userID, deviceInfo.ID)
	if err != nil {
		return nil, fmt.Errorf("查询绑定关系失败: %v", err)
	}
	if bindInfo == nil {
		return nil, fmt.Errorf("无权限访问该设备")
	}

	// 4. 优先查询 Redis 缓存
	shadowData, fromRedis, err := l.queryFromRedis(sn)
	if err != nil {
		logx.Errorf("查询 Redis 设备影子失败: sn=%s, err=%v", sn, err)
	}

	// 5. Redis 未命中则查询 MySQL
	if !fromRedis {
		shadowData, err = l.queryFromMySQL(sn, deviceInfo.ID)
		if err != nil {
			logx.Errorf("查询 MySQL 设备状态失败: sn=%s, err=%v", sn, err)
		}
	}

	// 6. 组装响应数据
	resp := l.buildResponse(sn, shadowData)

	logx.Infof("设备影子查询成功: user_id=%d, sn=%s", userID, sn)

	return resp, nil
}

// queryFromRedis 从 Redis 查询设备影子
// 返回 shadowData map[string]string, fromRedis bool, err error
func (l *DeviceShadowQueryLogic) queryFromRedis(sn string) (map[string]string, bool, error) {
	rdb := l.svcCtx.Redis
	if rdb == nil {
		return nil, false, nil
	}

	sk := shadow.ShadowKey(sn)
	result, err := rdb.HGetAll(l.ctx, sk).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, err
	}

	if len(result) == 0 {
		return nil, false, nil
	}

	return result, true, nil
}

// queryFromMySQL 从 MySQL 查询设备最新状态
// 返回 shadowData map[string]string, err error
func (l *DeviceShadowQueryLogic) queryFromMySQL(sn string, deviceID int64) (map[string]string, error) {
	if l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未就绪")
	}

	query := `
		SELECT battery, volume, online_status, network, ip, storage_used, storage_total, speaker_count,
		       uwb_x, uwb_y, uwb_z, acoustic_calibrated, acoustic_offset, created_at
		FROM device_state_log
		WHERE device_id = $1 AND sn = $2
		ORDER BY created_at DESC
		LIMIT 1
	`

	var battery int
	var volume int
	var onlineStatus int16
	var network string
	var ip string
	var storageUsed, storageTotal int64
	var speakerCount int
	var uwbX, uwbY, uwbZ sql.NullFloat64
	var acousticCal sql.NullInt16
	var acousticOff sql.NullFloat64
	var createdAt time.Time

	err := l.svcCtx.DB.QueryRowContext(l.ctx, query, deviceID, sn).Scan(
		&battery, &volume, &onlineStatus, &network, &ip,
		&storageUsed, &storageTotal, &speakerCount,
		&uwbX, &uwbY, &uwbZ, &acousticCal, &acousticOff, &createdAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询设备状态日志失败: %v", err)
	}

	data := map[string]string{
		shadow.FBattery:      strconv.Itoa(battery),
		shadow.FRunState:     "normal",
		shadow.FLastActiveMs: strconv.FormatInt(createdAt.UnixMilli(), 10),
		shadow.FUpdatedMs:    strconv.FormatInt(createdAt.UnixMilli(), 10),
		"volume":             strconv.Itoa(volume),
		"work_mode":          network,
		shadow.FIP:           ip,
		"storage_used":       strconv.FormatInt(storageUsed, 10),
		"storage_total":      strconv.FormatInt(storageTotal, 10),
		"speaker_count":      strconv.Itoa(speakerCount),
		"last_report_time":   createdAt.Format(time.RFC3339),
	}

	if uwbX.Valid {
		data["uwb_x"] = fmt.Sprintf("%.2f", uwbX.Float64)
	}
	if uwbY.Valid {
		data["uwb_y"] = fmt.Sprintf("%.2f", uwbY.Float64)
	}
	if uwbZ.Valid {
		data["uwb_z"] = fmt.Sprintf("%.2f", uwbZ.Float64)
	}

	if acousticCal.Valid {
		data["acoustic_calibrated"] = strconv.Itoa(int(acousticCal.Int16))
	}
	if acousticOff.Valid {
		data["acoustic_offset"] = fmt.Sprintf("%.4f", acousticOff.Float64)
	}

	if onlineStatus == 1 {
		data[shadow.FOnline] = "1"
	} else {
		data[shadow.FOnline] = "0"
	}

	// 回种 Redis 缓存
	l.seedRedis(sn, data)

	return data, nil
}

// seedRedis 将 MySQL 查询结果回种到 Redis
func (l *DeviceShadowQueryLogic) seedRedis(sn string, data map[string]string) {
	rdb := l.svcCtx.Redis
	if rdb == nil || len(data) == 0 {
		return
	}

	ttl := 300 * time.Second
	sk := shadow.ShadowKey(sn)

	fields := make(map[string]interface{})
	for k, v := range data {
		fields[k] = v
	}

	pipe := rdb.Pipeline()
	pipe.HSet(l.ctx, sk, fields)
	pipe.Expire(l.ctx, sk, ttl)

	if _, err := pipe.Exec(l.ctx); err != nil {
		logx.Errorf("回种 Redis 缓存失败: sn=%s, err=%v", sn, err)
	}
}

// buildResponse 组装响应数据
func (l *DeviceShadowQueryLogic) buildResponse(sn string, data map[string]string) *types.DeviceShadowQueryResp {
	resp := &types.DeviceShadowQueryResp{
		Sn: sn,
	}

	if data == nil {
		resp.Online = "offline"
		resp.RunState = "unknown"
		return resp
	}

	// 在线状态
	if data[shadow.FOnline] == "1" {
		resp.Online = "online"
	} else {
		resp.Online = "offline"
	}

	// 固件版本
	resp.FirmwareVersion = data[shadow.FFirmwareVersion]

	// 电量
	if v, err := strconv.Atoi(data[shadow.FBattery]); err == nil {
		resp.Battery = v
	}

	// 音量
	if v, err := strconv.Atoi(data["volume"]); err == nil {
		resp.Volume = v
	}

	// 工作模式
	resp.WorkMode = data["work_mode"]

	// 位置信息
	resp.Position = data["location"]

	// 扬声器数量
	if v, err := strconv.Atoi(data["speaker_count"]); err == nil {
		resp.SpeakerCount = v
	}

	// 存储空间
	if v, err := strconv.ParseInt(data["storage_used"], 10, 64); err == nil {
		resp.StorageUsed = v
	}
	if v, err := strconv.ParseInt(data["storage_total"], 10, 64); err == nil {
		resp.StorageTotal = v
	}

	// UWB定位数据
	resp.UWB = parseUWBFromData(data)

	// 声学校准参数
	resp.Acoustic = parseAcousticFromData(data)

	// 运行状态
	resp.RunState = data[shadow.FRunState]
	if resp.RunState == "" {
		resp.RunState = "unknown"
	}

	// 最后上报时间
	resp.LastReportTime = data["last_report_time"]
	if resp.LastReportTime == "" {
		if ms, err := strconv.ParseInt(data[shadow.FUpdatedMs], 10, 64); err == nil && ms > 0 {
			resp.LastReportTime = time.UnixMilli(ms).Format(time.RFC3339)
		}
	}

	return resp
}

// parseUWBFromData 从 Redis/MySQL 数据中解析 UWB 定位信息
func parseUWBFromData(data map[string]string) types.UWBPosition {
	var uwb types.UWBPosition

	if v, ok := data["uwb_x"]; ok && v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			uwb.X = &f
		}
	}
	if v, ok := data["uwb_y"]; ok && v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			uwb.Y = &f
		}
	}
	if v, ok := data["uwb_z"]; ok && v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			uwb.Z = &f
		}
	}

	return uwb
}

// parseAcousticFromData 从 Redis/MySQL 数据中解析声学校准参数
func parseAcousticFromData(data map[string]string) types.AcousticCalib {
	var acoustic types.AcousticCalib

	if v, ok := data["acoustic_calibrated"]; ok && v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			acoustic.Calibrated = &i
		}
	}
	if v, ok := data["acoustic_offset"]; ok && v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			acoustic.Offset = &f
		}
	}

	return acoustic
}

// validateShadowQuerySn 校验设备序列号格式
// 校验规则：16 位字母数字，正则 ^[A-Z0-9]{16}$，不区分大小写
// 参数 sn string: 设备序列号
// 返回 error: 校验失败时的错误信息
func validateShadowQuerySn(sn string) error {
	if strings.TrimSpace(sn) == "" {
		return fmt.Errorf("sn 参数不能为空")
	}

	snRegex := regexp.MustCompile(`(?i)^[A-Z0-9]{16}$`)
	if !snRegex.MatchString(sn) {
		return fmt.Errorf("SN 格式错误，必须为 16 位字母数字组合")
	}

	return nil
}
