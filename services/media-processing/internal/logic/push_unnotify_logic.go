package logic

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/streamutil"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/types"
)

type PushUnnotifyLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPushUnnotifyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PushUnnotifyLogic {
	return &PushUnnotifyLogic{ctx: ctx, svcCtx: svcCtx}
}

func (l *PushUnnotifyLogic) PushUnnotify(req *types.PushUnnotifyReq) (*types.PushUnnotifyResp, error) {
	// 参数校验
	req.StreamKey = streamutil.TrimSpace(req.StreamKey)
	req.ChannelID = streamutil.TrimSpace(req.ChannelID)
	req.ClientIP = streamutil.TrimSpace(req.ClientIP)
	req.ServerIP = streamutil.TrimSpace(req.ServerIP)
	req.Reason = streamutil.TrimSpace(req.Reason)

	if req.StreamKey == "" || req.ChannelID == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "参数错误")
	}

	// 查询推流记录
	pushRecord, err := l.getPushingRecord(req.StreamKey)
	if err != nil {
		return nil, errorx.NewCodeError(errorx.CodeResourceNotFound, "推流记录不存在")
	}

	// 确定停止原因
	stopReason := l.determineStopReason(req.Reason)

	// 计算推流时长
	now := time.Now()
	duration := now.Sub(pushRecord.PushStart).Seconds()
	if req.Duration > 0 {
		// 使用回调携带的时长（一致性校验）
		duration = float64(req.Duration)
	}

	// 计算流量统计
	trafficMB := float64(req.BytesSent) / 1024.0 / 1024.0
	avgBitrate := float64(req.BytesSent) * 8 / duration / 1000 // kbps

	// 确定状态码（1 正常停止，2 异常断开）
	status := 1 // 正常停止
	if stopReason == "network_error" || stopReason == "timeout" || stopReason == "server_maintenance" {
		status = 2 // 异常断开
	}

	// 更新推流记录
	if err := l.updatePushRecord(pushRecord.PushID, status, now, int64(duration), req.BytesSent); err != nil {
		fmt.Printf("更新推流记录失败：%v\n", err)
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "更新推流记录失败")
	}

	// 更新通道状态
	if err := l.updateChannelStatus(req.ChannelID, now); err != nil {
		fmt.Printf("更新通道状态失败：%v\n", err)
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "更新通道状态失败")
	}

	// 更新用户统计
	if err := l.updateUserStats(pushRecord.UserID, duration, req.BytesSent, now); err != nil {
		fmt.Printf("更新用户统计失败：%v\n", err)
	}

	// 清理 Redis 缓存
	if err := l.cleanRedisCache(req.StreamKey); err != nil {
		fmt.Printf("清理 Redis 缓存失败：%v\n", err)
	}

	// 通知下游服务
	notificationSent := false
	if err := l.notifyDownstreamServices(pushRecord, req, duration, trafficMB, avgBitrate, stopReason); err != nil {
		fmt.Printf("通知下游服务失败：%v\n", err)
		// 不返回错误，通知失败不影响主流程
	} else {
		notificationSent = true
	}

	// 记录操作日志
	if err := l.logStreamEvent(pushRecord.UserID, req.ChannelID, req.StreamKey, req.ClientIP, "publish_stop", stopReason, int64(duration), req.BytesSent); err != nil {
		fmt.Printf("记录日志失败：%v\n", err)
	}

	return &types.PushUnnotifyResp{
		Code:             0,
		Message:          "处理成功",
		ChannelID:        req.ChannelID,
		Processed:        true,
		Duration:         int64(duration),
		TrafficMB:        trafficMB,
		NotificationSent: notificationSent,
		Recorded:         true, // 默认已录制
	}, nil
}

// pushRecord 推流记录结构
type pushRecord struct {
	PushID     string
	ChannelID  string
	StreamKey  string
	SourceType string
	SourceID   string
	UserID     int64
	PushStart  time.Time
	Status     int // 0 推流中，1 正常停止，2 异常断开
}

// getPushingRecord 获取正在推流中的记录
func (l *PushUnnotifyLogic) getPushingRecord(streamKey string) (*pushRecord, error) {
	const q = `SELECT push_id, channel_id, stream_key, source_type, source_id, user_id, push_start, status 
	FROM public.stream_push_records 
	WHERE stream_key=$1 AND status=0 
	ORDER BY push_start DESC 
	LIMIT 1`

	var pr pushRecord
	err := l.svcCtx.DB.QueryRowContext(l.ctx, q, streamKey).Scan(
		&pr.PushID, &pr.ChannelID, &pr.StreamKey, &pr.SourceType, &pr.SourceID,
		&pr.UserID, &pr.PushStart, &pr.Status,
	)
	if err == sql.ErrNoRows {
		return nil, errorx.NewCodeError(errorx.CodeResourceNotFound, "推流记录不存在")
	}
	if err != nil {
		return nil, err
	}
	return &pr, nil
}

// determineStopReason 确定停止原因
func (l *PushUnnotifyLogic) determineStopReason(reason string) string {
	switch reason {
	case "client_disconnect", "client_disconnect_normal":
		return "client_disconnect" // 客户端主动断开
	case "network_error", "network_timeout":
		return "network_error" // 网络异常
	case "timeout", "max_duration_exceeded":
		return "timeout" // 超时
	case "manual", "admin_stop":
		return "manual" // 手动停止
	case "server_maintenance", "server_restart":
		return "server_maintenance" // 服务器维护
	default:
		return "unknown" // 未知原因
	}
}

// updatePushRecord 更新推流记录
func (l *PushUnnotifyLogic) updatePushRecord(pushID string, status int, endTime time.Time, duration, bytesSent int64) error {
	const q = `UPDATE public.stream_push_records 
	SET status=$2, push_end=$3, push_duration=$4, bytes_sent=$5, updated_at=NOW() 
	WHERE push_id=$1`

	_, err := l.svcCtx.DB.ExecContext(l.ctx, q, pushID, status, endTime, duration, bytesSent)
	return err
}

// updateChannelStatus 更新通道状态
func (l *PushUnnotifyLogic) updateChannelStatus(channelID string, endTime time.Time) error {
	const q = `UPDATE public.stream_channels 
	SET push_status=0, push_end_time=$2, updated_at=NOW() 
	WHERE channel_id=$1`

	_, err := l.svcCtx.DB.ExecContext(l.ctx, q, channelID, endTime)
	return err
}

// updateUserStats 更新用户统计
func (l *PushUnnotifyLogic) updateUserStats(userID int64, duration float64, bytesSent int64, now time.Time) error {
	// TODO: 更新用户统计表
	// UPDATE user_statistics SET 
	// total_push_duration = total_push_duration + $1,
	// total_traffic = total_traffic + $2,
	// last_push_time = $3
	// WHERE user_id = $4
	return nil
}

// cleanRedisCache 清理 Redis 缓存
func (l *PushUnnotifyLogic) cleanRedisCache(streamKey string) error {
	// 删除 Token 缓存
	tokenKey := streamutil.RedisPushTokenKey(streamKey)
	_ = redisx.Del(l.ctx, tokenKey)

	// 删除统计缓存
	statsKey := "push:stats:" + streamKey
	_ = redisx.Del(l.ctx, statsKey)

	return nil
}

// notifyDownstreamServices 通知下游服务
func (l *PushUnnotifyLogic) notifyDownstreamServices(record *pushRecord, req *types.PushUnnotifyReq, duration, trafficMB, avgBitrate float64, stopReason string) error {
	// TODO: 使用 Kafka 发送通知消息
	// Topic: stream-push-events
	// Message: JSON 格式包含 event_type, channel_id, stream_key, user_id 等
	
	event := map[string]any{
		"event_type":     "publish_stop",
		"channel_id":     record.ChannelID,
		"stream_key":     record.StreamKey,
		"user_id":        record.UserID,
		"source_type":    record.SourceType,
		"source_id":      record.SourceID,
		"push_start":     record.PushStart.Unix(),
		"push_end":       time.Now().Unix(),
		"duration":       int64(duration),
		"bytes_sent":     req.BytesSent,
		"traffic_mb":     trafficMB,
		"avg_bitrate":    avgBitrate,
		"stop_reason":    stopReason,
		"client_ip":      req.ClientIP,
		"server_ip":      req.ServerIP,
		"timestamp":      time.Now().Unix(),
	}

	// 打印日志（替代 Kafka 发送）
	eventJSON, _ := json.Marshal(event)
	fmt.Printf("通知下游服务：%s\n", string(eventJSON))
	
	return nil
}

// logStreamEvent 记录流媒体事件日志
func (l *PushUnnotifyLogic) logStreamEvent(userID int64, channelID, streamKey, clientIP, eventType, stopReason string, duration, bytesSent int64) error {
	const q = `INSERT INTO public.stream_logs 
	(log_type, user_id, channel_id, stream_key, client_ip, event_type, stop_reason, duration, bytes_sent, event_time, created_at) 
	VALUES ('push', $1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())`

	_, err := l.svcCtx.DB.ExecContext(l.ctx, q, userID, channelID, streamKey, clientIP, eventType, stopReason, duration, bytesSent)
	return err
}
