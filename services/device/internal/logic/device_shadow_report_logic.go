package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/services/device/internal/device/shadow"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// DeviceShadowReportLogic 设备影子定时上报逻辑
// 处理设备定时采集状态数据并通过 HTTP 上报到云端的业务逻辑
type DeviceShadowReportLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDeviceShadowReportLogic 创建设备影子定时上报逻辑实例
// 参数 ctx context.Context: 请求上下文
// 参数 svcCtx *svc.ServiceContext: 服务上下文
// 返回 *DeviceShadowReportLogic: 设备影子定时上报逻辑实例
func NewDeviceShadowReportLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceShadowReportLogic {
	return &DeviceShadowReportLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeviceShadowReport 设备影子定时上报
// 流程：
//  1. 校验请求数据格式（SN、电量等）
//  2. 查询设备是否存在且有效
//  3. 写入 MySQL 设备状态日志表
//  4. 更新 Redis 设备影子 Hash
//  5. 更新 Redis 设备在线状态
//  6. 返回上报结果
//
// 参数 req *types.DeviceShadowReportReq: 设备影子上报请求
// 返回 *types.DeviceShadowReportResp: 设备影子上报响应
// 返回 error: 上报失败时的错误信息
func (l *DeviceShadowReportLogic) DeviceShadowReport(req *types.DeviceShadowReportReq) (*types.DeviceShadowReportResp, error) {
	// 1. 校验请求数据格式
	if err := validateDeviceShadowReportReq(req); err != nil {
		return nil, fmt.Errorf("数据格式校验失败: %v", err)
	}

	sn := strings.ToUpper(strings.TrimSpace(req.Sn))

	// 2. 查询设备是否存在且有效
	deviceInfo, err := l.svcCtx.DeviceRegister.FindBySn(l.ctx, sn)
	if err != nil {
		return nil, fmt.Errorf("查询设备失败: %v", err)
	}
	if deviceInfo == nil {
		return nil, fmt.Errorf("设备未注册")
	}

	now := time.Now()
	if req.Timestamp > 0 {
		now = time.Unix(req.Timestamp, 0)
	}

	// 3. 写入 MySQL 设备状态日志表
	if err := l.writeStatusLog(sn, deviceInfo.ID, req, now); err != nil {
		logx.Errorf("写入设备状态日志失败: sn=%s, err=%v", sn, err)
	}

	// 4. 更新 Redis 设备影子 Hash
	if err := l.updateShadow(sn, deviceInfo.ID, req, now); err != nil {
		logx.Errorf("更新设备影子失败: sn=%s, err=%v", sn, err)
	}

	// 5. 更新 Redis 设备在线状态
	if err := l.updateOnlineStatus(sn); err != nil {
		logx.Errorf("更新设备在线状态失败: sn=%s, err=%v", sn, err)
	}

	logx.Infof("设备影子定时上报成功: sn=%s, battery=%d, run_state=%s, work_mode=%s",
		sn, req.Battery, req.RunState, req.WorkMode)

	// 6. 返回上报结果
	return &types.DeviceShadowReportResp{
		Sn:        sn,
		UpdatedAt: now.Format("2006-01-02T15:04:05Z"),
		Message:   "上报成功",
	}, nil
}

// writeStatusLog 写入 MySQL 设备状态日志表
func (l *DeviceShadowReportLogic) writeStatusLog(sn string, deviceID int64, req *types.DeviceShadowReportReq, now time.Time) error {
	if l.svcCtx.DB == nil {
		return fmt.Errorf("数据库未就绪")
	}

	query := `
		INSERT INTO device_state_log (device_id, sn, battery, volume, online_status, network, ip, storage_used, storage_total, speaker_count, uwb_x, uwb_y, uwb_z, acoustic_calibrated, acoustic_offset, created_at)
		VALUES ($1, $2, $3, $4, 1, $5, '', $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	network := req.WorkMode
	if network == "" {
		network = "unknown"
	}

	var uwbX, uwbY, uwbZ *float64
	if req.UWB.X != nil {
		uwbX = req.UWB.X
	}
	if req.UWB.Y != nil {
		uwbY = req.UWB.Y
	}
	if req.UWB.Z != nil {
		uwbZ = req.UWB.Z
	}

	var acousticCal int16
	if req.Acoustic.Calibrated != nil {
		acousticCal = int16(*req.Acoustic.Calibrated)
	}

	var acousticOff *float64
	if req.Acoustic.Offset != nil {
		acousticOff = req.Acoustic.Offset
	}

	_, err := l.svcCtx.DB.ExecContext(l.ctx, query, deviceID, sn, req.Battery, req.Volume, network,
		req.StorageUsed, req.StorageTotal, req.SpeakerCount, uwbX, uwbY, uwbZ, acousticCal, acousticOff, now)
	if err != nil {
		return fmt.Errorf("写入设备状态日志失败: %v", err)
	}

	return nil
}

// updateShadow 更新 Redis 设备影子 Hash
func (l *DeviceShadowReportLogic) updateShadow(sn string, deviceID int64, req *types.DeviceShadowReportReq, now time.Time) error {
	rdb := l.svcCtx.Redis
	if rdb == nil {
		return fmt.Errorf("Redis 未就绪")
	}

	ttl := time.Duration(l.svcCtx.Config.DeviceShadow.HeartbeatTTLSeconds) * time.Second
	if ttl <= 0 {
		ttl = 300 * time.Second
	}

	sk := shadow.ShadowKey(sn)
	nowMs := now.UnixMilli()

	fields := map[string]interface{}{
		shadow.FOnline:       "1",
		shadow.FSN:           sn,
		shadow.FRunState:     req.RunState,
		shadow.FBattery:      fmt.Sprintf("%d", req.Battery),
		shadow.FLastActiveMs: fmt.Sprintf("%d", nowMs),
		shadow.FUpdatedMs:    fmt.Sprintf("%d", nowMs),
		shadow.FDeviceID:     fmt.Sprintf("%d", deviceID),
	}

	if req.FirmwareVersion != "" {
		fields[shadow.FFirmwareVersion] = req.FirmwareVersion
	}

	if req.WorkMode != "" {
		fields["work_mode"] = req.WorkMode
	}

	if req.Location != "" {
		fields["location"] = req.Location
	}

	if req.Volume > 0 {
		fields["volume"] = fmt.Sprintf("%d", req.Volume)
	}

	if req.StorageUsed > 0 {
		fields["storage_used"] = strconv.FormatInt(req.StorageUsed, 10)
	}

	if req.StorageTotal > 0 {
		fields["storage_total"] = strconv.FormatInt(req.StorageTotal, 10)
	}

	if req.SpeakerCount > 0 {
		fields["speaker_count"] = fmt.Sprintf("%d", req.SpeakerCount)
	}

	if req.UWB.X != nil {
		fields["uwb_x"] = fmt.Sprintf("%.2f", *req.UWB.X)
	}
	if req.UWB.Y != nil {
		fields["uwb_y"] = fmt.Sprintf("%.2f", *req.UWB.Y)
	}
	if req.UWB.Z != nil {
		fields["uwb_z"] = fmt.Sprintf("%.2f", *req.UWB.Z)
	}

	if req.Acoustic.Calibrated != nil {
		fields["acoustic_calibrated"] = fmt.Sprintf("%d", *req.Acoustic.Calibrated)
	}
	if req.Acoustic.Offset != nil {
		fields["acoustic_offset"] = fmt.Sprintf("%.4f", *req.Acoustic.Offset)
	}

	uwbJSON, _ := json.Marshal(req.UWB)
	if len(uwbJSON) > 0 && string(uwbJSON) != "null" {
		fields["uwb_json"] = string(uwbJSON)
	}

	acousticJSON, _ := json.Marshal(req.Acoustic)
	if len(acousticJSON) > 0 && string(acousticJSON) != "null" {
		fields["acoustic_json"] = string(acousticJSON)
	}

	pipe := rdb.Pipeline()
	pipe.HSet(l.ctx, sk, fields)
	pipe.Expire(l.ctx, sk, ttl)

	if _, err := pipe.Exec(l.ctx); err != nil {
		return fmt.Errorf("更新设备影子失败: %v", err)
	}

	return nil
}

// updateOnlineStatus 更新 Redis 设备在线状态
func (l *DeviceShadowReportLogic) updateOnlineStatus(sn string) error {
	rdb := l.svcCtx.Redis
	if rdb == nil {
		return fmt.Errorf("Redis 未就绪")
	}

	ttl := time.Duration(l.svcCtx.Config.DeviceShadow.HeartbeatTTLSeconds) * time.Second
	if ttl <= 0 {
		ttl = 300 * time.Second
	}

	okKey := shadow.OnlineKey(sn)

	pipe := rdb.Pipeline()
	pipe.Set(l.ctx, okKey, "1", ttl)
	if l.svcCtx.Config.DeviceShadow.EnableOnlineSet {
		pipe.SAdd(l.ctx, shadow.KeyOnlineAll, sn)
	}

	if _, err := pipe.Exec(l.ctx); err != nil {
		return fmt.Errorf("更新设备在线状态失败: %v", err)
	}

	return nil
}

// validateDeviceShadowReportReq 校验设备影子上报请求数据格式
// 校验规则：
//   - SN: 16 位字母数字，正则 ^[A-Z0-9]{16}$，不区分大小写
//   - Battery: 电量 0-100
//   - Volume: 音量 0-100
//   - StorageUsed/StorageTotal: 存储不能为负
//   - SpeakerCount: 扬声器数量不能为负
//   - Acoustic.Calibrated: 仅支持 0 或 1
//
// 参数 req *types.DeviceShadowReportReq: 设备影子上报请求
// 返回 error: 校验失败时的错误信息
func validateDeviceShadowReportReq(req *types.DeviceShadowReportReq) error {
	snRegex := regexp.MustCompile(`(?i)^[A-Z0-9]{16}$`)
	if !snRegex.MatchString(req.Sn) {
		return fmt.Errorf("SN 格式错误，必须为 16 位字母数字组合")
	}

	if req.Battery < 0 || req.Battery > 100 {
		return fmt.Errorf("电量值错误，必须在 0-100 之间")
	}

	if req.Volume < 0 || req.Volume > 100 {
		return fmt.Errorf("音量值错误，必须在 0-100 之间")
	}

	if req.StorageUsed < 0 || req.StorageTotal < 0 {
		return fmt.Errorf("存储空间不能为负")
	}

	if req.SpeakerCount < 0 {
		return fmt.Errorf("扬声器数量不能为负")
	}

	if req.Acoustic.Calibrated != nil {
		v := *req.Acoustic.Calibrated
		if v != 0 && v != 1 {
			return fmt.Errorf("声学校准状态仅支持 0 或 1")
		}
	}

	return nil
}
