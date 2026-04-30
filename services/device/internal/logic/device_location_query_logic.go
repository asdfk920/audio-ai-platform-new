package logic

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/services/device/internal/device/shadow"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

// DeviceLocationQueryLogic 设备位置查询逻辑
// 处理用户查询指定设备最新 UWB 定位数据的业务逻辑
type DeviceLocationQueryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDeviceLocationQueryLogic 创建设备位置查询逻辑实例
func NewDeviceLocationQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceLocationQueryLogic {
	return &DeviceLocationQueryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeviceLocationQuery 查询设备最新位置
// 流程：
//  1. 从 URL Query 参数中获取 sn
//  2. 校验 sn 参数格式
//  3. 从 JWT token 中获取用户 ID
//  4. 验证用户是否有权限查询该设备（查询绑定关系）
//  5. 查询设备在线状态
//  6. 优先查询 Redis 缓存获取 UWB 定位数据
//  7. Redis 未命中则查询 MySQL 数据库
//  8. 组装响应数据并返回
//
// 参数 sn string: 设备序列号
// 返回 *types.DeviceLocationResp: 设备位置查询响应
// 返回 error: 查询失败时的错误信息
func (l *DeviceLocationQueryLogic) DeviceLocationQuery(sn string) (*types.DeviceLocationResp, error) {
	// 1. 校验 sn 参数格式
	if err := validateLocationQuerySn(sn); err != nil {
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

	// 4. 查询设备在线状态
	online := l.getOnlineStatus(sn)

	// 5. 优先查询 Redis 缓存获取 UWB 定位数据
	locationData, fromRedis, err := l.queryLocationFromRedis(sn)
	if err != nil {
		logx.Errorf("查询 Redis 设备位置失败: sn=%s, err=%v", sn, err)
	}

	// 6. Redis 未命中则查询 MySQL
	if !fromRedis {
		locationData, err = l.queryLocationFromMySQL(sn, deviceInfo.ID)
		if err != nil {
			logx.Errorf("查询 MySQL 设备位置失败: sn=%s, err=%v", sn, err)
		}
	}

	// 7. 组装响应数据
	resp := l.buildLocationResponse(sn, online, locationData)

	logx.Infof("设备位置查询成功: user_id=%d, sn=%s, online=%s", userID, sn, online)

	return resp, nil
}

// getOnlineStatus 查询设备在线状态
func (l *DeviceLocationQueryLogic) getOnlineStatus(sn string) string {
	rdb := l.svcCtx.Redis
	if rdb == nil {
		return "offline"
	}

	okKey := shadow.OnlineKey(sn)
	val, err := rdb.Get(l.ctx, okKey).Result()
	if err != nil {
		if err == redis.Nil {
			return "offline"
		}
		logx.Errorf("查询设备在线状态失败: sn=%s, err=%v", sn, err)
		return "offline"
	}

	if val == "1" {
		return "online"
	}
	return "offline"
}

// queryLocationFromRedis 从 Redis 查询设备 UWB 定位数据
func (l *DeviceLocationQueryLogic) queryLocationFromRedis(sn string) (map[string]string, bool, error) {
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

	// 检查是否有 UWB 数据
	hasUWB := false
	if _, ok := result["uwb_x"]; ok && result["uwb_x"] != "" {
		hasUWB = true
	}
	if _, ok := result["uwb_y"]; ok && result["uwb_y"] != "" {
		hasUWB = true
	}

	if !hasUWB {
		return nil, false, nil
	}

	return result, true, nil
}

// queryLocationFromMySQL 从 MySQL 查询设备最新 UWB 定位数据
func (l *DeviceLocationQueryLogic) queryLocationFromMySQL(sn string, deviceID int64) (map[string]string, error) {
	if l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未就绪")
	}

	query := `
		SELECT uwb_x, uwb_y, uwb_z, created_at
		FROM device_state_log
		WHERE device_id = $1 AND sn = $2 AND (uwb_x IS NOT NULL OR uwb_y IS NOT NULL)
		ORDER BY created_at DESC
		LIMIT 1
	`

	var uwbX, uwbY, uwbZ sql.NullFloat64
	var createdAt time.Time

	err := l.svcCtx.DB.QueryRowContext(l.ctx, query, deviceID, sn).Scan(&uwbX, &uwbY, &uwbZ, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询设备位置数据失败: %v", err)
	}

	data := map[string]string{
		"last_report_time": createdAt.Format(time.RFC3339),
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

	// 回种 Redis 缓存
	l.seedLocationRedis(sn, data)

	return data, nil
}

// seedLocationRedis 将 MySQL 查询结果回种到 Redis
func (l *DeviceLocationQueryLogic) seedLocationRedis(sn string, data map[string]string) {
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
		logx.Errorf("回种 Redis 位置缓存失败: sn=%s, err=%v", sn, err)
	}
}

// buildLocationResponse 组装位置响应数据
func (l *DeviceLocationQueryLogic) buildLocationResponse(sn, online string, data map[string]string) *types.DeviceLocationResp {
	resp := &types.DeviceLocationResp{
		Sn:     sn,
		Online: online,
	}

	if data == nil {
		return resp
	}

	// UWB定位数据
	resp.UWB = parseUWBFromData(data)

	// 定位精度（根据 UWB 数据计算，默认 0.1 米）
	resp.Accuracy = 0.1

	// 最后上报时间
	resp.LastReportTime = data["last_report_time"]

	// 判断是否为最新位置（5分钟内上报的视为最新）
	if resp.LastReportTime != "" {
		if t, err := time.Parse(time.RFC3339, resp.LastReportTime); err == nil {
			resp.IsLatest = time.Since(t) < 5*time.Minute
		}
	}

	return resp
}

// validateLocationQuerySn 校验设备序列号格式
func validateLocationQuerySn(sn string) error {
	if strings.TrimSpace(sn) == "" {
		return fmt.Errorf("sn 参数不能为空")
	}

	snRegex := regexp.MustCompile(`(?i)^[A-Z0-9]{16}$`)
	if !snRegex.MatchString(sn) {
		return fmt.Errorf("SN 格式错误，必须为 16 位字母数字组合")
	}

	return nil
}
