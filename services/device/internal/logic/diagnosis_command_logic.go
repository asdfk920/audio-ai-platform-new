package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

// DiagnosisCommandLogic 诊断指令下发逻辑
// 处理用户通过App/后台发起设备日志收集等诊断指令的业务逻辑
type DiagnosisCommandLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDiagnosisCommandLogic 创建诊断指令逻辑实例
func NewDiagnosisCommandLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DiagnosisCommandLogic {
	return &DiagnosisCommandLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// SendDiagnosisCommand 发送诊断指令
// 流程：
//  1. 校验请求数据格式（device_sn、cmd_action等）
//  2. 查询设备是否存在并验证用户绑定权限
//  3. 检查设备在线状态（必须在线才能下发）
//  4. 生成全局唯一trace_id（UUID格式）
//  5. 设置默认参数值（log_type、max_lines、timeout等）
//  6. 构造标准诊断指令载荷（符合纯文本协议规范）
//  7. 通过WebSocket网关下发指令给设备
//  8. 记录指令状态和超时时间
//  9. 返回指令下发响应（包含trace_id用于追踪）
//
// 参数 req *types.DiagnosisCommandReq: 诊断指令请求
// 参数 userID int64: 用户ID
// 返回 *types.DiagnosisCommandResp: 诊断指令响应
// 返回 error: 下发失败时的错误信息
func (l *DiagnosisCommandLogic) SendDiagnosisCommand(req *types.DiagnosisCommandReq, userID int64) (*types.DiagnosisCommandResp, error) {
	deviceSN := strings.TrimSpace(req.DeviceSN)
	if deviceSN == "" {
		return nil, fmt.Errorf("设备序列号不能为空")
	}

	cmdAction := strings.TrimSpace(req.CmdAction)
	if cmdAction == "" {
		return nil, fmt.Errorf("指令动作不能为空")
	}

	if cmdAction != "collect_log" {
		return nil, fmt.Errorf("不支持的指令动作: %s（目前仅支持 collect_log）", cmdAction)
	}

	logx.Infof("[DiagnosisCmd] 用户 %d 请求发送诊断指令: device_sn=%s, cmd_action=%s", userID, deviceSN, cmdAction)

	device, err := l.svcCtx.DeviceRepo.FindBySn(l.ctx, deviceSN)
	if err != nil {
		logx.Errorf("[DiagnosisCmd] 设备查询失败: sn=%s, err=%v", deviceSN, err)
		return nil, fmt.Errorf("设备不存在或查询失败")
	}

	bound, err := l.svcCtx.UserDeviceBindRepo.ExistsActiveByUserAndSN(l.ctx, userID, deviceSN)
	if err != nil {
		logx.Errorf("[DiagnosisCmd] 权限校验失败: user_id=%d, device_sn=%s, err=%v", userID, deviceSN, err)
		return nil, fmt.Errorf("权限校验失败")
	}
	if !bound {
		return nil, fmt.Errorf("无权操作该设备")
	}

	traceID := uuid.New().String()
	now := time.Now()

	logType := req.LogType
	if logType == "" {
		logType = "all"
	}
	validLogTypes := map[string]bool{"all": true, "system": true, "app": true, "error": true}
	if !validLogTypes[logType] {
		return nil, fmt.Errorf("不支持的日志类型: %s（支持: all/system/app/error）", logType)
	}

	maxLines := req.MaxLines
	if maxLines <= 0 {
		maxLines = 1000
	}
	if maxLines > 10000 {
		maxLines = 10000
	}

	timeoutSec := req.TimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 60
	}

	priority := strings.ToLower(req.Priority)
	if priority == "" {
		priority = "normal"
	}
	if priority != "high" {
		priority = "normal"
	}

	params := map[string]interface{}{
		"log_type":      logType,
		"max_lines":     maxLines,
		"compress":      req.Compress,
		"callback_flag": req.CallbackFlag,
	}

	if req.TimeRange != nil {
		params["time_range"] = map[string]interface{}{
			"start": req.TimeRange.Start,
			"end":   req.TimeRange.End,
		}
	}

	payload := types.DiagnosisCommandDevicePayload{
		TraceID:   traceID,
		CmdType:   "diagnosis",
		CmdAction: cmdAction,
		DeviceSN:  deviceSN,
		SendTime:  now.UnixMilli(),
		Priority:  priority,
		Params:    params,
	}

	deviceIDStr := fmt.Sprintf("%d", device.ID)

	logx.Infof("[DiagnosisCmd] 准备下发诊断指令: trace_id=%s, device_sn=%s, device_id=%s, priority=%s",
		traceID, deviceSN, deviceIDStr, priority)

	logx.Infof("[DiagnosisCmd] 诊断指令载荷: %s", mustJsonIndent(payload))

	isDeviceOnline := device.OnlineStatus == 1

	if isDeviceOnline {
		if l.svcCtx.WsDeliver != nil {
			result, err := l.svcCtx.WsDeliver(deviceIDStr, deviceSN, payload)
			if err != nil {
				logx.Errorf("[DiagnosisCmd] WebSocket下发失败: trace_id=%s, device_id=%s, err=%v",
					traceID, deviceIDStr, err)
				return nil, fmt.Errorf("指令下发失败: %v", err)
			}
			if result.LocalWriteOk || result.RelayPublishOk {
				logx.Infof("[DiagnosisCmd] WebSocket下发成功: trace_id=%s, device_id=%s, local=%v, relay=%v",
					traceID, deviceIDStr, result.LocalWriteOk, result.RelayPublishOk)
			} else {
				logx.Slowf("[DiagnosisCmd] WebSocket下发未成功，尝试缓存: trace_id=%s, device_id=%s", traceID, deviceIDStr)
				isDeviceOnline = false
			}
		} else if l.svcCtx.WsPushJSON != nil {
			err := l.svcCtx.WsPushJSON(deviceIDStr, payload)
			if err != nil {
				logx.Errorf("[DiagnosisCmd] WsPushJSON下发失败: trace_id=%s, device_id=%s, err=%v",
					traceID, deviceIDStr, err)
				return nil, fmt.Errorf("指令下发失败: %v", err)
			}
			logx.Infof("[DiagnosisCmd] WsPushJSON下发成功: trace_id=%s, device_id=%s", traceID, deviceIDStr)
		} else {
			logx.Errorf("[DiagnosisCmd] 无可用的WebSocket推送通道")
			isDeviceOnline = false
		}
	}

	var status string
	var message string

	if isDeviceOnline {
		status = "sent"
		message = "诊断指令已成功下发，请等待设备执行结果"
	} else {
		cache := GetDiagnosisCache(l.svcCtx)
		if cacheErr := cache.CacheCommand(deviceSN, &payload); cacheErr != nil {
			logx.Errorf("[DiagnosisCmd] 缓存诊断指令失败: trace_id=%s, device_sn=%s, err=%v",
				traceID, deviceSN, cacheErr)
			return nil, fmt.Errorf("设备离线且指令缓存失败: %v", cacheErr)
		}
		status = "cached"
		message = fmt.Sprintf("设备当前离线，诊断指令已缓存（trace_id=%s），将在设备上线后自动下发", traceID)
		logx.Infof("[DiagnosisCmd] ✅ 设备离线，指令已缓存: trace_id=%s, device_sn=%s, 将在上线时自动发送",
			traceID, deviceSN)
	}

	resp := &types.DiagnosisCommandResp{
		TraceID:   traceID,
		DeviceSN:  deviceSN,
		Status:    status,
		Message:   message,
		SendTime:  now.UnixMilli(),
		ExpiresAt: now.Add(time.Duration(timeoutSec) * time.Second).UnixMilli(),
	}

	logx.Infof("[DiagnosisCmd] 诊断指令处理完成: trace_id=%s, device_sn=%s, status=%s, expires_at=%d",
		traceID, deviceSN, status, resp.ExpiresAt)

	return resp, nil
}

// mustJsonIndent 将对象格式化为JSON字符串（用于日志输出）
func mustJsonIndent(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%+v", v)
	}
	return string(b)
}
