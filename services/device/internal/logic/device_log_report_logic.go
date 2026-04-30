package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// DeviceLogReportLogic 设备日志上报逻辑
// 处理设备通过 HTTP POST 请求上报运行日志的业务逻辑
type DeviceLogReportLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDeviceLogReportLogic 创建设备日志上报逻辑实例
// 参数 ctx context.Context: 请求上下文
// 参数 svcCtx *svc.ServiceContext: 服务上下文
// 返回 *DeviceLogReportLogic: 设备日志上报逻辑实例
func NewDeviceLogReportLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceLogReportLogic {
	return &DeviceLogReportLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeviceLogReport 设备日志上报
// 流程：
//  1. 校验请求数据格式（SN、日志类型、级别、内容等）
//  2. 查询设备是否存在且有效
//  3. 生成日志唯一 ID
//  4. 将日志写入 MySQL device_log 表
//  5. 返回接收确认响应
//
// 参数 req *types.DeviceLogReportReq: 设备日志上报请求
// 返回 *types.DeviceLogReportResp: 设备日志上报响应
// 返回 error: 上报失败时的错误信息
func (l *DeviceLogReportLogic) DeviceLogReport(req *types.DeviceLogReportReq) (*types.DeviceLogReportResp, error) {
	// 1. 校验请求数据格式
	if err := validateDeviceLogReportReq(req); err != nil {
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

	// 3. 生成日志唯一 ID
	now := time.Now()
	logID := fmt.Sprintf("log_%d_%s", now.UnixNano(), generateRandomString(8))

	// 4. 将日志写入 MySQL device_log 表
	if err := l.writeDeviceLog(sn, deviceInfo.ID, req, logID, now); err != nil {
		return nil, fmt.Errorf("写入日志失败: %v", err)
	}

	logx.Infof("设备日志上报成功: sn=%s, log_id=%s, log_type=%s, level=%s",
		sn, logID, req.LogType, req.Level)

	// 5. 返回接收确认响应
	return &types.DeviceLogReportResp{
		LogID:     logID,
		Success:   true,
		Message:   "日志接收成功",
		Timestamp: now.Format("2006-01-02T15:04:05Z"),
	}, nil
}

// writeDeviceLog 将日志写入 MySQL device_log 表
func (l *DeviceLogReportLogic) writeDeviceLog(sn string, deviceID int64, req *types.DeviceLogReportReq, logID string, now time.Time) error {
	if l.svcCtx.DB == nil {
		return fmt.Errorf("数据库未就绪")
	}

	// 将 metadata 转换为 JSON
	metadataJSON := ""
	if req.Metadata != nil && len(req.Metadata) > 0 {
		metadataBytes, err := json.Marshal(req.Metadata)
		if err != nil {
			return fmt.Errorf("metadata 序列化失败: %v", err)
		}
		metadataJSON = string(metadataBytes)
	}

	// 处理日志时间戳
	reportTime := now
	if req.Timestamp > 0 {
		reportTime = time.Unix(req.Timestamp, 0)
	}

	// 将 log_type 映射到数据库格式
	logType := strings.ToLower(strings.TrimSpace(req.LogType))
	if logType == "warning" {
		logType = "warn" // 数据库使用 warn 而不是 warning
	}

	// 将 level 映射到数据库格式
	level := strings.ToLower(strings.TrimSpace(req.Level))
	if level == "warning" {
		level = "warn" // 数据库使用 warn 而不是 warning
	}

	query := `
		INSERT INTO device_log (
			device_id, sn, log_type, log_level, module, content, 
			extra, report_time, report_source, ip_address, 
			processed, alert_sent, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, 
			$7::jsonb, $8, $9, $10, 
			0, 0, $11, $11
		)
	`

	_, err := l.svcCtx.DB.ExecContext(l.ctx, query,
		deviceID,     // $1: device_id
		sn,           // $2: sn
		logType,      // $3: log_type
		level,        // $4: log_level
		"",           // $5: module (默认为空)
		req.Content,  // $6: content
		metadataJSON, // $7: extra (JSONB)
		reportTime,   // $8: report_time
		"device",     // $9: report_source
		"",           // $10: ip_address (默认为空)
		now,          // $11: created_at/updated_at
	)
	if err != nil {
		return fmt.Errorf("写入设备日志失败: %v", err)
	}

	return nil
}

// validateDeviceLogReportReq 校验设备日志上报请求数据格式
// 校验规则：
//   - SN: 16 位字母数字，正则 ^[A-Z0-9]{16}$，不区分大小写
//   - LogType: 仅支持 error、warning、info、debug
//   - Level: 仅支持 debug、info、warn、error、fatal
//   - Content: 不能为空
//   - Timestamp: 如果提供，必须是合理的时间戳
//
// 参数 req *types.DeviceLogReportReq: 设备日志上报请求
// 返回 error: 校验失败时的错误信息
func validateDeviceLogReportReq(req *types.DeviceLogReportReq) error {
	snRegex := regexp.MustCompile(`(?i)^[A-Z0-9]{16}$`)
	if !snRegex.MatchString(req.Sn) {
		return fmt.Errorf("SN 格式错误，必须为 16 位字母数字组合")
	}

	// 校验 log_type
	validLogTypes := map[string]bool{
		"error":   true,
		"warning": true,
		"info":    true,
		"debug":   true,
	}
	logType := strings.ToLower(strings.TrimSpace(req.LogType))
	if !validLogTypes[logType] {
		return fmt.Errorf("log_type 无效，仅支持 error、warning、info、debug")
	}

	// 校验 level
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
		"fatal": true,
	}
	level := strings.ToLower(strings.TrimSpace(req.Level))
	if !validLevels[level] {
		return fmt.Errorf("level 无效，仅支持 debug、info、warn、error、fatal")
	}

	// 校验 content
	if strings.TrimSpace(req.Content) == "" {
		return fmt.Errorf("content 不能为空")
	}

	// 校验 timestamp (如果提供)
	if req.Timestamp < 0 {
		return fmt.Errorf("timestamp 不能为负数")
	}

	return nil
}

// generateRandomString 生成指定长度的随机字符串
// 用于生成日志 ID 中的随机部分
// 参数 length int: 字符串长度
// 返回 string: 随机字符串
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
